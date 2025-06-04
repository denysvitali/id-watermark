package main

import (
	"bytes"
	"fmt"
	"github.com/alexflint/go-arg"
	"github.com/golang/freetype/truetype"
	log "github.com/sirupsen/logrus"
	"golang.org/x/image/draw"
	"golang.org/x/image/font/opentype"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/vgimg"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"time"
)

// Args defines the command line arguments
type Args struct {
	Input    string  `arg:"required" help:"path to input image"`
	Output   string  `arg:"required" help:"path to output image"`
	Company  string  `arg:"required" help:"company name for watermark"`
	FontPath string  `arg:"-t" default:"./DejaVuSans.ttf" help:"path to TTF font file"`
	FontSize float64 `arg:"-s" default:"40" help:"font size for watermark"`
	Opacity  uint8   `arg:"-o" default:"40" help:"watermark opacity (0-255)"`
	Angle    float64 `arg:"-a" default:"34" help:"watermark angle in degrees"`
	LogLevel string  `arg:"-l" default:"info" help:"log level (debug, info, warn, error)"`
}

// WatermarkConfig holds the configuration for watermark application
type WatermarkConfig struct {
	CompanyName string
	Timestamp   time.Time
	FontSize    float64
	Opacity     uint8
	Angle       float64
	Font        *opentype.Font
}

func initLogger(args Args) {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	level, err := log.ParseLevel(args.LogLevel)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
}

func getFont(name string) (*truetype.Font, error) {
	data, err := os.ReadFile(name)
	font, err := truetype.Parse(data)
	return font, err
}

// Heavily inspired from:
// https://github.com/yumexupanic/idcard_watermark/blob/master/src/main.go
func WaterMark(img image.Image, markText string, config WatermarkConfig) (image.Image, error) {
	bounds := img.Bounds()
	w := vg.Length(bounds.Max.X) * vg.Inch / vgimg.DefaultDPI
	h := vg.Length(bounds.Max.Y) * vg.Inch / vgimg.DefaultDPI
	diagonal := vg.Length(math.Sqrt(float64(w*w + h*h)))

	// create a canvas, which width and height are diagonal
	c := vgimg.New(diagonal, diagonal)

	rect := vg.Rectangle{}
	rect.Min.X = diagonal/2 - w/2
	rect.Min.Y = diagonal/2 - h/2
	rect.Max.X = diagonal/2 + w/2
	rect.Max.Y = diagonal/2 + h/2
	c.DrawImage(rect, img)

	mincho := font.Font{
		Typeface: "MyFont",
		Size:     vg.Length(config.FontSize),
	}
	fnt := font.Face{
		Font: mincho,
		Face: config.Font,
	}
	plot.DefaultFont = mincho
	c.SetColor(color.RGBA{R: 150, G: 150, B: 150, A: config.Opacity})
	lineHeight := vg.Length(config.FontSize)
	textWidth := fnt.Width(markText)
	xDistance := vg.Length(30)
	yDistance := vg.Length(30)
	line := 0
	for offset := -2 * diagonal; offset < 2*diagonal; offset += lineHeight + yDistance {
		line++
		for xOffset := -vg.Length(line)*1.5*textWidth + vg.Length(0); xOffset < w; xOffset += textWidth + xDistance {
			c.FillString(fnt, vg.Point{X: xOffset, Y: offset}, markText)
		}
	}

	jc := vgimg.PngCanvas{Canvas: c}
	buff := new(bytes.Buffer)
	jc.WriteTo(buff)
	img, _, err := image.Decode(buff)
	if err != nil {
		return nil, err
	}

	ctp := int(diagonal * vgimg.DefaultDPI / vg.Inch / 2)

	// cutout the marked image
	size := bounds.Size()
	bounds = image.Rect(ctp-size.X/2, ctp-size.Y/2, ctp+size.X/2, ctp+size.Y/2)
	rv := image.NewRGBA(bounds)
	draw.Draw(rv, bounds, img, bounds.Min, draw.Src)
	return rv, nil
}

func processImage(inputPath, outputPath string, config WatermarkConfig) error {
	log.WithFields(log.Fields{
		"input":   inputPath,
		"output":  outputPath,
		"company": config.CompanyName,
	}).Info("Processing image")

	// Open and decode input image
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("error opening input file: %v", err)
	}
	defer inputFile.Close()

	inputImage, err := jpeg.Decode(inputFile)
	if err != nil {
		return fmt.Errorf("error decoding image: %v", err)
	}

	// Create output image
	bounds := inputImage.Bounds()
	output := image.NewRGBA(bounds)

	// Draw original image
	draw.Draw(output, bounds, inputImage, bounds.Min, draw.Src)

	// Create watermark text
	watermarkText := fmt.Sprintf("%s - %s",
		config.CompanyName,
		config.Timestamp.Format("2006-01-02"))

	// Apply watermark
	outputImage, err := WaterMark(output, watermarkText, config)
	if err != nil {
		return fmt.Errorf("error applying watermark: %v", err)
	}

	// Save output image
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outputFile.Close()

	err = jpeg.Encode(outputFile, outputImage, &jpeg.Options{Quality: 95})
	if err != nil {
		return fmt.Errorf("error encoding output image: %v", err)
	}

	log.Info("Watermark applied successfully!")
	return nil
}

func main() {
	var args Args
	arg.MustParse(&args)

	initLogger(args)

	// Load font
	loadedFont, err := loadFont(args.FontPath)
	if err != nil {
		log.WithError(err).Fatal("Failed to load font")
	}

	config := WatermarkConfig{
		CompanyName: args.Company,
		Timestamp:   time.Now(),
		FontSize:    args.FontSize,
		Opacity:     args.Opacity,
		Angle:       args.Angle,
		Font:        loadedFont,
	}

	if err := processImage(args.Input, args.Output, config); err != nil {
		log.WithError(err).Fatal("Failed to process image")
	}
}

func loadFont(path string) (*opentype.Font, error) {
	fontData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading font file: %v", err)
	}
	return opentype.Parse(fontData)
}
