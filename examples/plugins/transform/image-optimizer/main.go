package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/disintegration/imaging"
	"github.com/forest6511/gdl/pkg/plugin"
)

// ImageOptimizerPlugin implements image optimization transformation
type ImageOptimizerPlugin struct {
	quality         int
	maxWidth        int
	maxHeight       int
	enableResize    bool
	outputFormat    string // "jpeg", "png", "auto"
	compressionMode string // "fast", "balanced", "best"
}

// NewImageOptimizerPlugin creates a new image optimizer plugin
func NewImageOptimizerPlugin() *ImageOptimizerPlugin {
	return &ImageOptimizerPlugin{
		quality:         85,
		maxWidth:        1920,
		maxHeight:       1080,
		enableResize:    true,
		outputFormat:    "auto",
		compressionMode: "balanced",
	}
}

// Name returns the plugin name
func (p *ImageOptimizerPlugin) Name() string {
	return "image-optimizer"
}

// Version returns the plugin version
func (p *ImageOptimizerPlugin) Version() string {
	return "1.0.0"
}

// Init initializes the image optimizer plugin with configuration
func (p *ImageOptimizerPlugin) Init(config map[string]interface{}) error {
	if quality, ok := config["quality"].(float64); ok {
		if quality >= 1 && quality <= 100 {
			p.quality = int(quality)
		} else {
			return fmt.Errorf("quality must be between 1 and 100")
		}
	} else if quality, ok := config["quality"].(int); ok {
		if quality >= 1 && quality <= 100 {
			p.quality = quality
		} else {
			return fmt.Errorf("quality must be between 1 and 100")
		}
	}

	if maxWidth, ok := config["max_width"].(float64); ok {
		p.maxWidth = int(maxWidth)
	} else if maxWidth, ok := config["max_width"].(int); ok {
		p.maxWidth = maxWidth
	}

	if maxHeight, ok := config["max_height"].(float64); ok {
		p.maxHeight = int(maxHeight)
	} else if maxHeight, ok := config["max_height"].(int); ok {
		p.maxHeight = maxHeight
	}

	if enableResize, ok := config["enable_resize"].(bool); ok {
		p.enableResize = enableResize
	}

	if outputFormat, ok := config["output_format"].(string); ok {
		if outputFormat == "jpeg" || outputFormat == "png" || outputFormat == "auto" {
			p.outputFormat = outputFormat
		} else {
			return fmt.Errorf("output_format must be 'jpeg', 'png', or 'auto'")
		}
	}

	if compressionMode, ok := config["compression_mode"].(string); ok {
		if compressionMode == "fast" || compressionMode == "balanced" || compressionMode == "best" {
			p.compressionMode = compressionMode
		} else {
			return fmt.Errorf("compression_mode must be 'fast', 'balanced', or 'best'")
		}
	}

	return nil
}

// Close cleans up the plugin resources
func (p *ImageOptimizerPlugin) Close() error {
	// No resources to clean up
	return nil
}

// ValidateAccess validates security access for operations
func (p *ImageOptimizerPlugin) ValidateAccess(operation, resource string) error {
	// Allow basic transform operations
	allowedOps := []string{"transform", "optimize", "resize", "compress"}
	for _, op := range allowedOps {
		if operation == op {
			return nil
		}
	}
	return fmt.Errorf("operation %s not allowed for image optimizer plugin", operation)
}

// Transform optimizes image data
func (p *ImageOptimizerPlugin) Transform(data []byte) ([]byte, error) {
	// Check if the data is an image
	if !p.isImage(data) {
		// Not an image, return unchanged
		return data, nil
	}

	// Decode the image
	img, format, err := p.decodeImage(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if enabled and necessary
	if p.enableResize {
		img = p.resizeImage(img)
	}

	// Determine output format
	outputFormat := p.outputFormat
	if outputFormat == "auto" {
		outputFormat = format
		// Convert PNG to JPEG for better compression if no transparency
		if format == "png" && !p.hasTransparency(img) {
			outputFormat = "jpeg"
		}
	}

	// Encode the optimized image
	optimizedData, err := p.encodeImage(img, outputFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to encode optimized image: %w", err)
	}

	return optimizedData, nil
}

// isImage checks if the data represents an image
func (p *ImageOptimizerPlugin) isImage(data []byte) bool {
	if len(data) < 12 {
		return false
	}

	// Check for common image signatures
	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return true
	}

	// JPEG: FF D8 FF
	if bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
		return true
	}

	// GIF: 47 49 46 38 (GIF8)
	if bytes.HasPrefix(data, []byte{0x47, 0x49, 0x46, 0x38}) {
		return true
	}

	// WebP: 52 49 46 46 ... 57 45 42 50 (RIFF...WEBP)
	if len(data) >= 12 && bytes.HasPrefix(data, []byte{0x52, 0x49, 0x46, 0x46}) &&
		bytes.Equal(data[8:12], []byte{0x57, 0x45, 0x42, 0x50}) {
		return true
	}

	return false
}

// decodeImage decodes image data and returns the image and format
func (p *ImageOptimizerPlugin) decodeImage(data []byte) (image.Image, string, error) {
	reader := bytes.NewReader(data)

	// Try to decode as different formats
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, "", err
	}

	return img, format, nil
}

// resizeImage resizes the image if it exceeds maximum dimensions
func (p *ImageOptimizerPlugin) resizeImage(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check if resizing is needed
	if width <= p.maxWidth && height <= p.maxHeight {
		return img
	}

	// Calculate new dimensions maintaining aspect ratio
	var newWidth, newHeight int
	if width > height {
		newWidth = p.maxWidth
		newHeight = int(float64(height) * float64(p.maxWidth) / float64(width))
		if newHeight > p.maxHeight {
			newHeight = p.maxHeight
			newWidth = int(float64(width) * float64(p.maxHeight) / float64(height))
		}
	} else {
		newHeight = p.maxHeight
		newWidth = int(float64(width) * float64(p.maxHeight) / float64(height))
		if newWidth > p.maxWidth {
			newWidth = p.maxWidth
			newHeight = int(float64(height) * float64(p.maxWidth) / float64(width))
		}
	}

	// Choose resampling filter based on compression mode
	var filter imaging.ResampleFilter
	switch p.compressionMode {
	case "fast":
		filter = imaging.Box
	case "balanced":
		filter = imaging.Linear
	case "best":
		filter = imaging.Lanczos
	default:
		filter = imaging.Linear
	}

	return imaging.Resize(img, newWidth, newHeight, filter)
}

// hasTransparency checks if the image has transparency
func (p *ImageOptimizerPlugin) hasTransparency(img image.Image) bool {
	switch img := img.(type) {
	case *image.RGBA:
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a != 0xffff { // Not fully opaque
					return true
				}
			}
		}
	case *image.NRGBA:
		bounds := img.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a != 0xffff { // Not fully opaque
					return true
				}
			}
		}
	default:
		// For other formats, assume no transparency
		return false
	}

	return false
}

// encodeImage encodes the image in the specified format
func (p *ImageOptimizerPlugin) encodeImage(img image.Image, format string) ([]byte, error) {
	var buf bytes.Buffer

	switch format {
	case "jpeg":
		options := &jpeg.Options{
			Quality: p.quality,
		}
		err := jpeg.Encode(&buf, img, options)
		if err != nil {
			return nil, err
		}
	case "png":
		encoder := &png.Encoder{
			CompressionLevel: p.getPNGCompressionLevel(),
		}
		err := encoder.Encode(&buf, img)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported output format: %s", format)
	}

	return buf.Bytes(), nil
}

// getPNGCompressionLevel returns PNG compression level based on compression mode
func (p *ImageOptimizerPlugin) getPNGCompressionLevel() png.CompressionLevel {
	switch p.compressionMode {
	case "fast":
		return png.NoCompression
	case "balanced":
		return png.DefaultCompression
	case "best":
		return png.BestCompression
	default:
		return png.DefaultCompression
	}
}

// GetStats returns optimization statistics (could be used for logging)
func (p *ImageOptimizerPlugin) GetStats(originalSize, optimizedSize int) map[string]interface{} {
	compressionRatio := float64(optimizedSize) / float64(originalSize) * 100
	spaceSaved := originalSize - optimizedSize

	return map[string]interface{}{
		"original_size":       originalSize,
		"optimized_size":      optimizedSize,
		"compression_ratio":   fmt.Sprintf("%.1f%%", compressionRatio),
		"space_saved":         spaceSaved,
		"space_saved_percent": fmt.Sprintf("%.1f%%", (1.0-compressionRatio/100)*100),
	}
}

// ValidateConfig validates the plugin configuration
func (p *ImageOptimizerPlugin) ValidateConfig(config map[string]interface{}) error {
	if quality, ok := config["quality"]; ok {
		switch q := quality.(type) {
		case float64:
			if q < 1 || q > 100 {
				return fmt.Errorf("quality must be between 1 and 100")
			}
		case int:
			if q < 1 || q > 100 {
				return fmt.Errorf("quality must be between 1 and 100")
			}
		default:
			return fmt.Errorf("quality must be a number")
		}
	}

	if outputFormat, ok := config["output_format"].(string); ok {
		if outputFormat != "jpeg" && outputFormat != "png" && outputFormat != "auto" {
			return fmt.Errorf("output_format must be 'jpeg', 'png', or 'auto'")
		}
	}

	if compressionMode, ok := config["compression_mode"].(string); ok {
		if compressionMode != "fast" && compressionMode != "balanced" && compressionMode != "best" {
			return fmt.Errorf("compression_mode must be 'fast', 'balanced', or 'best'")
		}
	}

	return nil
}

// Plugin variable to be loaded by the plugin system
var Plugin plugin.TransformPlugin = &ImageOptimizerPlugin{}

func main() {
	// This is a plugin, so main() is not used when loaded as a shared library
	// But it can be useful for testing the plugin standalone
	fmt.Println("Image Optimizer Transform Plugin")
	fmt.Printf("Name: %s\n", Plugin.Name())
	fmt.Printf("Version: %s\n", Plugin.Version())
}
