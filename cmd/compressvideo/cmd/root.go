// Package cmd implements command line interface for CompressVideo
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cccarv82/compressvideo/pkg/analyzer"
	"github.com/cccarv82/compressvideo/pkg/cache"
	"github.com/cccarv82/compressvideo/pkg/compressor"
	"github.com/cccarv82/compressvideo/pkg/ffmpeg"
	"github.com/cccarv82/compressvideo/pkg/reporter"
	"github.com/cccarv82/compressvideo/pkg/util"
	"github.com/spf13/cobra"
)

var (
	// Input/output file paths
	inputFile  string
	outputFile string

	// Compression options
	quality int     // 1-5 (1 = max compression, 5 = max quality)
	preset  string  // fast, balanced, thorough
	force   bool    // Overwrite output if exists
	verbose bool    // Verbose logging
	hwaccel string  // Hardware acceleration (none, auto, nvidia, intel, amd)
	
	// Cache options
	useCache        bool   // Whether to use analysis cache
	cacheClearExpired bool // Whether to clear expired cache entries
	cacheMaxAge     int    // Maximum age of cache entries in days

	// Logger
	logger *util.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "compressvideo",
	Short: "A smart video compression tool",
	Long: `CompressVideo - Intelligent video compression

A smart video compression CLI tool that reduces video file sizes
while maintaining the highest possible visual quality.

Examples:
  compressvideo -i input.mp4
  compressvideo -i input.mp4 -o output.mp4 -q 4 -p thorough -f -v`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return process(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Define required flags
	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input video file (required)")
	rootCmd.MarkFlagRequired("input")

	// Define optional flags
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: input-compressed.ext)")
	rootCmd.Flags().IntVarP(&quality, "quality", "q", 3, "Quality level (1-5, 1=max compression, 5=max quality)")
	rootCmd.Flags().StringVarP(&preset, "preset", "p", "balanced", "Compression preset (fast, balanced, thorough)")
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite output file if it exists")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
	rootCmd.Flags().StringVarP(&hwaccel, "hwaccel", "a", "none", "Hardware acceleration (none, auto, nvidia, intel, amd)")
	rootCmd.Flags().BoolVarP(&useCache, "use-cache", "c", false, "Whether to use analysis cache")
	rootCmd.Flags().BoolVarP(&cacheClearExpired, "clear-cache", "C", false, "Whether to clear expired cache entries")
	rootCmd.Flags().IntVarP(&cacheMaxAge, "cache-max-age", "A", 7, "Maximum age of cache entries in days")
}

// validateFlags validates the input flags
func validateFlags() error {
	// Validate input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	// Validate quality level
	if quality < 1 || quality > 5 {
		return fmt.Errorf("quality must be between 1-5 (got %d)", quality)
	}

	// Validate preset
	validPresets := map[string]bool{
		"fast":      true,
		"balanced":  true,
		"thorough":  true,
	}
	if !validPresets[preset] {
		return fmt.Errorf("preset must be one of: fast, balanced, thorough (got %s)", preset)
	}

	// Validate hardware acceleration
	validHWAccel := map[string]bool{
		"none":   true,
		"auto":   true,
		"nvidia": true,
		"intel":  true,
		"amd":    true,
	}
	if !validHWAccel[hwaccel] {
		return fmt.Errorf("hardware acceleration must be one of: none, auto, nvidia, intel, amd (got %s)", hwaccel)
	}

	// Validate output file
	if outputFile != "" {
		// Check if output file already exists and not force flag
		if _, err := os.Stat(outputFile); err == nil && !force {
			return fmt.Errorf("output file already exists (use -f to force overwrite): %s", outputFile)
		}
	} else {
		// Generate output filename if not provided
		ext := filepath.Ext(inputFile)
		base := strings.TrimSuffix(inputFile, ext)
		outputFile = base + "-compressed" + ext
	}

	return nil
}

// process runs the main compression process
func process(cmd *cobra.Command, args []string) error {
	// Configure logger
	logger = util.NewLogger(verbose)
	logger.Title("CompressVideo - Smart Video Compression")

	// Validate required flags
	err := validateFlags()
	if err != nil {
		return err
	}
	
	// Resolve output file if not specified
	if outputFile == "" {
		dir := filepath.Dir(inputFile)
		ext := filepath.Ext(inputFile)
		base := filepath.Base(inputFile)
		base = strings.TrimSuffix(base, ext)
		outputFile = filepath.Join(dir, base+"-compressed"+ext)
	}

	// Check if input file is a directory
	fileInfo, err := os.Stat(inputFile)
	if err != nil {
		return fmt.Errorf("error accessing input file: %w", err)
	}

	// Initialize cache if enabled
	var videoCache *cache.VideoAnalysisCache
	if useCache {
		videoCache, err = cache.NewVideoAnalysisCache(logger)
		if err != nil {
			logger.Warning("Failed to initialize cache: %v", err)
			logger.Warning("Continuing without cache...")
			useCache = false
		} else {
			// Set cache options
			videoCache.SetMaxAge(cacheMaxAge * 24) // Convert days to hours
			
			// Clean expired cache entries if requested
			if cacheClearExpired {
				cleaned, err := videoCache.CleanExpiredEntries()
				if err != nil {
					logger.Warning("Failed to clean expired cache entries: %v", err)
				} else if cleaned > 0 {
					logger.Info("Cleaned %d expired cache entries", cleaned)
				}
			}
			
			// Show cache statistics
			total, valid, err := videoCache.GetCacheStats()
			if err != nil {
				logger.Warning("Failed to get cache statistics: %v", err)
			} else {
				logger.Info("Cache status: %d total entries, %d valid entries", total, valid)
			}
		}
		
		// Clean up cache when done
		defer func() {
			if videoCache != nil {
				videoCache.Close()
			}
		}()
	}

	// Process directory or single file
	if fileInfo.IsDir() {
		// Directory provided
		logger.Section("Processing Directory")
		logger.Field("Input Directory", inputFile)
		
		// If no output directory provided, create one with _compressed suffix
		if outputFile == "" {
			outputFile = inputFile + "_compressed"
		}
		
		logger.Field("Output Directory", outputFile)
		
		// Process the directory
		return processDirectory(inputFile, outputFile, videoCache)
	}
	
	// Single file processing
	logger.Section("Processing Video")
	logger.Field("Input File", inputFile)
	logger.Field("Output File", outputFile)
	logger.Field("Quality Level", "%d/5", quality)
	logger.Field("Preset", preset)
	logger.Field("Hardware Acceleration", hwaccel)
	
	// Process the file
	return processSingleFile(inputFile, outputFile, videoCache)
}

// Função auxiliar para verificar se um slice contém um determinado valor
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// displayVideoInfo shows detailed information about the video file
func displayVideoInfo(videoFile *ffmpeg.VideoFile) {
	logger.Section("Video Information")
	
	// Format size for better display
	sizeStr := formatSize(videoFile.Size)
	
	logger.Field("Format", "%s", videoFile.Format)
	logger.Field("Size", "%s", sizeStr)
	logger.Field("Duration", "%.2f seconds", videoFile.Duration)
	
	// Show bitrate if available
	if videoFile.BitRate > 0 {
		logger.Field("Bitrate", "%s", formatBitrate(videoFile.BitRate))
	}
	
	// Video stream info
	logger.Info("\nVideo Stream:")
	logger.Field("  Codec", "%s", videoFile.VideoInfo.Codec)
	logger.Field("  Resolution", "%dx%d", videoFile.VideoInfo.Width, videoFile.VideoInfo.Height)
	logger.Field("  Frame Rate", "%.2f fps", videoFile.VideoInfo.FPS)
	
	if videoFile.VideoInfo.BitRate > 0 {
		logger.Field("    Video Bitrate", "%s", formatBitrate(videoFile.VideoInfo.BitRate))
	}
	
	logger.Field("  Pixel Format", "%s", videoFile.VideoInfo.PixelFormat)
	if videoFile.VideoInfo.ProfileLevel != "" {
		logger.Field("  Profile", "%s", videoFile.VideoInfo.ProfileLevel)
	}
	logger.Field("  HDR", "%t", videoFile.VideoInfo.IsHDR)
	
	// Audio stream info
	if len(videoFile.AudioInfo) > 0 {
		logger.Info("\nAudio Streams: %d", len(videoFile.AudioInfo))
		for i, audio := range videoFile.AudioInfo {
			logger.Info("  Stream #%d:", i+1)
			logger.Field("    Codec", "%s", audio.Codec)
			logger.Field("    Channels", "%d", audio.Channels)
			logger.Field("    Sample Rate", "%d Hz", audio.SampleRate)
			
			if audio.BitRate > 0 {
				bitrateStr := formatBitrate(audio.BitRate)
				logger.Field("    Bitrate", "%s", bitrateStr)
			}
			
			if audio.Language != "" {
				logger.Field("    Language", "%s", audio.Language)
			}
		}
	}
}

// displayAnalysisResults shows the content analysis results
func displayAnalysisResults(analysis *analyzer.VideoAnalysis) {
	logger.Section("Content Analysis Results")
	
	// Content type with emoji indicator
	contentEmoji := getContentTypeEmoji(analysis.ContentType.String())
	logger.Field("Content Type", "%s %s", contentEmoji, analysis.ContentType.String())
	
	// Motion complexity with emoji indicator
	motionEmoji := getMotionComplexityEmoji(analysis.MotionComplexity.String())
	logger.Field("Motion Complexity", "%s %s", motionEmoji, analysis.MotionComplexity.String())
	
	logger.Field("Scene Changes", "%d", analysis.SceneChanges)
	logger.Field("Frame Complexity", "%.2f", analysis.FrameComplexity)
	logger.Field("Spatial Complexity", "%.2f", analysis.SpatialComplexity)
	logger.Field("Recommended Codec", "%s", analysis.RecommendedCodec)
	
	bitrateStr := formatBitrate(analysis.OptimalBitrate)
	logger.Field("Optimal Bitrate", "%s", bitrateStr)
	
	resolutionType := "SD"
	if analysis.IsUHDContent {
		resolutionType = "UHD/4K"
	} else if analysis.IsHDContent {
		resolutionType = "HD"
	}
	logger.Field("Resolution Type", "%s", resolutionType)
	
	// Show compression potential
	logger.Field("Est. Compression Potential", "%.0f%%", float64(analysis.CompressionPotential))
}

// getContentTypeEmoji returns an appropriate emoji for the content type
func getContentTypeEmoji(contentType string) string {
	switch contentType {
	case "Screencast":
		return "💻"
	case "Animation":
		return "🎨"
	case "Gaming":
		return "🎮"
	case "Sports Action":
		return "⚽"
	case "Live Action":
		return "🎬"
	case "Documentary":
		return "🌍"
	default:
		return "📹"
	}
}

// getMotionComplexityEmoji returns an appropriate emoji for motion complexity
func getMotionComplexityEmoji(complexity string) string {
	switch complexity {
	case "Low":
		return "🐢"
	case "Medium":
		return "🚶"
	case "High":
		return "🏃"
	case "Very High":
		return "🚀"
	default:
		return "🚶"
	}
}

// formatSize returns a human-readable file size
func formatSize(sizeBytes int64) string {
	const unit = 1024
	if sizeBytes < unit {
		return fmt.Sprintf("%d B", sizeBytes)
	}
	
	div, exp := int64(unit), 0
	for n := sizeBytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	switch exp {
	case 0:
		return fmt.Sprintf("%.2f KB", float64(sizeBytes)/float64(div*unit/unit))
	case 1:
		return fmt.Sprintf("%.2f MB", float64(sizeBytes)/float64(div*unit/unit))
	case 2:
		return fmt.Sprintf("%.2f GB", float64(sizeBytes)/float64(div*unit/unit))
	}
	
	return fmt.Sprintf("%.2f TB", float64(sizeBytes)/float64(div*unit/unit))
}

// formatBitrate returns a human-readable bitrate
func formatBitrate(bitrate int64) string {
	if bitrate >= 1000000 {
		return fmt.Sprintf("%.2f Mbps", float64(bitrate)/1000000)
	} else {
		return fmt.Sprintf("%.2f kbps", float64(bitrate)/1000)
	}
}

// getFileExtension returns the file extension including the dot
func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}

// processDirectory processes a directory of video files
func processDirectory(inputDir, outputDir string, videoCache *cache.VideoAnalysisCache) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// List files in the input directory
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read input directory: %w", err)
	}

	// Count of video files found
	videoCount := 0

	// Process each file
	for _, file := range files {
		if file.IsDir() {
			continue // Skip subdirectories for now
		}

		// Check if it's a video file
		fileName := file.Name()
		inputPath := filepath.Join(inputDir, fileName)
		if !isVideoFile(fileName) {
			continue
		}

		videoCount++

		// Define output path
		outputFileName := strings.TrimSuffix(fileName, filepath.Ext(fileName)) + "-compressed" + filepath.Ext(fileName)
		outputPath := filepath.Join(outputDir, outputFileName)

		// Check if output file exists and handle overwrite
		if _, err := os.Stat(outputPath); err == nil && !force {
			logger.Warning("Skipping %s: output file already exists (use -f to force overwrite)", fileName)
			continue
		}

		// Process the video file
		logger.Info("Processing video %s...", fileName)
		err = processSingleFile(inputPath, outputPath, videoCache)
		if err != nil {
			logger.Error("Failed to process %s: %v", fileName, err)
			continue
		}
	}

	if videoCount == 0 {
		logger.Warning("No video files found in directory")
	} else {
		logger.Success("Processed %d video files", videoCount)
	}

	return nil
}

// processSingleFile processes a single video file
func processSingleFile(inputFile, outputFile string, videoCache *cache.VideoAnalysisCache) error {
	if verbose {
		logger.Debug("File Info:")
		logger.Debug("  Input Path: %s", inputFile)
		logger.Debug("  Output Path: %s", outputFile)
		logger.Debug("  Quality Level: %d", quality)
		logger.Debug("  Compression Preset: %s", preset)
		logger.Debug("  Force Overwrite: %v", force)
		logger.Debug("  Hardware Acceleration: %s", hwaccel)
	}

	// Check if output file exists and handle overwrite
	if _, err := os.Stat(outputFile); err == nil && !force {
		return fmt.Errorf("output file already exists (use -f to force overwrite): %s", outputFile)
	}

	// Configure FFmpeg options
	options := &ffmpeg.Options{
		Quality: quality,
		Preset:  preset,
		HWAccel: hwaccel,
	}

	// Create FFmpeg instance
	ffmpegInstance := ffmpeg.NewFFmpeg(inputFile, outputFile, options, logger)

	// Check hardware acceleration
	if hwaccel != "none" {
		accelerators, err := ffmpegInstance.DetectAvailableHWAccelerators()
		if err != nil {
			logger.Warning("Não foi possível detectar aceleradores de hardware: %v", err)
			logger.Warning("Usando processamento via CPU (sem aceleração)")
			options.HWAccel = "none"
		} else if len(accelerators) == 0 {
			logger.Warning("Nenhum acelerador de hardware detectado. Usando CPU.")
			options.HWAccel = "none"
		} else if hwaccel == "auto" {
			// Choose the best available accelerator
			// Priority: nvidia > intel > amd > CPU
			chosenAccel := "none"
			if contains(accelerators, "nvidia") {
				chosenAccel = "nvidia"
			} else if contains(accelerators, "intel") {
				chosenAccel = "intel"
			} else if contains(accelerators, "amd") {
				chosenAccel = "amd"
			}
			
			if chosenAccel != "none" {
				logger.Info("Aceleração de hardware ativada: %s (detectado automaticamente)", chosenAccel)
				options.HWAccel = chosenAccel
			} else {
				logger.Warning("Nenhum acelerador de hardware utilizável detectado. Usando CPU.")
				options.HWAccel = "none"
			}
		} else if !contains(accelerators, hwaccel) {
			logger.Warning("O acelerador de hardware '%s' não está disponível. Usando CPU.", hwaccel)
			logger.Warning("Aceleradores disponíveis: %v", accelerators)
			options.HWAccel = "none"
		} else {
			logger.Info("Usando aceleração de hardware: %s", hwaccel)
		}
	}

	// Create analyzer
	contentAnalyzer := analyzer.NewContentAnalyzer(ffmpegInstance, logger)

	// Variables to hold video info and analysis
	var videoFile *ffmpeg.VideoFile
	var analysis *analyzer.VideoAnalysis
	var err error
	var cacheUsed bool

	// Try to get from cache if enabled
	if videoCache != nil && useCache {
		logger.Info("Checking analysis cache...")
		analysis, videoFile, cacheUsed, err = videoCache.Get(inputFile)
		if err != nil {
			logger.Warning("Error reading from cache: %v", err)
		}
		
		if cacheUsed {
			logger.Info("Using cached analysis for %s", filepath.Base(inputFile))
		} else {
			logger.Info("No valid cache entry found, analyzing video...")
		}
	}

	// If not using cache or cache miss, perform analysis
	if !cacheUsed {
		// Get video info
		videoFile, err = ffmpegInstance.GetVideoInfo(inputFile)
		if err != nil {
			return fmt.Errorf("falha ao obter informações do vídeo: %v", err)
		}

		// Display video info
		displayVideoInfo(videoFile)

		// Analyze video
		analysis, err = contentAnalyzer.AnalyzeVideo(videoFile)
		if err != nil {
			return fmt.Errorf("falha ao analisar vídeo: %v", err)
		}

		// Store in cache for future use if cache is enabled
		if videoCache != nil && useCache {
			err = videoCache.Put(inputFile, analysis, videoFile)
			if err != nil {
				logger.Warning("Failed to cache analysis: %v", err)
			} else {
				logger.Debug("Stored analysis in cache for %s", filepath.Base(inputFile))
			}
		}
	} else {
		// Still display info even when using cached data
		displayVideoInfo(videoFile)
	}

	// Display analysis results
	displayAnalysisResults(analysis)

	// Get compression settings based on analysis
	compressionSettings, err := contentAnalyzer.GetCompressionSettings(analysis, quality)
	if err != nil {
		logger.Error("Failed to determine compression settings: %v", err)
		return err
	}

	// Display recommended settings
	logger.Info("Recommended compression settings:")
	for key, value := range compressionSettings {
		logger.Info("  %s: %s", key, value)
	}

	// Create a new video compressor
	videoCompressor := compressor.NewVideoCompressor(ffmpegInstance, contentAnalyzer, logger)

	// Initialize the report generator
	reportGenerator := reporter.NewReportGenerator(logger, ffmpegInstance)

	// Create initial report with basic information
	report := reportGenerator.CreateReport(inputFile, outputFile, videoFile, analysis)

	// Create a more advanced progress tracker
	progressOptions := util.ProgressTrackerOptions{
		Total:          100,
		Description:    "Compressing Video",
		Logger:         logger,
		ShowPercentage: true,
		ShowSpeed:      true,
	}

	progressBar := util.NewProgressTrackerWithOptions(progressOptions)

	// Set up status callback for real-time updates
	progressBar.SetStatusCallback(func(progress int64, timeRemaining time.Duration, rate float64) {
		logger.Debug("Compression Status: %d%% complete, %.1f seconds remaining", 
			progress, timeRemaining.Seconds())
	})

	// Start compression
	logger.Section("Compression Process")
	logger.Info("Starting compression process...")
	startTime := time.Now()

	result, err := videoCompressor.CompressVideo(
		inputFile,
		outputFile, 
		analysis,
		compressionSettings,
		quality,
		preset,
		progressBar,
	)

	if err != nil {
		logger.Error("Compression failed: %v", err)
		return err
	}

	// Ensure progress bar is completed
	progressBar.Finish()

	// Complete the report with results
	report = reportGenerator.FinalizeReport(report, result)

	// Display comprehensive report to console
	reportGenerator.DisplayReportToConsole(report)

	// Save report to file
	reportPath, err := reportGenerator.SaveReportToFile(report)
	if err != nil {
		logger.Warning("Failed to save report to file: %v", err)
	} else {
		logger.Info("Compression report saved to: %s", reportPath)
	}

	// Display a user-friendly completion message
	processingTime := time.Since(startTime).Round(time.Second)
	savings := fmt.Sprintf("%.1f%%", result.SavedSpacePercent)

	logger.Success("Video compression completed successfully!")
	logger.Info("Compressed %s in %s, saving %s of space", 
		filepath.Base(inputFile), 
		processingTime, 
		savings)

	// Note about cache usage for user awareness
	if cacheUsed {
		logger.Info("Analysis time saved by using cache!")
	}

	return nil
}

// isVideoFile checks if a file is a video based on its extension
func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	videoExts := []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".mpg", ".mpeg", ".3gp"}
	return contains(videoExts, ext)
} 