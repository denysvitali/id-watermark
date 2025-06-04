// Package config provides configuration management for the ID watermark tool
package config

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"

	"github.com/denysvitali/id-watermark/pkg/watermark"
)

// AppConfig represents the application configuration
type AppConfig struct {
	// Default values
	FontPath    string  `mapstructure:"font_path"`
	FontSize    float64 `mapstructure:"font_size"`
	Opacity     uint8   `mapstructure:"opacity"`
	TextSpacing float64 `mapstructure:"text_spacing"`
	LineSpacing float64 `mapstructure:"line_spacing"`
	Quality     int     `mapstructure:"quality"`
	LogLevel    string  `mapstructure:"log_level"`

	// Watermark color
	WatermarkColor struct {
		R uint8 `mapstructure:"r"`
		G uint8 `mapstructure:"g"`
		B uint8 `mapstructure:"b"`
	} `mapstructure:"watermark_color"`

	// System font paths
	SystemFontPaths []string `mapstructure:"system_font_paths"`

	// Batch processing
	DefaultWorkers int `mapstructure:"default_workers"`
}

// Manager handles configuration loading and management
type Manager struct {
	config *AppConfig
	viper  *viper.Viper
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	return &Manager{
		config: &AppConfig{},
		viper:  v,
	}
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	v.SetDefault("font_path", "./DejaVuSans.ttf")
	v.SetDefault("font_size", 40.0)
	v.SetDefault("opacity", 40)
	v.SetDefault("text_spacing", 30.0)
	v.SetDefault("line_spacing", 30.0)
	v.SetDefault("quality", 95)
	v.SetDefault("log_level", "info")
	v.SetDefault("default_workers", 4)

	// Default watermark color (gray)
	v.SetDefault("watermark_color.r", 150)
	v.SetDefault("watermark_color.g", 150)
	v.SetDefault("watermark_color.b", 150)

	// Default system font paths
	v.SetDefault("system_font_paths", []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
		"/System/Library/Fonts/Arial.ttf",
		"/System/Library/Fonts/Helvetica.ttc",
		"/Windows/Fonts/arial.ttf",
		"/Windows/Fonts/Arial.ttf",
		"/usr/share/fonts/TTF/arial.ttf",
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
	})
}

// LoadConfig loads configuration from file and environment
func (m *Manager) LoadConfig(configFile string) error {
	if configFile != "" {
		m.viper.SetConfigFile(configFile)
	} else {
		// Look for config in standard locations
		m.viper.SetConfigName("id-watermark")
		m.viper.SetConfigType("yaml")
		m.viper.AddConfigPath(".")
		m.viper.AddConfigPath("$HOME/.config/id-watermark")
		m.viper.AddConfigPath("/etc/id-watermark")
	}

	// Environment variable support
	m.viper.SetEnvPrefix("WATERMARK")
	m.viper.AutomaticEnv()

	// Read config file if it exists
	if err := m.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("reading config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}

	// Unmarshal into struct
	if err := m.viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	return nil
}

// GetAppConfig returns the loaded application configuration
func (m *Manager) GetAppConfig() *AppConfig {
	return m.config
}

// CreateWatermarkConfig creates a watermark configuration from app config and parameters
func (m *Manager) CreateWatermarkConfig(companyName, fontPath string, overrides map[string]interface{}) (*watermark.Config, error) {
	// Apply any overrides
	for key, value := range overrides {
		m.viper.Set(key, value)
	}

	// Use provided font path or fall back to config
	if fontPath == "" {
		fontPath = m.viper.GetString("font_path")
	}

	// Load font
	fontManager := watermark.NewFontManager()
	fontManager.SetSystemFontPaths(m.viper.GetStringSlice("system_font_paths"))

	font, err := fontManager.LoadFont(fontPath)
	if err != nil {
		return nil, fmt.Errorf("loading font: %w", err)
	}

	// Create watermark config
	config := &watermark.Config{
		CompanyName: companyName,
		Timestamp:   time.Now(),
		FontSize:    m.viper.GetFloat64("font_size"),
		Opacity:     uint8(m.viper.GetInt("opacity")),
		Angle:       0, // TODO: make configurable
		Font:        font,
		TextSpacing: m.viper.GetFloat64("text_spacing"),
		LineSpacing: m.viper.GetFloat64("line_spacing"),
		Quality:     m.viper.GetInt("quality"),
		WatermarkColor: color.RGBA{
			R: uint8(m.viper.GetInt("watermark_color.r")),
			G: uint8(m.viper.GetInt("watermark_color.g")),
			B: uint8(m.viper.GetInt("watermark_color.b")),
			A: uint8(m.viper.GetInt("opacity")),
		},
	}

	return config, nil
}

// SaveConfig saves the current configuration to a file
func (m *Manager) SaveConfig(filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	return m.viper.WriteConfigAs(filename)
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./id-watermark.yaml"
	}
	return filepath.Join(homeDir, ".config", "id-watermark", "config.yaml")
}

// GenerateExampleConfig creates an example configuration file
func GenerateExampleConfig(filename string) error {
	manager := NewManager()

	// Set some example values
	manager.viper.Set("font_size", 45.0)
	manager.viper.Set("opacity", 60)
	manager.viper.Set("text_spacing", 35.0)
	manager.viper.Set("line_spacing", 35.0)
	manager.viper.Set("quality", 90)

	return manager.SaveConfig(filename)
}
