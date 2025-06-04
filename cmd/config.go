package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/denysvitali/id-watermark/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Manage configuration files for the ID watermark tool.`,
}

var generateConfigCmd = &cobra.Command{
	Use:   "generate [filename]",
	Short: "Generate example configuration file",
	Long: `Generate an example configuration file with default values.
	
Example:
  id-watermark config generate config.yaml
  id-watermark config generate  # generates to default location`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGenerateConfig,
}

var showConfigCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration values.`,
	RunE:  runShowConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(generateConfigCmd)
	configCmd.AddCommand(showConfigCmd)
}

func runGenerateConfig(cmd *cobra.Command, args []string) error {
	var filename string
	if len(args) > 0 {
		filename = args[0]
	} else {
		filename = config.GetDefaultConfigPath()
	}

	logger.WithField("file", filename).Info("Generating configuration file")

	if err := config.GenerateExampleConfig(filename); err != nil {
		return fmt.Errorf("generating config file: %w", err)
	}

	logger.Infof("Configuration file generated: %s", filename)
	return nil
}

func runShowConfig(cmd *cobra.Command, args []string) error {
	appConfig := configMgr.GetAppConfig()

	fmt.Printf("Current Configuration:\n")
	fmt.Printf("  Font Path:         %s\n", appConfig.FontPath)
	fmt.Printf("  Font Size:         %.1f\n", appConfig.FontSize)
	fmt.Printf("  Opacity:           %d\n", appConfig.Opacity)
	fmt.Printf("  Text Spacing:      %.1f\n", appConfig.TextSpacing)
	fmt.Printf("  Line Spacing:      %.1f\n", appConfig.LineSpacing)
	fmt.Printf("  Quality:           %d\n", appConfig.Quality)
	fmt.Printf("  Log Level:         %s\n", appConfig.LogLevel)
	fmt.Printf("  Default Workers:   %d\n", appConfig.DefaultWorkers)
	fmt.Printf("  Watermark Color:   RGB(%d, %d, %d)\n",
		appConfig.WatermarkColor.R,
		appConfig.WatermarkColor.G,
		appConfig.WatermarkColor.B)

	fmt.Printf("\nSystem Font Paths:\n")
	for _, path := range appConfig.SystemFontPaths {
		fmt.Printf("  - %s\n", path)
	}

	return nil
}
