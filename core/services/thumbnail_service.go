package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/fogleman/fauxgl"
	"github.com/hpinc/go3mf"
)

// Thumbnail generation safety limits. The renderer (fauxgl) is a third-party
// rasterizer that can hang, allocate huge buffers or panic on malformed or
// extreme geometry. These bounds keep a single bad upload from taking the process
// down: a hard triangle cap, a per-render wall-clock timeout, and a process-wide
// concurrency gate so at most maxConcurrentRenders goroutines can ever be pinned
// by a stuck render.
const (
	defaultMaxTriangles      = 2_000_000
	defaultRenderTimeout     = 10 * time.Second
	defaultAcquireTimeout    = 1 * time.Second
	maxConcurrentRenders     = 2
	stlBinaryHeaderSize      = 84 // 80-byte header + uint32 triangle count
	stlBinaryTriangleRecSize = 50
)

// renderSem is the process-wide gate bounding concurrent renders. A render that
// hangs holds its slot until it finishes (it cannot be killed), so with a small
// buffer at most maxConcurrentRenders stuck goroutines can accumulate; further
// uploads skip the thumbnail instead of piling up.
var renderSem = make(chan struct{}, maxConcurrentRenders)

// ThumbnailService handles generating thumbnail images from 3D model files.
type ThumbnailService struct {
	logger logger.Logger

	// render is the seam that rasterizes a parsed mesh. It is a field so tests can
	// inject a panicking or slow renderer without touching the parse path.
	render func(*fauxgl.Mesh) (io.Reader, error)

	// Tunables (fields so tests can shrink them); defaults set by the constructor.
	renderTimeout  time.Duration
	acquireTimeout time.Duration
	maxTriangles   int
	sem            chan struct{}
}

// NewThumbnailService creates a new ThumbnailService instance.
func NewThumbnailService(logger logger.Logger) *ThumbnailService {
	return &ThumbnailService{
		logger:         logger,
		render:         renderMesh,
		renderTimeout:  defaultRenderTimeout,
		acquireTimeout: defaultAcquireTimeout,
		maxTriangles:   defaultMaxTriangles,
		sem:            renderSem,
	}
}

// Generate renders a 3D model file to a PNG thumbnail, returning the PNG as an
// io.Reader or nil when a thumbnail cannot (or should not) be produced. It never
// returns an error and never propagates a parse/render failure, timeout, OOM-risk
// or panic to the caller: thumbnails are best-effort and must not block or break an
// upload. Supported formats: .stl (binary/ASCII), .3mf.
//
// Safety: cheap pre-parse guards reject obviously-malicious inputs; a bounded
// semaphore caps concurrent renders; the parse+render runs in a goroutine guarded
// by recover and a context timeout derived from ctx.
func (s *ThumbnailService) Generate(ctx context.Context, file io.Reader, format string) io.Reader {
	format = strings.ToLower(format)
	if format != ".stl" && format != ".3mf" {
		s.logger.Warning(ctx, "Unsupported format for thumbnail", map[string]interface{}{"format": format})
		return nil
	}

	// Buffer the input once so we can run cheap guards before spending a slot and
	// so go3mf (which needs a ReaderAt + size) and the STL temp file both work.
	data, err := io.ReadAll(file)
	if err != nil {
		s.logger.Warning(ctx, "Failed to read file for thumbnail", map[string]interface{}{"error": err.Error()})
		return nil
	}

	// Cheap pre-parse guard: a binary STL whose header claims an absurd triangle
	// count would make the loader allocate gigabytes. Reject before using a slot.
	if format == ".stl" {
		if count, isBinary := binarySTLTriangleCount(data); isBinary && count > uint32(s.maxTriangles) {
			s.logger.Warning(ctx, "Thumbnail skipped: STL triangle count over cap", map[string]interface{}{"count": count, "cap": s.maxTriangles})
			return nil
		}
	}

	// Acquire a render slot. If none frees up quickly, skip the thumbnail rather
	// than queueing behind a stuck render.
	select {
	case s.sem <- struct{}{}:
	case <-time.After(s.acquireTimeout):
		s.logger.Warning(ctx, "Thumbnail skipped: no render slot available", nil)
		return nil
	case <-ctx.Done():
		return nil
	}

	renderCtx, cancel := context.WithTimeout(ctx, s.renderTimeout)
	defer cancel()

	// Buffered so a send never blocks even if we already returned via timeout.
	ch := make(chan io.Reader, 1)
	go func() {
		// The goroutine owns the slot and releases it only when it actually
		// finishes, so a stuck render keeps its slot (bounding total damage).
		defer func() { <-s.sem }()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error(context.Background(), "Thumbnail generation panicked", map[string]interface{}{"panic": fmt.Sprintf("%v", r)})
				ch <- nil
			}
		}()

		r, gerr := s.generateByFormat(data, format)
		if gerr != nil {
			s.logger.Warning(context.Background(), "Thumbnail generation failed", map[string]interface{}{"format": format, "error": gerr.Error()})
			r = nil
		}
		ch <- r
	}()

	select {
	case r := <-ch:
		return r
	case <-renderCtx.Done():
		s.logger.Warning(ctx, "Thumbnail generation timed out", map[string]interface{}{"timeout": s.renderTimeout.String()})
		return nil
	}
}

func (s *ThumbnailService) generateByFormat(data []byte, format string) (io.Reader, error) {
	switch format {
	case ".stl":
		return s.generateSTLThumbnail(data)
	case ".3mf":
		return s.generate3MFThumbnail(data)
	default:
		return nil, fmt.Errorf("unsupported format for thumbnail: %s", format)
	}
}

// binarySTLTriangleCount reports the declared triangle count when data looks like
// a BINARY STL (its size matches the header exactly: 84 + 50*count). ASCII STLs
// won't match, so it returns (0, false) for them and the caller skips this guard.
func binarySTLTriangleCount(data []byte) (uint32, bool) {
	if len(data) < stlBinaryHeaderSize {
		return 0, false
	}
	count := binary.LittleEndian.Uint32(data[80:84])
	expected := int64(stlBinaryHeaderSize) + int64(stlBinaryTriangleRecSize)*int64(count)
	if int64(len(data)) == expected {
		return count, true
	}
	return 0, false
}

func (s *ThumbnailService) generateSTLThumbnail(data []byte) (io.Reader, error) {
	// fauxgl.LoadSTL requires a file path.
	tmpFile, err := os.CreateTemp("", "model3d-*.stl")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	_ = tmpFile.Close()

	mesh, err := fauxgl.LoadSTL(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to parse STL: %w", err)
	}

	if len(mesh.Triangles) > s.maxTriangles {
		return nil, fmt.Errorf("mesh has %d triangles, over the %d cap", len(mesh.Triangles), s.maxTriangles)
	}

	return s.render(mesh)
}

// generate3MFThumbnail parses a 3MF file, extracts all mesh geometry from build
// items (bounded by the triangle cap) and renders a thumbnail.
func (s *ThumbnailService) generate3MFThumbnail(data []byte) (io.Reader, error) {
	reader := bytes.NewReader(data)
	decoder := go3mf.NewDecoder(reader, int64(len(data)))

	var model go3mf.Model
	if err := decoder.Decode(&model); err != nil {
		return nil, fmt.Errorf("failed to decode 3MF: %w", err)
	}

	mesh := fauxgl.NewEmptyMesh()
	triangleCount := 0

	for _, item := range model.Build.Items {
		obj, ok := model.FindObject(item.ObjectPath(), item.ObjectID)
		if !ok || obj == nil || obj.Mesh == nil {
			continue
		}

		m := obj.Mesh
		vertices := m.Vertices.Vertex
		for _, tri := range m.Triangles.Triangle {
			if int(tri.V1) >= len(vertices) || int(tri.V2) >= len(vertices) || int(tri.V3) >= len(vertices) {
				continue // skip invalid triangle indices
			}

			triangleCount++
			if triangleCount > s.maxTriangles {
				return nil, fmt.Errorf("3MF mesh exceeds the %d triangle cap", s.maxTriangles)
			}

			v1 := vertices[tri.V1]
			v2 := vertices[tri.V2]
			v3 := vertices[tri.V3]

			p1 := fauxgl.Vector{X: float64(v1.X()), Y: float64(v1.Y()), Z: float64(v1.Z())}
			p2 := fauxgl.Vector{X: float64(v2.X()), Y: float64(v2.Y()), Z: float64(v2.Z())}
			p3 := fauxgl.Vector{X: float64(v3.X()), Y: float64(v3.Y()), Z: float64(v3.Z())}

			// Compute face normal from edge vectors
			e1 := p2.Sub(p1)
			e2 := p3.Sub(p1)
			n := e1.Cross(e2).Normalize()

			t := fauxgl.Triangle{
				V1: fauxgl.Vertex{Position: p1, Normal: n},
				V2: fauxgl.Vertex{Position: p2, Normal: n},
				V3: fauxgl.Vertex{Position: p3, Normal: n},
			}

			mesh.Triangles = append(mesh.Triangles, &t)
		}
	}

	if triangleCount == 0 {
		return nil, fmt.Errorf("3MF file contains no renderable geometry")
	}

	return s.render(mesh)
}

// renderMesh rasterizes a mesh to a PNG. It is a package function (no service
// state) so it can be used as the default render seam.
func renderMesh(mesh *fauxgl.Mesh) (io.Reader, error) {
	const (
		width  = 512
		height = 512
		fovy   = 30.0
		near   = 0.1
	)

	// Compute bounding box to position camera
	box := mesh.BoundingBox()
	center := box.Center()
	size := box.Size()
	diagonal := math.Sqrt(size.X*size.X + size.Y*size.Y + size.Z*size.Z)

	if diagonal == 0 {
		return nil, fmt.Errorf("mesh has zero size")
	}

	// Position the camera far enough that the model's bounding sphere (radius
	// diagonal/2) fits ENTIRELY inside the frustum. With a 30deg vertical FOV the
	// sphere fits when distance >= R/sin(15deg) ~= 1.93*diagonal. Sitting closer
	// makes triangles cross the frustum planes and exercise fauxgl's triangle
	// clipper, which can emit vertices on the near plane (w~0) and send the
	// rasterizer into a near-infinite pixel loop. Keeping the whole model inside the
	// frustum avoids the clip path entirely. The unit direction keeps the center at
	// ~distance from the eye.
	const camDistanceFactor = 2.5 // > 1.93 margin so nothing is ever clipped
	distance := diagonal * camDistanceFactor
	dir := fauxgl.Vector{X: 0.6, Y: 0.6, Z: 0.5}.Normalize()
	eye := fauxgl.Vector{
		X: center.X + distance*dir.X,
		Y: center.Y + distance*dir.Y,
		Z: center.Z + distance*dir.Z,
	}

	// Compute far plane dynamically based on model size to avoid clipping large models
	far := math.Max(1000.0, diagonal*8.0)

	aspect := float64(width) / float64(height)
	matrix := fauxgl.LookAt(eye, center, fauxgl.Vector{X: 0, Y: 0, Z: 1})
	matrix = matrix.Perspective(fovy, aspect, near, far)

	// Create the rendering context
	ctx := fauxgl.NewContext(width, height)
	ctx.ClearColorBufferWith(fauxgl.HexColor("F5F5F5"))
	ctx.ClearDepthBuffer()

	// Set up shader with lighting
	shader := fauxgl.NewPhongShader(matrix, eye, fauxgl.Vector{
		X: eye.X + distance,
		Y: eye.Y + distance,
		Z: eye.Z + distance*0.5,
	})
	shader.ObjectColor = fauxgl.HexColor("808080")
	shader.AmbientColor = fauxgl.HexColor("404040")
	shader.DiffuseColor = fauxgl.HexColor("C0C0C0")
	shader.SpecularColor = fauxgl.HexColor("404040")
	shader.SpecularPower = 32.0
	ctx.Shader = shader

	// Draw triangles sequentially rather than ctx.DrawMesh, which fans out across
	// runtime.NumCPU goroutines that race on the shared color/depth buffers (a data
	// race fauxgl does not guard). Single-threaded rendering is deterministic and
	// race-free, at a negligible cost for thumbnail-sized meshes.
	for _, tri := range mesh.Triangles {
		ctx.DrawTriangle(tri)
	}

	// Convert fauxgl image to standard image.Image and encode as PNG
	img := ctx.Image()
	return encodePNG(img)
}

func encodePNG(img image.Image) (io.Reader, error) {
	if img == nil {
		return nil, fmt.Errorf("nil image")
	}

	// fauxgl returns its own image type; convert to NRGBA for PNG encoding
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			nrgba.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, nrgba); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return &buf, nil
}
