// Package watermark provides image watermarking functionality for ID cards and documents
package watermark

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
	"golang.org/x/image/font/opentype"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/vgimg"
)

// Config holds the configuration for watermark application
type Config struct {
	CompanyName    string
	Timestamp      time.Time
	FontSize       float64
	Opacity        uint8
	Angle          float64
	Font           *opentype.Font
	TextSpacing    float64
	LineSpacing    float64
	Quality        int
	WatermarkColor color.RGBA
}

// Processor handles image watermarking operations
type Processor struct {
	config *Config
}

// NewProcessor creates a new watermark processor with the given configuration
func NewProcessor(config *Config) *Processor {
	if config.Timestamp.IsZero() {
		config.Timestamp = time.Now()
	}
	return &Processor{config: config}
}

// ProcessFile applies watermark to a single image file
func (p *Processor) ProcessFile(inputPath, outputPath string) error {
	// Open and decode input image
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("opening input file: %w", err)
	}
	defer inputFile.Close()

	// Detect and decode image format
	inputImage, err := p.decodeImage(inputFile, inputPath)
	if err != nil {
		return fmt.Errorf("decoding image: %w", err)
	}

	// Apply watermark
	watermarkedImage, err := p.applyWatermark(inputImage)
	if err != nil {
		return fmt.Errorf("applying watermark: %w", err)
	}

	// Save output image
	if err := p.saveImage(watermarkedImage, outputPath); err != nil {
		return fmt.Errorf("saving image: %w", err)
	}

	return nil
}

// ProcessImage applies watermark to an image.Image and returns the result
func (p *Processor) ProcessImage(img image.Image) (image.Image, error) {
	return p.applyWatermark(img)
}

// decodeImage decodes an image from a file based on its extension
func (p *Processor) decodeImage(file *os.File, filename string) (image.Image, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".png":
		return png.Decode(file)
	case ".jpg", ".jpeg":
		return jpeg.Decode(file)
	default:
		return nil, fmt.Errorf("unsupported input format: %s (supported: .jpg, .jpeg, .png)", ext)
	}
}

// saveImage saves an image to a file based on the output path extension
func (p *Processor) saveImage(img image.Image, outputPath string) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	ext := strings.ToLower(filepath.Ext(outputPath))

	switch ext {
	case ".png":
		return png.Encode(outputFile, img)
	case ".jpg", ".jpeg":
		return jpeg.Encode(outputFile, img, &jpeg.Options{Quality: p.config.Quality})
	default:
		return fmt.Errorf("unsupported output format: %s (supported: .jpg, .jpeg, .png)", ext)
	}
}

// applyWatermark applies the watermark to an image
func (p *Processor) applyWatermark(img image.Image) (image.Image, error) {
	bounds := img.Bounds()
	w := vg.Length(bounds.Max.X) * vg.Inch / vgimg.DefaultDPI
	h := vg.Length(bounds.Max.Y) * vg.Inch / vgimg.DefaultDPI
	diagonal := vg.Length(math.Sqrt(float64(w*w + h*h)))

	// Create a canvas with diagonal dimensions
	c := vgimg.New(diagonal, diagonal)

	// Position the original image in the center
	rect := vg.Rectangle{
		Min: vg.Point{X: diagonal/2 - w/2, Y: diagonal/2 - h/2},
		Max: vg.Point{X: diagonal/2 + w/2, Y: diagonal/2 + h/2},
	}
	c.DrawImage(rect, img)

	// Configure font
	fontConfig := font.Font{
		Typeface: "WatermarkFont",
		Size:     vg.Length(p.config.FontSize),
	}

	fontFace := font.Face{
		Font: fontConfig,
		Face: p.config.Font,
	}

	plot.DefaultFont = fontConfig
	c.SetColor(p.config.WatermarkColor)

	// Create watermark text
	watermarkText := fmt.Sprintf("%s - %s",
		p.config.CompanyName,
		p.config.Timestamp.Format("2006-01-02"))

	// Apply repeating watermark pattern
	lineHeight := vg.Length(p.config.FontSize)
	textWidth := fontFace.Width(watermarkText)
	xDistance := vg.Length(p.config.TextSpacing)
	yDistance := vg.Length(p.config.LineSpacing)

	line := 0
	for offset := -2 * diagonal; offset < 2*diagonal; offset += lineHeight + yDistance {
		line++
		for xOffset := -vg.Length(line) * 1.5 * textWidth; xOffset < w; xOffset += textWidth + xDistance {
			c.FillString(fontFace, vg.Point{X: xOffset, Y: offset}, watermarkText)
		}
	}

	// Convert back to image
	jc := vgimg.PngCanvas{Canvas: c}
	buff := new(bytes.Buffer)
	if _, err := jc.WriteTo(buff); err != nil {
		return nil, fmt.Errorf("writing canvas: %w", err)
	}

	processedImg, _, err := image.Decode(buff)
	if err != nil {
		return nil, fmt.Errorf("decoding processed image: %w", err)
	}

	// Crop to original size
	ctp := int(diagonal * vgimg.DefaultDPI / vg.Inch / 2)
	size := bounds.Size()
	cropBounds := image.Rect(ctp-size.X/2, ctp-size.Y/2, ctp+size.X/2, ctp+size.Y/2)

	result := image.NewRGBA(bounds)
	draw.Draw(result, bounds, processedImg, cropBounds.Min, draw.Src)

	return result, nil
}

// ValidateConfig validates the watermark configuration
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if strings.TrimSpace(config.CompanyName) == "" {
		return fmt.Errorf("company name cannot be empty")
	}

	if config.FontSize < 10 || config.FontSize > 200 {
		return fmt.Errorf("font size must be between 10 and 200, got: %.1f", config.FontSize)
	}

	if config.TextSpacing < 5 || config.TextSpacing > 200 {
		return fmt.Errorf("text spacing must be between 5 and 200, got: %.1f", config.TextSpacing)
	}

	if config.LineSpacing < 5 || config.LineSpacing > 200 {
		return fmt.Errorf("line spacing must be between 5 and 200, got: %.1f", config.LineSpacing)
	}

	if config.Quality < 1 || config.Quality > 100 {
		return fmt.Errorf("quality must be between 1 and 100, got: %d", config.Quality)
	}

	if config.Font == nil {
		return fmt.Errorf("font cannot be nil")
	}

	return nil
}
