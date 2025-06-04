package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/denysvitali/id-watermark/pkg/watermark"
)

var batchCmd = &cobra.Command{
	Use:   "batch [input-dir] [output-dir]",
	Short: "Process multiple images in a directory",
	Long: `Process multiple images in a directory by adding watermarks.
	
Example:
  id-watermark batch ./images ./watermarked --company "ACME Corp" --workers 8 --recursive`,
	Args: cobra.ExactArgs(2),
	RunE: runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)

	// Required flags
	batchCmd.Flags().StringP("company", "c", "", "company name for watermark (required)")
	batchCmd.MarkFlagRequired("company")

	// Optional flags
	batchCmd.Flags().StringP("font", "f", "", "path to TTF font file")
	batchCmd.Flags().Float64P("size", "s", 0, "font size for watermark (10-200)")
	batchCmd.Flags().Uint8P("opacity", "o", 0, "watermark opacity (0-255)")
	batchCmd.Flags().Float64P("text-spacing", "x", 0, "horizontal spacing between watermarks")
	batchCmd.Flags().Float64P("line-spacing", "y", 0, "vertical spacing between watermark lines")
	batchCmd.Flags().IntP("quality", "q", 0, "JPEG output quality (1-100)")

	// Batch-specific flags
	batchCmd.Flags().IntP("workers", "w", 0, "number of parallel workers")
	batchCmd.Flags().BoolP("recursive", "r", false, "process subdirectories recursively")

	// Bind flags to viper
	viper.BindPFlag("company", batchCmd.Flags().Lookup("company"))
	viper.BindPFlag("font_path", batchCmd.Flags().Lookup("font"))
	viper.BindPFlag("font_size", batchCmd.Flags().Lookup("size"))
	viper.BindPFlag("opacity", batchCmd.Flags().Lookup("opacity"))
	viper.BindPFlag("text_spacing", batchCmd.Flags().Lookup("text-spacing"))
	viper.BindPFlag("line_spacing", batchCmd.Flags().Lookup("line-spacing"))
	viper.BindPFlag("quality", batchCmd.Flags().Lookup("quality"))
	viper.BindPFlag("workers", batchCmd.Flags().Lookup("workers"))
	viper.BindPFlag("recursive", batchCmd.Flags().Lookup("recursive"))
}

func runBatch(cmd *cobra.Command, args []string) error {
	inputDir := args[0]
	outputDir := args[1]
	companyName := viper.GetString("company")

	logger.WithField("input_dir", inputDir).WithField("output_dir", outputDir).Info("Starting batch processing")

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

	// Get batch options
	workers := viper.GetInt("workers")
	if workers == 0 {
		workers = configMgr.GetAppConfig().DefaultWorkers
	}

	batchOptions := &watermark.BatchOptions{
		Workers:   workers,
		Recursive: viper.GetBool("recursive"),
		Logger:    logger,
	}

	// Create batch processor
	batchProcessor, err := watermark.NewBatchProcessor(config, batchOptions)
	if err != nil {
		return fmt.Errorf("creating batch processor: %w", err)
	}

	// Process directory
	result, err := batchProcessor.ProcessDirectory(inputDir, outputDir)
	if err != nil {
		return fmt.Errorf("processing directory: %w", err)
	}

	// Report results
	if result.ErrorCount > 0 {
		logger.Warnf("Completed with %d errors out of %d files", result.ErrorCount, result.TotalCount)
		for _, batchErr := range result.Errors {
			logger.WithError(batchErr.Error).WithField("file", batchErr.FilePath).Error("Processing failed")
		}
	} else {
		logger.Infof("Successfully processed all %d files", result.SuccessCount)
	}

	return nil
}
