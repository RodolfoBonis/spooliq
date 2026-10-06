package services

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image/png"
	"io"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cubeFaces lists the 12 triangles (as vertex-index triples) of an axis-aligned
// cube whose 8 corners are cubeCorners. Shared by the ASCII STL and 3MF fixtures.
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

// minimal3MF builds a minimal valid 3MF package (an OPC/zip container with the
// content-types map, the package relationships, and a single cube mesh).
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

func TestThumbnailService_GenerateSTL(t *testing.T) {
	svc := NewThumbnailService(&logger.NoopLogger{})
	r, err := svc.Generate(bytes.NewReader([]byte(asciiSTLCube())), ".stl")
	require.NoError(t, err)
	assertIsPNG(t, r)
}

func TestThumbnailService_Generate3MF(t *testing.T) {
	svc := NewThumbnailService(&logger.NoopLogger{})
	r, err := svc.Generate(bytes.NewReader(minimal3MF(t)), ".3mf")
	require.NoError(t, err)
	assertIsPNG(t, r)
}

func TestThumbnailService_UnsupportedFormat(t *testing.T) {
	svc := NewThumbnailService(&logger.NoopLogger{})
	r, err := svc.Generate(bytes.NewReader([]byte("data")), ".obj")
	assert.Error(t, err)
	assert.Nil(t, r)
}

func TestThumbnailService_InvalidSTLReturnsNilNil(t *testing.T) {
	// Garbage content for a supported extension is handled fault-tolerantly:
	// Generate logs and returns (nil, nil) rather than propagating an error.
	svc := NewThumbnailService(&logger.NoopLogger{})
	r, err := svc.Generate(bytes.NewReader([]byte("not an stl")), ".stl")
	assert.NoError(t, err)
	assert.Nil(t, r)
}

func TestContentTypeForFile(t *testing.T) {
	assert.Equal(t, "model/stl", ContentTypeForFile("a.stl"))
	assert.Equal(t, "model/stl", ContentTypeForFile("A.STL"))
	assert.Equal(t, "model/3mf", ContentTypeForFile("a.3mf"))
	assert.Equal(t, "application/octet-stream", ContentTypeForFile("a.unknownext"))
}
