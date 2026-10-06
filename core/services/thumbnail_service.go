package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"strings"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/fogleman/fauxgl"
	"github.com/hpinc/go3mf"
)

// ThumbnailService handles generating thumbnail images from 3D model files.
type ThumbnailService struct {
	logger logger.Logger
}

// NewThumbnailService creates a new ThumbnailService instance.
func NewThumbnailService(logger logger.Logger) *ThumbnailService {
	return &ThumbnailService{logger: logger}
}

// Generate renders a 3D model file to a PNG thumbnail.
// Returns the PNG image as an io.Reader, or nil if rendering fails or format is unsupported.
// This method is fault-tolerant: errors are logged but never propagated to the caller.
//
// Supported formats: .stl (binary/ASCII), .3mf (via go3mf mesh extraction).
func (s *ThumbnailService) Generate(file io.Reader, format string) (io.Reader, error) {
	format = strings.ToLower(format)

	var reader io.Reader
	var err error

	switch format {
	case ".stl":
		reader, err = s.generateSTLThumbnail(file)
	case ".3mf":
		reader, err = s.generate3MFThumbnail(file)
	default:
		return nil, fmt.Errorf("unsupported format for thumbnail: %s", format)
	}

	if err != nil {
		s.logger.Warning(context.Background(), "Thumbnail generation failed", map[string]interface{}{
			"format": format,
			"error":  err.Error(),
		})
		return nil, nil
	}

	return reader, nil
}

func (s *ThumbnailService) generateSTLThumbnail(file io.Reader) (io.Reader, error) {
	// Write to temp file since fauxgl.LoadSTL requires a file path
	tmpFile, err := os.CreateTemp("", "model3d-*.stl")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	mesh, err := fauxgl.LoadSTL(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to parse STL: %w", err)
	}

	return s.renderMesh(mesh)
}

// generate3MFThumbnail parses a 3MF file, extracts all mesh geometry from build items,
// converts to fauxgl triangles and renders a thumbnail using the shared renderMesh pipeline.
func (s *ThumbnailService) generate3MFThumbnail(file io.Reader) (io.Reader, error) {
	// go3mf.NewDecoder requires an io.ReaderAt + size, so buffer the entire file
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read 3MF data: %w", err)
	}

	reader := bytes.NewReader(data)
	decoder := go3mf.NewDecoder(reader, int64(len(data)))

	var model go3mf.Model
	if err := decoder.Decode(&model); err != nil {
		return nil, fmt.Errorf("failed to decode 3MF: %w", err)
	}

	// Extract all meshes from build items and merge into a single fauxgl mesh
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
			triangleCount++
		}
	}

	if triangleCount == 0 {
		return nil, fmt.Errorf("3MF file contains no renderable geometry")
	}

	s.logger.Info(context.Background(), "3MF mesh extracted for thumbnail", map[string]interface{}{
		"triangles": triangleCount,
	})

	return s.renderMesh(mesh)
}

func (s *ThumbnailService) renderMesh(mesh *fauxgl.Mesh) (io.Reader, error) {
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

	ctx.DrawMesh(mesh)

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
