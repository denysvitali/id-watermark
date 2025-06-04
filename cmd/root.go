package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/denysvitali/id-watermark/internal/config"
)

var (
	cfgFile   string
	configMgr *config.Manager
	logger    *logrus.Logger
	rootCmd   = &cobra.Command{
		Use:   "id-watermark",
		Short: "A tool for adding watermarks to ID cards and sensitive documents",
		Long: `ID Watermark is a CLI tool for adding diagonal watermarks to images.
It's specifically designed for ID cards and sensitive documents to prevent
unauthorized use by applying a repeating diagonal pattern of company name
and timestamp across the entire image.`,
		PersistentPreRun: initializeConfig,
	}
)

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/id-watermark/config.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")

	// Bind flags to viper
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables
func initConfig() {
	configMgr = config.NewManager()

	if err := configMgr.LoadConfig(cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
}

// initializeConfig initializes the logger and other components
func initializeConfig(cmd *cobra.Command, args []string) {
	// Initialize logger
	logger = logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(viper.GetString("log_level"))
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter
	if viper.GetBool("verbose") {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
			DisableColors:    false,
		})
	}
}
