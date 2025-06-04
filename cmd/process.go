package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/denysvitali/id-watermark/pkg/watermark"
)

var processCmd = &cobra.Command{
	Use:   "process [input] [output]",
	Short: "Process a single image file",
	Long: `Process a single image file by adding a watermark.
	
Example:
  id-watermark process input.jpg output.jpg --company "ACME Corp"`,
	Args: cobra.ExactArgs(2),
	RunE: runProcess,
}

func init() {
	rootCmd.AddCommand(processCmd)

	// Required flags
	processCmd.Flags().StringP("company", "c", "", "company name for watermark (required)")
	processCmd.MarkFlagRequired("company")

	// Optional flags
	processCmd.Flags().StringP("font", "f", "", "path to TTF font file")
	processCmd.Flags().Float64P("size", "s", 0, "font size for watermark (10-200)")
	processCmd.Flags().Uint8P("opacity", "o", 0, "watermark opacity (0-255)")
	processCmd.Flags().Float64P("text-spacing", "x", 0, "horizontal spacing between watermarks")
	processCmd.Flags().Float64P("line-spacing", "y", 0, "vertical spacing between watermark lines")
	processCmd.Flags().IntP("quality", "q", 0, "JPEG output quality (1-100)")

	// Bind flags to viper
	viper.BindPFlag("company", processCmd.Flags().Lookup("company"))
	viper.BindPFlag("font_path", processCmd.Flags().Lookup("font"))
	viper.BindPFlag("font_size", processCmd.Flags().Lookup("size"))
	viper.BindPFlag("opacity", processCmd.Flags().Lookup("opacity"))
	viper.BindPFlag("text_spacing", processCmd.Flags().Lookup("text-spacing"))
	viper.BindPFlag("line_spacing", processCmd.Flags().Lookup("line-spacing"))
	viper.BindPFlag("quality", processCmd.Flags().Lookup("quality"))
}

func runProcess(cmd *cobra.Command, args []string) error {
	inputPath := args[0]
	outputPath := args[1]
	companyName := viper.GetString("company")

	logger.WithField("input", inputPath).WithField("output", outputPath).Info("Processing single image")

	// Create overrides map for any provided flags
	overrides := make(map[string]interface{})

	if cmd.Flags().Changed("size") {
		overrides["font_size"] = viper.GetFloat64("font_size")
	}
	if cmd.Flags().Changed("opacity") {
		overrides["opacity"] = viper.GetInt("opacity")
	}
	if cmd.Flags().Changed("text-spacing") {
		overrides["text_spacing"] = viper.GetFloat64("text_spacing")
	}
	if cmd.Flags().Changed("line-spacing") {
		overrides["line_spacing"] = viper.GetFloat64("line_spacing")
	}
	if cmd.Flags().Changed("quality") {
		overrides["quality"] = viper.GetInt("quality")
	}

	// Create watermark config
	config, err := configMgr.CreateWatermarkConfig(companyName, viper.GetString("font_path"), overrides)
	if err != nil {
		return fmt.Errorf("creating watermark config: %w", err)
	}

	// Create processor and process the image
	processor := watermark.NewProcessor(config)
	if err := processor.ProcessFile(inputPath, outputPath); err != nil {
		return fmt.Errorf("processing image: %w", err)
	}

	logger.Info("Image processed successfully")
	return nil
}
