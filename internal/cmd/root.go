// Package cmd implements command line interface for CompressVideo
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cccarv82/compressvideo/pkg/analyzer"
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
	quality int  // 1-5 (1 = max compression, 5 = max quality)
	preset  string  // fast, balanced, thorough
	force   bool    // Overwrite output if exists
	verbose bool    // Verbose logging

	// Logger
	logger *util.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "compressvideo",
	Short: "Um compressor de vídeo simples",
	Long: `CompressVideo é uma ferramenta de linha de comando para comprimir vídeos
usando o FFmpeg com configurações ótimas para diversos casos de uso.

O aplicativo tenta fornecer um bom equilíbrio entre qualidade e tamanho do arquivo,
escolhendo automaticamente os melhores parâmetros de compressão para cada vídeo.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger = util.NewLogger(verbose)
		
		// Executar a compressão
		if err := executeCompression(); err != nil {
			logger.Error("Erro: %v", err)
			os.Exit(1)
		}
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
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: input_compressed.ext)")
	rootCmd.Flags().IntVarP(&quality, "quality", "q", 3, "Quality level (1-5, 1=max compression, 5=max quality)")
	rootCmd.Flags().StringVarP(&preset, "preset", "p", "balanced", "Compression preset (fast, balanced, thorough)")
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite output file if it exists")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
	
	// Add repair-ffmpeg command
	rootCmd.AddCommand(repairFFmpegCmd)
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
		outputFile = filepath.Join(filepath.Dir(inputFile), base+"_compressed"+ext)
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
		outputFile = filepath.Join(dir, base+"_compressed"+ext)
	}
	
	logger.Section("Processing Video")
	logger.Field("Input File", "%s", inputFile)
	logger.Field("Output File", "%s", outputFile)
	logger.Field("Quality Level", "%d/5", quality)
	logger.Field("Preset", "%s", preset)
	
	if verbose {
		logger.Debug("File Info:")
		logger.Debug("  Input Path: %s", inputFile)
		logger.Debug("  Output Path: %s", outputFile)
		logger.Debug("  Quality Level: %d", quality)
		logger.Debug("  Compression Preset: %s", preset)
		logger.Debug("  Force Overwrite: %v", force)
	}
	
	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}
	
	// Check if output file exists and handle overwrite
	if _, err := os.Stat(outputFile); err == nil && !force {
		return fmt.Errorf("output file already exists (use -f to force overwrite): %s", outputFile)
	}
	
	// Initialize FFmpeg
	logger.Info("Initializing FFmpeg...")
	ffmpegInfo, err := util.EnsureFFmpeg(logger)
	if err != nil {
		logger.Error("Failed to initialize FFmpeg: %v", err)
		return err
	}
	
	// Usar o caminho do FFmpeg obtido
	logger.Debug("Using FFmpeg path: %s", ffmpegInfo.Path)
	logger.Debug("Using FFprobe path: %s", ffmpegInfo.FFprobePath)

	// Criar uma instância de FFmpeg
	ffmpegUtil := ffmpeg.NewFFmpeg("", "", &ffmpeg.Options{}, logger)
	
	// Get video information
	logger.Info("Extracting video metadata...")
	videoFile, err := ffmpegUtil.GetVideoInfo(inputFile)
	if err != nil {
		logger.Error("Failed to get video information: %v", err)
		return err
	}
	
	// Display video information
	displayVideoInfo(videoFile)
	
	// Initialize content analyzer
	contentAnalyzer := analyzer.NewContentAnalyzer(ffmpegUtil, logger)
	
	// Analyze video content
	logger.Section("Video Analysis")
	logger.Info("Analyzing video content...")
	analysis, err := contentAnalyzer.AnalyzeVideo(videoFile)
	if err != nil {
		logger.Error("Video analysis failed: %v", err)
		return err
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
	videoCompressor := compressor.NewVideoCompressor(ffmpegUtil, contentAnalyzer, logger)
	
	// Initialize the report generator
	reportGenerator := reporter.NewReportGenerator(logger, ffmpegUtil)
	
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
	
	return nil
}

// displayVideoInfo shows detailed information about the video file
func displayVideoInfo(videoFile *ffmpeg.VideoFile) {
	logger.Section("Video Information")
	
	// Format size for better display
	sizeStr := formatSize(videoFile.Size)
	
	logger.Field("Format", "%s", videoFile.Format)
	logger.Field("Size", "%s", sizeStr)
	logger.Field("Duration", "%.2f seconds", videoFile.Duration)
	
	if videoFile.BitRate > 0 {
		bitrateStr := formatBitrate(videoFile.BitRate)
		logger.Field("Bitrate Total", "%s", bitrateStr)
	} else {
		logger.Field("Bitrate Total", "Desconhecido")
	}
	
	// Video stream
	logger.Section("Stream de Vídeo")
	logger.Field("Codec", "%s", videoFile.VideoInfo.Codec)
	logger.Field("Resolução", "%dx%d", videoFile.VideoInfo.Width, videoFile.VideoInfo.Height)
	logger.Field("FPS", "%.2f", videoFile.VideoInfo.FPS)
	
	if videoFile.VideoInfo.BitRate > 0 {
		bitrateStr := formatBitrate(videoFile.VideoInfo.BitRate)
		logger.Field("Bitrate do Vídeo", "%s", bitrateStr)
	}
	
	logger.Field("Pixel Format", "%s", videoFile.VideoInfo.PixelFormat)
	if videoFile.VideoInfo.ProfileLevel != "" {
		logger.Field("Profile", "%s", videoFile.VideoInfo.ProfileLevel)
	}
	logger.Field("HDR", "%t", videoFile.VideoInfo.IsHDR)
	
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

// repairFFmpegCmd represents the command to repair the FFmpeg installation
var repairFFmpegCmd = &cobra.Command{
	Use:   "repair-ffmpeg",
	Short: "Repair FFmpeg installation",
	Long:  `Repair the FFmpeg installation used by CompressVideo.

This command is useful when you encounter issues with FFmpeg, such as:
- "Failed to get video information" errors
- Exit status errors with FFmpeg or FFprobe
- Missing codecs or format support

The repair process will:
1. Remove the existing FFmpeg installation
2. Download a fresh copy of FFmpeg
3. Verify the installation works correctly`,
	
	Run: func(cmd *cobra.Command, args []string) {
		logger = util.NewLogger(true) // Force verbose for repair
		
		logger.Title("CompressVideo - FFmpeg Repair Tool")
		logger.Info("Starting FFmpeg repair process...")
		
		info, err := util.RepairFFmpeg(logger)
		if err != nil {
			logger.Fatal("Failed to repair FFmpeg: %v", err)
		}
		
		// Test the repaired FFmpeg with a simple command
		logger.Info("Testing repaired FFmpeg...")
		testCmd := exec.Command(info.Path, "-version")
		output, err := testCmd.CombinedOutput()
		if err != nil {
			logger.Error("FFmpeg test failed: %v", err)
			logger.Error("Output: %s", string(output))
			os.Exit(1)
		}
		
		if info.FFprobePath != "" {
			testProbeCmd := exec.Command(info.FFprobePath, "-version")
			probeOutput, err := testProbeCmd.CombinedOutput()
			if err != nil {
				logger.Error("FFprobe test failed: %v", err)
				logger.Error("Output: %s", string(probeOutput))
				os.Exit(1)
			}
		}
		
		logger.Success("FFmpeg repair completed successfully!")
		logger.Info("FFmpeg version: %s", info.Version)
		logger.Info("FFmpeg path: %s", info.Path)
		
		if info.FFprobePath != "" {
			logger.Info("FFprobe path: %s", info.FFprobePath)
		}
		
		logger.Info("\nYou can now use CompressVideo normally.")
	},
}

func executeCompression() error {
	// Configure logger
	logger = util.NewLogger(verbose)
	logger.Title("CompressVideo v%s", util.Version)
	logger.Info("Iniciando compressão de vídeo")
	
	// Validate input file
	logger.Info("Arquivo de entrada: %s", inputFile)
	
	// If output file is not specified, use default naming
	if outputFile == "" {
		ext := filepath.Ext(inputFile)
		outputFile = strings.TrimSuffix(inputFile, ext) + "_compressed" + ext
	}
	
	logger.Info("Arquivo de saída: %s", outputFile)
	
	// Check if output file exists
	if _, err := os.Stat(outputFile); err == nil && !force {
		return fmt.Errorf("o arquivo de saída já existe. Use a flag -f para sobrescrever")
	}
	
	// Set compression options
	logger.Info("Qualidade: %d/5", quality)
	logger.Info("Preset: %s", preset)
	
	// Create ffmpeg object
	options := &ffmpeg.Options{
		Quality: quality,
		Preset:  preset,
	}
	
	ff := ffmpeg.NewFFmpeg(inputFile, outputFile, options, logger)
	if err := ff.Execute(); err != nil {
		logger.Error("Falha na compressão: %v", err)
		return err
	}
	
	logger.Success("Compressão concluída com sucesso!")
	
	// Exibir estatísticas de compressão
	inputStat, err := os.Stat(inputFile)
	if err != nil {
		logger.Error("Erro ao obter informações do arquivo de entrada: %v", err)
		return nil // Não retorna erro para não interromper o fluxo
	}
	inputSize := inputStat.Size()

	outputStat, err := os.Stat(outputFile)
	if err != nil {
		logger.Error("Erro ao obter informações do arquivo de saída: %v", err)
		return nil // Não retorna erro para não interromper o fluxo
	}
	outputSize := outputStat.Size()

	if outputSize > 0 && inputSize > 0 {
		savings := 100.0 - (float64(outputSize) / float64(inputSize) * 100.0)
		logger.Info("Tamanho original: %s", util.FormatSize(inputSize))
		logger.Info("Tamanho final: %s", util.FormatSize(outputSize))
		logger.Info("Redução: %.1f%% (economia de %s)", savings, util.FormatSize(inputSize-outputSize))
	}

	return nil
} 