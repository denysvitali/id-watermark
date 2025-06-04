package watermark

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// BatchProcessor handles batch processing of multiple images
type BatchProcessor struct {
	processor *Processor
	workers   int
	recursive bool
	logger    *logrus.Logger
}

// BatchOptions configures batch processing behavior
type BatchOptions struct {
	Workers   int
	Recursive bool
	Logger    *logrus.Logger
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(config *Config, options *BatchOptions) (*BatchProcessor, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	processor := NewProcessor(config)

	workers := options.Workers
	if workers <= 0 {
		workers = 4
	}

	logger := options.Logger
	if logger == nil {
		logger = logrus.New()
	}

	return &BatchProcessor{
		processor: processor,
		workers:   workers,
		recursive: options.Recursive,
		logger:    logger,
	}, nil
}

// ProcessDirectory processes all images in a directory
func (bp *BatchProcessor) ProcessDirectory(inputDir, outputDir string) (*BatchResult, error) {
	// Find all image files
	imageFiles, err := bp.findImageFiles(inputDir)
	if err != nil {
		return nil, fmt.Errorf("finding image files: %w", err)
	}

	if len(imageFiles) == 0 {
		return nil, fmt.Errorf("no image files found in %s", inputDir)
	}

	bp.logger.WithFields(logrus.Fields{
		"input_dir":  inputDir,
		"output_dir": outputDir,
		"files":      len(imageFiles),
		"workers":    bp.workers,
		"recursive":  bp.recursive,
	}).Info("Starting batch processing")

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	// Process files
	result := bp.processFiles(imageFiles, inputDir, outputDir)

	bp.logger.WithFields(logrus.Fields{
		"success": result.SuccessCount,
		"errors":  result.ErrorCount,
		"total":   result.TotalCount,
	}).Info("Batch processing completed")

	return result, nil
}

// BatchResult contains the results of batch processing
type BatchResult struct {
	TotalCount   int
	SuccessCount int
	ErrorCount   int
	Errors       []BatchError
}

// BatchError represents an error that occurred during batch processing
type BatchError struct {
	FilePath string
	Error    error
}

// job represents a single processing job
type job struct {
	inputPath  string
	outputPath string
}

// jobResult represents the result of a single job
type jobResult struct {
	inputPath string
	err       error
}

// processFiles processes a list of image files using worker goroutines
func (bp *BatchProcessor) processFiles(imageFiles []string, inputDir, outputDir string) *BatchResult {
	jobs := make(chan job, len(imageFiles))
	results := make(chan jobResult, len(imageFiles))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < bp.workers; i++ {
		wg.Add(1)
		go bp.worker(jobs, results, &wg)
	}

	// Send jobs
	for _, file := range imageFiles {
		relPath, err := filepath.Rel(inputDir, file)
		if err != nil {
			relPath = filepath.Base(file)
		}
		outputPath := filepath.Join(outputDir, relPath)

		// Create output subdirectory if needed
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			bp.logger.WithError(err).WithField("path", filepath.Dir(outputPath)).Warn("Failed to create output directory")
			continue
		}

		jobs <- job{
			inputPath:  file,
			outputPath: outputPath,
		}
	}
	close(jobs)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	result := &BatchResult{
		TotalCount: len(imageFiles),
		Errors:     make([]BatchError, 0),
	}

	for jobResult := range results {
		if jobResult.err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, BatchError{
				FilePath: jobResult.inputPath,
				Error:    jobResult.err,
			})
			bp.logger.WithError(jobResult.err).WithField("file", jobResult.inputPath).Error("Failed to process image")
		} else {
			result.SuccessCount++
			bp.logger.WithField("file", jobResult.inputPath).Debug("Successfully processed image")
		}
	}

	return result
}

// worker processes jobs from the job channel
func (bp *BatchProcessor) worker(jobs <-chan job, results chan<- jobResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		err := bp.processor.ProcessFile(job.inputPath, job.outputPath)
		results <- jobResult{
			inputPath: job.inputPath,
			err:       err,
		}
	}
}

// findImageFiles finds all image files in the given directory
func (bp *BatchProcessor) findImageFiles(inputDir string) ([]string, error) {
	var imageFiles []string
	supportedExts := []string{".jpg", ".jpeg", ".png"}

	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip subdirectories if not recursive
			if !bp.recursive && path != inputDir {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, supportedExt := range supportedExts {
			if ext == supportedExt {
				imageFiles = append(imageFiles, path)
				break
			}
		}

		return nil
	})

	return imageFiles, err
}
