package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image/png"
	"io"
	"testing"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/fogleman/fauxgl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cubeCorners / cubeFaces describe an axis-aligned cube, shared by fixtures.
var cubeCorners = [8][3]float64{
	{0, 0, 0}, {20, 0, 0}, {20, 20, 0}, {0, 20, 0},
	{0, 0, 20}, {20, 0, 20}, {20, 20, 20}, {0, 20, 20},
}

var cubeFaces = [12][3]int{
	{0, 1, 2}, {0, 2, 3}, {4, 6, 5}, {4, 7, 6},
	{0, 5, 1}, {0, 4, 5}, {1, 6, 2}, {1, 5, 6},
	{2, 7, 3}, {2, 6, 7}, {3, 4, 0}, {3, 7, 4},
}

// asciiSTLCube builds a tiny valid ASCII STL cube.
func asciiSTLCube() string {
	var b bytes.Buffer
	b.WriteString("solid cube\n")
	for _, f := range cubeFaces {
		b.WriteString("facet normal 0 0 1\n outer loop\n")
		for _, idx := range f {
			v := cubeCorners[idx]
			fmt.Fprintf(&b, "  vertex %g %g %g\n", v[0], v[1], v[2])
		}
		b.WriteString(" endloop\nendfacet\n")
	}
	b.WriteString("endsolid cube\n")
	return b.String()
}

// binarySTLOffCenterCubes builds a valid BINARY STL made of n cubes stacked along
// a diagonal and translated far from the origin, exercising the "large/off-center"
// geometry that used to send the renderer into a near-infinite loop.
func binarySTLOffCenterCubes(n int) []byte {
	offset := 100000.0
	var tris [][4][3]float32
	for i := 0; i < n; i++ {
		d := offset + float64(i)*25.0
		var corners [8][3]float64
		for j, c := range cubeCorners {
			corners[j] = [3]float64{c[0] + d, c[1] + d, c[2] + d}
		}
		for _, f := range cubeFaces {
			var t [4][3]float32
			t[0] = [3]float32{0, 0, 1}
			for k, idx := range f {
				t[k+1] = [3]float32{float32(corners[idx][0]), float32(corners[idx][1]), float32(corners[idx][2])}
			}
			tris = append(tris, t)
		}
	}
	var buf bytes.Buffer
	buf.Write(make([]byte, 80))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(tris)))
	for _, t := range tris {
		for _, v := range t {
			_ = binary.Write(&buf, binary.LittleEndian, v)
		}
		_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	}
	return buf.Bytes()
}

// binarySTLClaiming builds a size-consistent binary STL that declares `count`
// triangles (bodies are zeroed; it is only used to exercise the header guard).
func binarySTLClaiming(count uint32) []byte {
	var buf bytes.Buffer
	buf.Write(make([]byte, 80))
	_ = binary.Write(&buf, binary.LittleEndian, count)
	buf.Write(make([]byte, int(count)*stlBinaryTriangleRecSize))
	return buf.Bytes()
}

// minimal3MF builds a minimal valid 3MF package containing a single cube mesh.
func minimal3MF(t *testing.T) []byte {
	t.Helper()

	contentTypes := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="model" ContentType="application/vnd.ms-package.3dmanufacturing-3dmodel+xml"/>
</Types>`

	rels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Target="/3D/3dmodel.model" Id="rel0" Type="http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"/>
</Relationships>`

	var mesh bytes.Buffer
	mesh.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<model unit="millimeter" xml:lang="en-US" xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02">
 <resources>
  <object id="1" type="model">
   <mesh>
    <vertices>
`)
	for _, v := range cubeCorners {
		fmt.Fprintf(&mesh, `     <vertex x="%g" y="%g" z="%g"/>`+"\n", v[0], v[1], v[2])
	}
	mesh.WriteString("    </vertices>\n    <triangles>\n")
	for _, f := range cubeFaces {
		fmt.Fprintf(&mesh, `     <triangle v1="%d" v2="%d" v3="%d"/>`+"\n", f[0], f[1], f[2])
	}
	mesh.WriteString(`    </triangles>
   </mesh>
  </object>
 </resources>
 <build>
  <item objectid="1"/>
 </build>
</model>`)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range map[string]string{
		"[Content_Types].xml": contentTypes,
		"_rels/.rels":         rels,
		"3D/3dmodel.model":    mesh.String(),
	} {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func assertIsPNG(t *testing.T, r io.Reader) {
	t.Helper()
	require.NotNil(t, r)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NotEmpty(t, data)
	_, err = png.Decode(bytes.NewReader(data))
	assert.NoError(t, err, "thumbnail must be a decodable PNG")
}

func newSvc() *ThumbnailService { return NewThumbnailService(&logger.NoopLogger{}) }

func TestThumbnailService_GenerateSTL(t *testing.T) {
	r := newSvc().Generate(context.Background(), bytes.NewReader([]byte(asciiSTLCube())), ".stl")
	assertIsPNG(t, r)
}

func TestThumbnailService_Generate3MF(t *testing.T) {
	r := newSvc().Generate(context.Background(), bytes.NewReader(minimal3MF(t)), ".3mf")
	assertIsPNG(t, r)
}

func TestThumbnailService_UnsupportedFormat(t *testing.T) {
	r := newSvc().Generate(context.Background(), bytes.NewReader([]byte("data")), ".obj")
	assert.Nil(t, r)
}

func TestThumbnailService_InvalidSTLReturnsNil(t *testing.T) {
	r := newSvc().Generate(context.Background(), bytes.NewReader([]byte("not an stl")), ".stl")
	assert.Nil(t, r)
}

// Regression: a large, far-off-center mesh used to hang the rasterizer. It must now
// render (or at worst time out) well within the render timeout.
func TestThumbnailService_LargeOffCenterMeshDoesNotHang(t *testing.T) {
	svc := newSvc()
	svc.renderTimeout = 9 * time.Second
	data := binarySTLOffCenterCubes(300) // 3600 triangles, offset ~100k

	done := make(chan io.Reader, 1)
	start := time.Now()
	go func() { done <- svc.Generate(context.Background(), bytes.NewReader(data), ".stl") }()

	select {
	case r := <-done:
		assert.NotNil(t, r, "off-center mesh should render")
		assert.Less(t, time.Since(start), 9*time.Second)
	case <-time.After(15 * time.Second):
		t.Fatal("Generate hung on a large off-center mesh")
	}
}

// A binary STL whose header declares more triangles than the cap is rejected by the
// cheap pre-parse guard, fast, without allocating/parsing the mesh.
func TestThumbnailService_BinarySTLHugeCountRejectedFast(t *testing.T) {
	svc := newSvc()
	svc.maxTriangles = 10 // shrink the cap so a tiny fixture trips it
	data := binarySTLClaiming(11)

	start := time.Now()
	r := svc.Generate(context.Background(), bytes.NewReader(data), ".stl")
	assert.Nil(t, r)
	assert.Less(t, time.Since(start), time.Second, "guard must reject before any render slot/work")
}

// A panicking renderer must be recovered: Generate returns nil and does not crash.
func TestThumbnailService_PanicInRenderIsRecovered(t *testing.T) {
	svc := newSvc()
	svc.render = func(*fauxgl.Mesh) (io.Reader, error) { panic("boom") }

	assert.NotPanics(t, func() {
		r := svc.Generate(context.Background(), bytes.NewReader([]byte(asciiSTLCube())), ".stl")
		assert.Nil(t, r)
	})
}

// A slow renderer must be abandoned at the timeout: Generate returns nil promptly.
func TestThumbnailService_RenderTimeout(t *testing.T) {
	svc := newSvc()
	svc.renderTimeout = 100 * time.Millisecond
	svc.render = func(*fauxgl.Mesh) (io.Reader, error) {
		time.Sleep(3 * time.Second)
		return bytes.NewReader([]byte("late")), nil
	}

	start := time.Now()
	r := svc.Generate(context.Background(), bytes.NewReader([]byte(asciiSTLCube())), ".stl")
	assert.Nil(t, r)
	assert.Less(t, time.Since(start), time.Second, "must return at the timeout, not wait for the render")
}

// When all render slots are taken, Generate skips the thumbnail instead of queueing.
func TestThumbnailService_SemaphoreSkipsWhenSaturated(t *testing.T) {
	svc := newSvc()
	svc.sem = make(chan struct{}, 2) // isolated from the package default
	svc.acquireTimeout = 50 * time.Millisecond
	svc.sem <- struct{}{}
	svc.sem <- struct{}{}
	defer func() { <-svc.sem; <-svc.sem }()

	start := time.Now()
	r := svc.Generate(context.Background(), bytes.NewReader([]byte(asciiSTLCube())), ".stl")
	assert.Nil(t, r)
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
	assert.Less(t, elapsed, time.Second)
}

func TestContentTypeForFile(t *testing.T) {
	assert.Equal(t, "model/stl", ContentTypeForFile("a.stl"))
	assert.Equal(t, "model/stl", ContentTypeForFile("A.STL"))
	assert.Equal(t, "model/3mf", ContentTypeForFile("a.3mf"))
	assert.Equal(t, "application/octet-stream", ContentTypeForFile("a.unknownext"))
}
