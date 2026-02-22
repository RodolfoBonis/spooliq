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
func (s *ThumbnailService) Generate(file io.Reader, format string) (io.Reader, error) {
	format = strings.ToLower(format)

	if format == ".3mf" {
		s.logger.Warning(context.Background(), "Thumbnail generation not yet supported for 3MF files", nil)
		return nil, nil
	}

	if format != ".stl" {
		return nil, fmt.Errorf("unsupported format for thumbnail: %s", format)
	}

	reader, err := s.generateSTLThumbnail(file)
	if err != nil {
		s.logger.Warning(context.Background(), "STL thumbnail generation failed", map[string]interface{}{
			"error": err.Error(),
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

func (s *ThumbnailService) renderMesh(mesh *fauxgl.Mesh) (io.Reader, error) {
	const (
		width  = 512
		height = 512
		fovy   = 30.0
		near   = 0.1
		far    = 1000.0
	)

	// Compute bounding box to position camera
	box := mesh.BoundingBox()
	center := box.Center()
	size := box.Size()
	diagonal := math.Sqrt(size.X*size.X + size.Y*size.Y + size.Z*size.Z)

	if diagonal == 0 {
		return nil, fmt.Errorf("mesh has zero size")
	}

	// Position camera to see the entire model
	distance := diagonal * 1.5
	eye := fauxgl.Vector{
		X: center.X + distance*0.6,
		Y: center.Y + distance*0.6,
		Z: center.Z + distance*0.5,
	}

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
