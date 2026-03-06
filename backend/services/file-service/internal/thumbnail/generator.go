package thumbnail

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/nfnt/resize"
)

// Generator generates thumbnails from images
type Generator struct {
	maxWidth  int
	maxHeight int
	quality   int
	format    string
}

// NewGenerator creates a new thumbnail generator
func NewGenerator(maxWidth, maxHeight, quality int, format string) *Generator {
	if maxWidth == 0 {
		maxWidth = 512
	}
	if maxHeight == 0 {
		maxHeight = 512
	}
	if quality == 0 {
		quality = 85
	}
	if format == "" {
		format = "jpeg"
	}

	return &Generator{
		maxWidth:  maxWidth,
		maxHeight: maxHeight,
		quality:   quality,
		format:    format,
	}
}

// Generate generates a thumbnail from an image
func (g *Generator) Generate(reader io.Reader) ([]byte, int, int, error) {
	// Decode image
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	// Calculate new dimensions maintaining aspect ratio
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	newWidth, newHeight := g.calculateDimensions(width, height)

	// Resize image
	thumbnail := resize.Resize(uint(newWidth), uint(newHeight), img, resize.Lanczos3)

	// Encode to buffer
	var buf bytes.Buffer

	switch g.format {
	case "png":
		err = png.Encode(&buf, thumbnail)
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: g.quality})
	default:
		err = jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: g.quality})
	}

	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	// Suppress unused variable warning
	_ = format

	return buf.Bytes(), newWidth, newHeight, nil
}

// calculateDimensions calculates new dimensions maintaining aspect ratio
func (g *Generator) calculateDimensions(width, height int) (int, int) {
	if width <= g.maxWidth && height <= g.maxHeight {
		return width, height
	}

	widthRatio := float64(g.maxWidth) / float64(width)
	heightRatio := float64(g.maxHeight) / float64(height)

	// Use the smaller ratio to ensure the image fits within bounds
	ratio := widthRatio
	if heightRatio < widthRatio {
		ratio = heightRatio
	}

	newWidth := int(float64(width) * ratio)
	newHeight := int(float64(height) * ratio)

	return newWidth, newHeight
}

// CanThumbnail checks if a content type can be thumbnailed
func (g *Generator) CanThumbnail(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/jpg", "image/png", "image/gif", "image/bmp":
		return true
	default:
		return false
	}
}
