package watermark

import (
	"fmt"
	"os"

	"golang.org/x/image/font/opentype"
)

// FontManager handles font loading and management
type FontManager struct {
	systemFontPaths []string
}

// NewFontManager creates a new font manager with default system font paths
func NewFontManager() *FontManager {
	return &FontManager{
		systemFontPaths: []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
			"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
			"/System/Library/Fonts/Arial.ttf",
			"/System/Library/Fonts/Helvetica.ttc",
			"/Windows/Fonts/arial.ttf",
			"/Windows/Fonts/Arial.ttf",
			"/usr/share/fonts/TTF/arial.ttf",
			"/usr/share/fonts/TTF/DejaVuSans.ttf",
		},
	}
}

// SetSystemFontPaths sets custom system font paths
func (fm *FontManager) SetSystemFontPaths(paths []string) {
	fm.systemFontPaths = paths
}

// LoadFont loads a font from the specified path, with fallback to system fonts
func (fm *FontManager) LoadFont(fontPath string) (*opentype.Font, error) {
	// Try to load the specified font first
	if fontPath != "" {
		if font, err := fm.loadFontFromPath(fontPath); err == nil {
			return font, nil
		}
	}

	// Fallback to system fonts
	for _, path := range fm.systemFontPaths {
		if fm.fileExists(path) {
			if font, err := fm.loadFontFromPath(path); err == nil {
				return font, nil
			}
		}
	}

	return nil, fmt.Errorf("no suitable font found. Tried: %s and system fonts", fontPath)
}

// loadFontFromPath loads a font from a specific file path
func (fm *FontManager) loadFontFromPath(path string) (*opentype.Font, error) {
	fontData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading font file %s: %w", path, err)
	}

	font, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("parsing font file %s: %w", path, err)
	}

	return font, nil
}

// fileExists checks if a file exists
func (fm *FontManager) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetAvailableSystemFonts returns a list of available system fonts
func (fm *FontManager) GetAvailableSystemFonts() []string {
	var available []string
	for _, path := range fm.systemFontPaths {
		if fm.fileExists(path) {
			available = append(available, path)
		}
	}
	return available
}
