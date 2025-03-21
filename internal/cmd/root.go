// Package cmd implements command line interface for CompressVideo
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	recursive bool  // Process directories recursively

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
	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input video file or directory (required)")
	rootCmd.MarkFlagRequired("input")

	// Define optional flags
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file or directory (default: input_compressed.ext or input_compressed directory)")
	rootCmd.Flags().IntVarP(&quality, "quality", "q", 3, "Quality level (1-5, 1=max compression, 5=max quality)")
	rootCmd.Flags().StringVarP(&preset, "preset", "p", "balanced", "Compression preset (fast, balanced, thorough)")
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite output file or directory if it exists")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Process subdirectories recursively when input is a directory")
	
	// Add repair-ffmpeg command
	rootCmd.AddCommand(repairFFmpegCmd)
}

// validateFlags validates the input flags
func validateFlags() error {
	// Check if inputFile exists
	fileInfo, err := os.Stat(inputFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("input file or directory does not exist: %s", inputFile)
	}

	// Note: We now check for directory later, so we don't validate file extension here if it's a directory

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

	// If input is a file, validate output file
	if !fileInfo.IsDir() {
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
	} else if outputFile != "" {
		// If input is a directory and output is specified, output must be a directory too
		outputInfo, err := os.Stat(outputFile)
		if err == nil && !outputInfo.IsDir() {
			return fmt.Errorf("when input is a directory, output must also be a directory: %s", outputFile)
		}
		// Create output directory if it doesn't exist
		if os.IsNotExist(err) {
			if err := os.MkdirAll(outputFile, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %s", err)
			}
		}
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
	
	// Check if input is a directory
	fileInfo, _ := os.Stat(inputFile)
	if fileInfo.IsDir() {
		return processDirectory(inputFile, outputFile)
	}
	
	// Continue with single file processing
	// Resolve output file if not specified
	if outputFile == "" {
		dir := filepath.Dir(inputFile)
		ext := filepath.Ext(inputFile)
		base := filepath.Base(inputFile)
		base = strings.TrimSuffix(base, ext)
		outputFile = filepath.Join(dir, base+"_compressed"+ext)
	}
	
	// Process single file
	return processSingleFile(inputFile, outputFile)
}

// isVideoFile checks if a file extension matches common video formats
func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	videoExts := map[string]bool{
		".mp4":  true,
		".avi":  true,
		".mkv":  true,
		".mov":  true,
		".wmv":  true,
		".flv":  true,
		".webm": true,
		".m4v":  true,
		".3gp":  true,
		".ts":   true,
	}
	return videoExts[ext]
}

// hasCompressedSuffix checks if a filename has the "-compressed" suffix
func hasCompressedSuffix(filename string) bool {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	return strings.HasSuffix(nameWithoutExt, "-compressed")
}

// hasCompressedVersion checks if a compressed version of the file already exists
func hasCompressedVersion(filePath string) bool {
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filePath, ext)
	compressedPath := base + "-compressed" + ext
	
	// Check if the compressed version exists
	_, err := os.Stat(compressedPath)
	return err == nil
}

// processDirectory processes all video files in a directory
func processDirectory(inputDir, outputDir string) error {
	logger.Section("Processing Directory")
	logger.Field("Input Directory", "%s", inputDir)
	
	// If output directory is not specified, use the input directory
	if outputDir == "" {
		outputDir = inputDir
	}
	logger.Field("Output Directory", "%s", outputDir)
	
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}
	
	// Read all files in the directory
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}
	
	// Count number of video files to process
	var videoCount int
	var processedCount int
	var skippedCount int
	var subdirCount int
	
	// First pass: count video files
	for _, file := range files {
		if file.IsDir() {
			if recursive {
				subdirCount++
			}
			continue // Skip subdirectories in counting
		}
		
		filename := file.Name()
		filePath := filepath.Join(inputDir, filename)
		
		if isVideoFile(filename) && !hasCompressedSuffix(filename) && !hasCompressedVersion(filePath) {
			videoCount++
		}
	}
	
	if videoCount == 0 && subdirCount == 0 {
		logger.Warning("No video files found in the directory")
		return nil
	}
	
	logger.Info("Found %d video files to process", videoCount)
	
	// Process each video file
	fileIndex := 1 // Track the index of files being processed
	for _, file := range files {
		if file.IsDir() {
			// Process subdirectory if recursive flag is set
			if recursive {
				subInputDir := filepath.Join(inputDir, file.Name())
				subOutputDir := filepath.Join(outputDir, file.Name())
				
				logger.Section("Processing subdirectory: %s", file.Name())
				err := processDirectory(subInputDir, subOutputDir)
				if err != nil {
					logger.Error("Error processing subdirectory %s: %v", file.Name(), err)
				}
			}
			continue
		}
		
		filename := file.Name()
		
		// Skip non-video files
		if !isVideoFile(filename) {
			continue
		}
		
		// Skip already compressed files
		if hasCompressedSuffix(filename) {
			logger.Debug("Skipping already compressed file: %s", filename)
			skippedCount++
			continue
		}
		
		// Get input and output paths
		inputFilePath := filepath.Join(inputDir, filename)
		
		// Skip files that already have a compressed version
		if hasCompressedVersion(inputFilePath) {
			logger.Debug("Skipping file that already has a compressed version: %s", filename)
			skippedCount++
			continue
		}
		
		// Generate output filename
		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		outputFilename := base + "-compressed" + ext
		outputFilePath := filepath.Join(outputDir, outputFilename)
		
		// Check if output file already exists
		if _, err := os.Stat(outputFilePath); err == nil && !force {
			logger.Warning("Output file already exists (use -f to overwrite): %s", outputFilePath)
			skippedCount++
			continue
		}
		
		logger.Section("Processing File %d/%d", fileIndex, videoCount)
		logger.Field("Input File", "%s", inputFilePath)
		logger.Field("Output File", "%s", outputFilePath)
		
		// Process the file
		if err := processSingleFile(inputFilePath, outputFilePath); err != nil {
			logger.Error("Failed to process file %s: %v", inputFilePath, err)
			continue
		}
		
		processedCount++
		fileIndex++
	}
	
	logger.Section("Directory Processing Complete")
	logger.Info("Total files processed: %d/%d", processedCount, videoCount)
	if skippedCount > 0 {
		logger.Info("Files skipped: %d", skippedCount)
	}
	
	return nil
}

// processSingleFile processes a single video file
func processSingleFile(inputFile, outputFile string) error {
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
	
	// Check if input file exists (should already be validated, but double check)
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
	
	// Create content analyzer
	contentAnalyzer := analyzer.NewContentAnalyzer(ffmpegUtil, logger)
	
	// Analyze the video
	logger.Info("Analyzing video content...")
	analysis, err := contentAnalyzer.AnalyzeVideo(videoFile)
	if err != nil {
		logger.Error("Failed to analyze video: %v", err)
		return err
	}
	
	// Display analysis results
	displayAnalysisResults(analysis)
	
	// Get compression settings
	logger.Info("Determining optimal compression settings...")
	settings, err := contentAnalyzer.GetCompressionSettings(analysis, quality)
	if err != nil {
		logger.Error("Failed to determine compression settings: %v", err)
		return err
	}
	
	// Adjust settings based on preset
	if preset == "fast" {
		settings["preset"] = "veryfast"
	} else if preset == "thorough" {
		settings["preset"] = "slow"
	}
	
	// Create video compressor
	videoCompressor := compressor.NewVideoCompressor(ffmpegUtil, contentAnalyzer, logger)
	
	// Create progress tracker
	progressTracker := util.NewProgressTracker(100, "Comprimindo vídeo", logger)
	
	// Compress the video
	logger.Info("Starting video compression...")
	result, err := videoCompressor.CompressVideo(
		inputFile, 
		outputFile, 
		analysis, 
		settings, 
		quality, 
		preset, 
		progressTracker,
	)
	
	if err != nil {
		logger.Error("Compression failed: %v", err)
		return err
	}
	
	// Create report generator
	reportGen := reporter.NewReportGenerator(logger, ffmpegUtil)
	
	// Create initial report
	report := reportGen.CreateReport(inputFile, outputFile, videoFile, analysis)
	
	// Finalize the report with compression results
	report = reportGen.FinalizeReport(report, result)
	
	// Display the report
	reportGen.DisplayReportToConsole(report)
	
	// Save report to file if verbose
	if verbose {
		reportPath, err := reportGen.SaveReportToFile(report)
		if err != nil {
			logger.Warning("Failed to save report: %v", err)
		} else {
			logger.Info("Detailed report saved to: %s", reportPath)
		}
	}
	
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
	
	// Validate input
	fileInfo, err := os.Stat(inputFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("o arquivo ou diretório de entrada não existe: %s", inputFile)
		}
		return fmt.Errorf("erro ao acessar arquivo ou diretório de entrada: %v", err)
	}
	
	logger.Info("Arquivo de entrada: %s", inputFile)
	
	// Check if input is a directory
	if fileInfo.IsDir() {
		// Process directory
		logger.Info("Processando diretório")
		return processDirectoryCompression(inputFile, outputFile)
	}
	
	// Single file processing
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
	
	// Process the single file
	return executeSingleFileCompression(inputFile, outputFile)
}

// processDirectoryCompression processes all video files in a directory
func processDirectoryCompression(inputDir, outputDir string) error {
	// If output directory is not specified, use input directory
	if outputDir == "" {
		outputDir = inputDir
	}
	
	logger.Info("Diretório de saída: %s", outputDir)
	
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar diretório de saída: %v", err)
	}
	
	// Read all files in the directory
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("falha ao ler diretório: %v", err)
	}
	
	// Count number of video files to process
	var videoCount int
	var processedCount int
	var skippedCount int
	var subdirCount int
	
	// First pass: count video files
	for _, file := range files {
		if file.IsDir() {
			if recursive {
				subdirCount++
			}
			continue // Skip subdirectories in counting
		}
		
		filename := file.Name()
		filePath := filepath.Join(inputDir, filename)
		
		if isVideoFile(filename) && !hasCompressedSuffix(filename) && !hasCompressedVersion(filePath) {
			videoCount++
		}
	}
	
	if videoCount == 0 && subdirCount == 0 {
		logger.Warning("Nenhum arquivo de vídeo encontrado no diretório")
		return nil
	}
	
	logger.Info("Encontrados %d arquivos de vídeo para processar", videoCount)
	
	// Process each video file
	fileIndex := 1 // Track the index of files being processed
	for _, file := range files {
		if file.IsDir() {
			// Process subdirectory if recursive flag is set
			if recursive {
				subInputDir := filepath.Join(inputDir, file.Name())
				subOutputDir := filepath.Join(outputDir, file.Name())
				
				logger.Section("Processando subdiretório: %s", file.Name())
				err := processDirectoryCompression(subInputDir, subOutputDir)
				if err != nil {
					logger.Error("Erro ao processar subdiretório %s: %v", file.Name(), err)
				}
			}
			continue
		}
		
		filename := file.Name()
		
		// Skip non-video files
		if !isVideoFile(filename) {
			continue
		}
		
		// Skip already compressed files
		if hasCompressedSuffix(filename) {
			logger.Debug("Ignorando arquivo já comprimido: %s", filename)
			skippedCount++
			continue
		}
		
		// Get input and output paths
		inputFilePath := filepath.Join(inputDir, filename)
		
		// Skip files that already have a compressed version
		if hasCompressedVersion(inputFilePath) {
			logger.Debug("Ignorando arquivo que já possui versão comprimida: %s", filename)
			skippedCount++
			continue
		}
		
		// Generate output filename
		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		outputFilename := base + "-compressed" + ext
		outputFilePath := filepath.Join(outputDir, outputFilename)
		
		// Check if output file already exists
		if _, err := os.Stat(outputFilePath); err == nil && !force {
			logger.Warning("Arquivo de saída já existe (use -f para sobrescrever): %s", outputFilePath)
			skippedCount++
			continue
		}
		
		logger.Section("Processando arquivo %d/%d", fileIndex, videoCount)
		logger.Info("Arquivo de entrada: %s", inputFilePath)
		logger.Info("Arquivo de saída: %s", outputFilePath)
		
		// Process the file
		if err := executeSingleFileCompression(inputFilePath, outputFilePath); err != nil {
			logger.Error("Falha ao processar arquivo %s: %v", inputFilePath, err)
			continue
		}
		
		processedCount++
		fileIndex++
	}
	
	logger.Section("Processamento de diretório concluído")
	logger.Info("Total de arquivos processados: %d/%d", processedCount, videoCount)
	if skippedCount > 0 {
		logger.Info("Arquivos ignorados: %d", skippedCount)
	}
	
	return nil
}

// executeSingleFileCompression processes a single video file
func executeSingleFileCompression(inputFile, outputFile string) error {
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