package compressor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cccarv82/compressvideo/pkg/analyzer"
	"github.com/cccarv82/compressvideo/pkg/ffmpeg"
	"github.com/cccarv82/compressvideo/pkg/util"
)

// CompressionResult contains the results of a video compression
type CompressionResult struct {
	InputFile           string
	OutputFile          string
	OriginalSize        int64
	CompressedSize      int64
	CompressionRatio    float64
	SavedSpaceBytes     int64
	SavedSpacePercent   float64
	ProcessingTime      time.Duration
	AverageFrameQuality float64
	FFmpegCommand       string
	Settings            map[string]string
	Error               error
}

// VideoCompressor handles video compression operations
type VideoCompressor struct {
	FFmpeg           *ffmpeg.FFmpeg
	Logger           *util.Logger
	Analyzer         *analyzer.ContentAnalyzer
	ConcurrentWorkers int
	TempDir          string
}

// NewVideoCompressor creates a new video compressor
func NewVideoCompressor(ffmpeg *ffmpeg.FFmpeg, analyzer *analyzer.ContentAnalyzer, logger *util.Logger) *VideoCompressor {
	// Set the number of concurrent workers to CPU cores
	concurrentWorkers := runtime.NumCPU()
	
	// Create temp directory for processing
	tempDir := filepath.Join(os.TempDir(), "compressvideo")
	os.MkdirAll(tempDir, 0755)
	
	return &VideoCompressor{
		FFmpeg:           ffmpeg,
		Logger:           logger,
		Analyzer:         analyzer,
		ConcurrentWorkers: concurrentWorkers,
		TempDir:          tempDir,
	}
}

// CompressVideo compresses a video with the given settings
func (vc *VideoCompressor) CompressVideo(inputFile, outputFile string, analysis *analyzer.VideoAnalysis, 
	settings map[string]string, quality int, preset string, progress *util.ProgressTracker) (*CompressionResult, error) {
	
	startTime := time.Now()
	
	// Get original file size
	inputInfo, err := os.Stat(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get input file info: %w", err)
	}
	originalSize := inputInfo.Size()
	
	// Calculate optimal compression settings if not provided
	if settings == nil {
		settings, err = vc.Analyzer.GetCompressionSettings(analysis, quality)
		if err != nil {
			return nil, fmt.Errorf("failed to get compression settings: %w", err)
		}
	}
	
	// Adjust settings based on preset
	vc.adjustSettingsForPreset(settings, preset)
	
	// Prepare result
	result := &CompressionResult{
		InputFile:    inputFile,
		OutputFile:   outputFile,
		OriginalSize: originalSize,
		Settings:     settings,
	}
	
	// Determine compression approach based on content type and video length
	useParallelCompression := analysis.VideoFile.Duration > 60 && 
		analysis.ContentType != analyzer.ContentTypeScreencast
	
	// Execute compression
	if useParallelCompression {
		err = vc.compressVideoParallel(inputFile, outputFile, settings, progress)
	} else {
		err = vc.compressVideoSingle(inputFile, outputFile, settings, progress)
	}
	
	if err != nil {
		result.Error = err
		return result, err
	}
	
	// Get compressed file size
	outputInfo, err := os.Stat(outputFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to get output file info: %w", err)
		return result, result.Error
	}
	result.CompressedSize = outputInfo.Size()
	
	// Calculate compression metrics
	result.ProcessingTime = time.Since(startTime)
	result.SavedSpaceBytes = originalSize - result.CompressedSize
	result.CompressionRatio = float64(originalSize) / float64(result.CompressedSize)
	result.SavedSpacePercent = float64(result.SavedSpaceBytes) / float64(originalSize) * 100
	
	// Calculate average frame quality (can be done through VMAF or SSIM if needed)
	// For now, we'll use a placeholder that estimates based on settings
	result.AverageFrameQuality = vc.EstimateFrameQuality(settings)
	
	return result, nil
}

// adjustSettingsForPreset adjusts the compression settings based on the chosen preset
func (vc *VideoCompressor) adjustSettingsForPreset(settings map[string]string, preset string) {
	// Get current preset speed from settings
	currentPreset := settings["preset"]
	
	switch preset {
	case "fast":
		// For fast preset, use a faster encoding setting
		if currentPreset == "veryslow" {
			settings["preset"] = "medium"
		} else if currentPreset == "slower" || currentPreset == "slow" {
			settings["preset"] = "fast"
		} else if currentPreset == "medium" {
			settings["preset"] = "veryfast"
		} else {
			settings["preset"] = "ultrafast"
		}
		
		// Decrease thread count for speed
		settings["threads"] = strconv.Itoa(vc.ConcurrentWorkers)
		
	case "thorough":
		// For thorough preset, use slower, more efficient encoding
		if currentPreset == "ultrafast" || currentPreset == "veryfast" {
			settings["preset"] = "medium"
		} else if currentPreset == "fast" || currentPreset == "medium" {
			settings["preset"] = "slow"
		} else {
			settings["preset"] = "veryslow"
		}
		
		// Improve quality slightly
		if crf, ok := settings["crf"]; ok {
			crfValue, _ := strconv.Atoi(crf)
			if crfValue > 20 { // Lower CRF means higher quality
				settings["crf"] = strconv.Itoa(crfValue - 2)
			}
		}
	}
	
	// "balanced" preset uses the default settings from the analyzer
}

// compressVideoSingle compresses a single video file
func (vc *VideoCompressor) compressVideoSingle(inputFile, outputFile string, settings map[string]string, progress *util.ProgressTracker) error {
	// Obter o caminho para o FFmpeg
	ffmpegInfo, err := util.FindFFmpeg()
	if err != nil {
		return fmt.Errorf("erro ao encontrar FFmpeg: %v", err)
	}
	ffmpegPath := ffmpegInfo.Path
	
	// Build FFmpeg command arguments
	args := vc.BuildFFmpegArgs(inputFile, outputFile, settings)
	
	// Log the command
	cmdStr := fmt.Sprintf("%s %s", ffmpegPath, strings.Join(args, " "))
	vc.Logger.Debug("Running FFmpeg command: %s", cmdStr)
	
	// Create command
	cmd := exec.Command(ffmpegPath, args...)
	
	// Get video duration for progress calculation
	videoFile, err := vc.FFmpeg.GetVideoInfo(inputFile)
	if err != nil {
		return fmt.Errorf("error getting video duration: %w", err)
	}
	totalDuration := videoFile.Duration
	
	// Variável para capturar a saída completa de stderr para análise de erros
	var stderrOutput strings.Builder
	
	// Pipe stderr to capture progress info
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %w", err)
	}
	
	// Monitor progress
	go func() {
		// Read FFmpeg output and update progress
		buf := make([]byte, 2048)
		var lastProgressReported int64
		
		for {
			n, err := stderr.Read(buf)
			if n == 0 || err != nil {
				break
			}
			
			output := string(buf[:n])
			stderrOutput.WriteString(output) // Capturar a saída completa
			
			// Parse time information from FFmpeg output
			timeMatch := strings.Index(output, "time=")
			if timeMatch != -1 {
				// Encontrar o fim da string de tempo (até o espaço)
				endIdx := timeMatch + 5
				for endIdx < len(output) && output[endIdx] != ' ' {
					endIdx++
				}
				
				if endIdx > timeMatch+5 {
					timeStr := output[timeMatch+5:endIdx]
					timeStr = strings.TrimSpace(timeStr)
					
					// Parse time in HH:MM:SS format
					if len(timeStr) >= 8 { // Garantir que tem pelo menos "HH:MM:SS"
						parts := strings.Split(timeStr, ":")
						if len(parts) >= 3 {
							hours, _ := strconv.Atoi(parts[0])
							minutes, _ := strconv.Atoi(parts[1])
							
							// Remover sufixos que possam existir na parte dos segundos
							secondsStr := parts[2]
							if dotIndex := strings.Index(secondsStr, "."); dotIndex >= 0 {
								secondsStr = secondsStr[:dotIndex+3] // Manter até 2 casas decimais
							}
							
							seconds, _ := strconv.ParseFloat(secondsStr, 64)
							currentTime := float64(hours*3600) + float64(minutes*60) + seconds
							
							// Update progress
							if totalDuration > 0 {
								percentComplete := int64((currentTime / totalDuration) * 100)
								if percentComplete > 100 {
									percentComplete = 100
								}
								
								// Update progress only if it's different from last reported
								if percentComplete != lastProgressReported {
									progress.Update(percentComplete)
									lastProgressReported = percentComplete
								}
							}
						}
					}
				}
			}
		}
	}()
	
	// Wait for command to finish
	err = cmd.Wait()
	if err != nil {
		errorOutput := stderrOutput.String()
		return fmt.Errorf("FFmpeg error: %w\nDetails: %s", err, errorOutput)
	}
	
	// Set progress to 100%
	progress.Update(100)
	
	return nil
}

// compressVideoParallel compresses a video by splitting it into segments and processing in parallel
func (vc *VideoCompressor) compressVideoParallel(inputFile, outputFile string, settings map[string]string, progress *util.ProgressTracker) error {
	vc.Logger.Info("Using parallel compression for faster processing")
	
	// Get video duration to split into segments
	videoFile, err := vc.FFmpeg.GetVideoInfo(inputFile)
	if err != nil {
		return fmt.Errorf("error getting video info: %w", err)
	}
	
	// Create temporary directory for segments
	segmentDir := filepath.Join(vc.TempDir, fmt.Sprintf("segments_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(segmentDir, 0755); err != nil {
		return fmt.Errorf("failed to create segment directory: %w", err)
	}
	defer os.RemoveAll(segmentDir) // Clean up when done
	
	// Calculate how many segments to create (1 per CPU core)
	numSegments := vc.ConcurrentWorkers
	if numSegments > 8 {
		numSegments = 8 // Cap at 8 segments to avoid overhead
	}
	
	// Split the video into segments
	segmentDuration := videoFile.Duration / float64(numSegments)
	segments, err := vc.splitVideo(inputFile, segmentDir, segmentDuration, numSegments)
	if err != nil {
		return fmt.Errorf("failed to split video: %w", err)
	}
	
	// Create list file for concat
	listPath := filepath.Join(segmentDir, "segments.txt")
	listFile, err := os.Create(listPath)
	if err != nil {
		return fmt.Errorf("failed to create segment list: %w", err)
	}
	
	// Compress segments in parallel
	var wg sync.WaitGroup
	compressedSegments := make([]string, len(segments))
	errorChan := make(chan error, len(segments))
	progressChan := make(chan int, 100) // For progress updates
	
	// Start a goroutine to aggregate progress updates
	go func() {
		segmentProgress := make([]int, len(segments))
		for progressUpdate := range progressChan {
			segmentID := progressUpdate >> 16    // Primeiros 16 bits contêm o ID do segmento
			progressValue := progressUpdate & 0xFFFF  // Últimos 16 bits contêm o valor do progresso
			
			segmentProgress[segmentID] = progressValue
			
			// Calcular o progresso médio de todos os segmentos
			totalProgress := 0
			for _, p := range segmentProgress {
				totalProgress += p
			}
			
			// 90% para compressão, 10% reservado para a fusão final
			avgProgress := int64(float64(totalProgress) / float64(len(segments) * 100) * 90)
			if avgProgress < 90 {
				progress.Update(avgProgress)
			}
		}
	}()
	
	// Start workers for each segment
	for i, segment := range segments {
		wg.Add(1)
		go func(i int, segment string) {
			defer wg.Done()
			
			// Create output path for compressed segment
			outSegment := filepath.Join(segmentDir, fmt.Sprintf("out_%04d.mp4", i))
			compressedSegments[i] = outSegment
			
			// Clone settings map to avoid race conditions
			segmentSettings := make(map[string]string)
			for k, v := range settings {
				segmentSettings[k] = v
			}
			
			// Force key frames at segment boundaries
			segmentSettings["force_key_frames"] = "expr:eq(n,0)"
			
			// Create segment progress tracker that reports to the channel
			segmentProgress := &segmentProgressTracker{
				segmentID: i,
				progressChan: progressChan,
			}
			
			// Compress this segment
			err := vc.compressSegment(segment, outSegment, segmentSettings, segmentProgress)
			if err != nil {
				errorChan <- fmt.Errorf("segment %d error: %w", i, err)
				return
			}
			
			// Write entry to list file (thread-safe by using a synchronized file)
			path, _ := filepath.Rel(segmentDir, outSegment)
			line := fmt.Sprintf("file '%s'\n", path)
			if _, err := listFile.WriteString(line); err != nil {
				errorChan <- fmt.Errorf("failed to write to list file: %w", err)
			}
		}(i, segment)
	}
	
	// Wait for all segments to be compressed
	wg.Wait()
	close(progressChan)
	listFile.Close()
	
	// Check for errors
	select {
	case err := <-errorChan:
		return err
	default:
		// No errors
	}
	
	// Update progress
	progress.Update(90)
	
	// Merge the segments
	vc.Logger.Info("Merging compressed segments...")
	err = vc.mergeSegments(listPath, outputFile, settings["codec"])
	if err != nil {
		return fmt.Errorf("failed to merge segments: %w", err)
	}
	
	// Set progress to 100%
	progress.Update(100)
	
	return nil
}

// splitVideo splits a video into multiple segments of equal duration
func (vc *VideoCompressor) splitVideo(inputFile, segmentDir string, segmentDuration float64, numSegments int) ([]string, error) {
	segments := make([]string, numSegments)
	
	// Obter o caminho para o FFmpeg
	ffmpegInfo, err := util.FindFFmpeg()
	if err != nil {
		return nil, fmt.Errorf("erro ao encontrar FFmpeg: %v", err)
	}
	ffmpegPath := ffmpegInfo.Path
	
	for i := 0; i < numSegments; i++ {
		startTime := float64(i) * segmentDuration
		outPath := filepath.Join(segmentDir, fmt.Sprintf("segment_%04d.mp4", i))
		segments[i] = outPath
		
		args := []string{
			"-ss", fmt.Sprintf("%.3f", startTime),
			"-i", inputFile,
			"-t", fmt.Sprintf("%.3f", segmentDuration),
			"-c", "copy", // Use copy to make splitting fast
			"-avoid_negative_ts", "1",
			"-y", outPath,
		}
		
		cmd := exec.Command(ffmpegPath, args...)
		vc.Logger.Debug("Splitting segment %d: %s", i, strings.Join(cmd.Args, " "))
		
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to split segment %d: %w\nOutput: %s", i, err, string(output))
		}
	}
	
	return segments, nil
}

// compressSegment compresses a single video segment
func (vc *VideoCompressor) compressSegment(inputFile, outputFile string, settings map[string]string, progress progressReporter) error {
	// Obter o caminho para o FFmpeg
	ffmpegInfo, err := util.FindFFmpeg()
	if err != nil {
		return fmt.Errorf("erro ao encontrar FFmpeg: %v", err)
	}
	ffmpegPath := ffmpegInfo.Path
	
	// Build FFmpeg command
	args := vc.BuildFFmpegArgs(inputFile, outputFile, settings)
	
	// Run FFmpeg
	cmd := exec.Command(ffmpegPath, args...)
	
	// Get video duration for progress calculation
	videoFile, err := vc.FFmpeg.GetVideoInfo(inputFile)
	if err != nil {
		return fmt.Errorf("error getting video duration: %w", err)
	}
	totalDuration := videoFile.Duration
	
	// Pipe stderr to capture progress info
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %w", err)
	}
	
	// Monitor progress
	go func() {
		// Read FFmpeg output and update progress
		buf := make([]byte, 2048)
		var lastProgressReported int64
		
		for {
			n, err := stderr.Read(buf)
			if n == 0 || err != nil {
				break
			}
			
			output := string(buf[:n])
			
			// Parse time information from FFmpeg output
			timeMatch := strings.Index(output, "time=")
			if timeMatch != -1 {
				// Encontrar o fim da string de tempo (até o espaço)
				endIdx := timeMatch + 5
				for endIdx < len(output) && output[endIdx] != ' ' {
					endIdx++
				}
				
				if endIdx > timeMatch+5 {
					timeStr := output[timeMatch+5:endIdx]
					timeStr = strings.TrimSpace(timeStr)
					
					// Parse time in HH:MM:SS format
					if len(timeStr) >= 8 { // Garantir que tem pelo menos "HH:MM:SS"
						parts := strings.Split(timeStr, ":")
						if len(parts) >= 3 {
							hours, _ := strconv.Atoi(parts[0])
							minutes, _ := strconv.Atoi(parts[1])
							
							// Remover sufixos que possam existir na parte dos segundos
							secondsStr := parts[2]
							if dotIndex := strings.Index(secondsStr, "."); dotIndex >= 0 {
								secondsStr = secondsStr[:dotIndex+3] // Manter até 2 casas decimais
							}
							
							seconds, _ := strconv.ParseFloat(secondsStr, 64)
							currentTime := float64(hours*3600) + float64(minutes*60) + seconds
							
							// Update progress
							if totalDuration > 0 {
								percentComplete := int64((currentTime / totalDuration) * 100)
								if percentComplete > 100 {
									percentComplete = 100
								}
								
								// Só atualizar se houver mudança significativa ou for o final
								if percentComplete > lastProgressReported || percentComplete >= 100 {
									progress.reportProgress(int(percentComplete))
									lastProgressReported = percentComplete
								}
							}
						}
					}
				}
			}
			
			// Log errors in verbose mode
			if strings.Contains(strings.ToLower(output), "error") {
				vc.Logger.Debug("FFmpeg output: %s", output)
			}
		}
	}()
	
	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("FFmpeg error: %w", err)
	}
	
	// Set progress to 100%
	progress.reportProgress(100)
	
	return nil
}

// mergeSegments merges multiple video segments into one output file
func (vc *VideoCompressor) mergeSegments(listFile, outputFile, codec string) error {
	// Obter o caminho para o FFmpeg
	ffmpegInfo, err := util.FindFFmpeg()
	if err != nil {
		return fmt.Errorf("erro ao encontrar FFmpeg: %v", err)
	}
	ffmpegPath := ffmpegInfo.Path
	
	// Use FFmpeg's concat demuxer to merge segments
	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy", // Just copy the streams without re-encoding
		"-y", outputFile,
	}
	
	cmd := exec.Command(ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to merge segments: %w\nOutput: %s", err, string(output))
	}
	
	return nil
}

// BuildFFmpegArgs constrói os argumentos para o comando FFmpeg
func (vc *VideoCompressor) BuildFFmpegArgs(inputFile, outputFile string, settings map[string]string) []string {
	// Base arguments
	args := []string{"-y"}
	
	// Add input file
	args = append(args, "-i", inputFile)
	
	// Add codec settings
	codec := settings["codec"]
	if codec != "" {
		args = append(args, "-c:v", codec)
	}
	
	// Add preset
	preset := settings["preset"]
	if preset != "" {
		args = append(args, "-preset", preset)
	}
	
	// Add CRF value for quality
	crf := settings["crf"]
	if crf != "" {
		args = append(args, "-crf", crf)
	}
	
	// Add profile
	profile := settings["profile"]
	if profile != "" {
		args = append(args, "-profile:v", profile)
	}
	
	// Add level
	level := settings["level"]
	if level != "" {
		args = append(args, "-level", level)
	}
	
	// Add tuning parameter
	tune := settings["tune"]
	if tune != "" {
		args = append(args, "-tune", tune)
	}
	
	// Add codec-specific parameters
	if codec == "libx265" && settings["x265-params"] != "" {
		args = append(args, "-x265-params", settings["x265-params"])
	} else if strings.Contains(codec, "nvenc") {
		// Adicionar parâmetros específicos para NVENC para melhorar a compatibilidade
		if codec == "h264_nvenc" || codec == "hevc_nvenc" {
			args = append(args, "-rc", "vbr")
			
			// Garantir que temos um valor de bitrate para usar
			if bitrate, ok := settings["bitrate"]; ok && bitrate != "" {
				args = append(args, "-b:v", bitrate)
			} else {
				// Definir um bitrate padrão baseado na resolução
				defaultBitrate := "4M" // Valor padrão para maioria dos vídeos
				if video, _ := vc.FFmpeg.GetVideoInfo(inputFile); video != nil {
					if video.VideoInfo.Height <= 720 {
						defaultBitrate = "2M" // 2 Mbps para 720p
					} else if video.VideoInfo.Height <= 1080 {
						defaultBitrate = "4M" // 4 Mbps para 1080p
					} else {
						defaultBitrate = "8M" // 8 Mbps para 4K
					}
				}
				args = append(args, "-b:v", defaultBitrate)
				vc.Logger.Debug("Usando bitrate padrão para NVENC: %s", defaultBitrate)
			}
			
			// Usar diferentes parâmetros dependendo da plataforma
			if runtime.GOOS == "windows" {
				// Parâmetros mais simples para Windows para evitar bugs
				vc.Logger.Debug("Usando configuração NVENC simplificada para Windows")
			} else {
				// Configuração completa para outras plataformas
				args = append(args, "-rc-lookahead", "20")
				args = append(args, "-spatial-aq", "1")
				args = append(args, "-temporal-aq", "1")
			}
		}
	}
	
	// Add pixel format if specified
	pixFmt := settings["pix_fmt"]
	if pixFmt != "" {
		args = append(args, "-pix_fmt", pixFmt)
	}
	
	// Add force key frames if specified
	forceKeyFrames := settings["force_key_frames"]
	if forceKeyFrames != "" {
		args = append(args, "-force_key_frames", forceKeyFrames)
	}
	
	// Add audio codec settings
	audioCodec := settings["audio_codec"]
	if audioCodec != "" {
		if audioCodec == "copy" {
			args = append(args, "-c:a", "copy")
		} else {
			args = append(args, "-c:a", audioCodec)
			
			// Add audio bitrate if specified
			audioBitrate := settings["audio_bitrate"]
			if audioBitrate != "" {
				args = append(args, "-b:a", audioBitrate)
			}
		}
	}
	
	// Add thread count
	threads := settings["threads"]
	if threads != "" {
		args = append(args, "-threads", threads)
	}
	
	// Add target bitrate if specified
	bitrate := settings["bitrate"]
	if bitrate != "" {
		args = append(args, "-b:v", bitrate)
	}
	
	// Add output file
	args = append(args, outputFile)
	
	return args
}

// EstimateFrameQuality estima a qualidade do quadro com base nas configurações de compressão
func (vc *VideoCompressor) EstimateFrameQuality(settings map[string]string) float64 {
	codec := settings["codec"]
	crf := 23.0 // Default CRF
	
	// Parse CRF if available
	if crfStr, ok := settings["crf"]; ok {
		parsedCRF, err := strconv.ParseFloat(crfStr, 64)
		if err == nil {
			crf = parsedCRF
		}
	}
	
	// Estimate quality based on codec and CRF
	quality := 0.0
	
	if codec == "libx264" {
		// For H.264, CRF 0-51 (0=lossless, 23=default, 51=worst)
		// Convert to 0-100 scale
		quality = 100 - (crf * 1.96) // 100-(23*1.96) ≈ 55 (default quality)
	} else if codec == "libx265" {
		// For H.265, CRF 0-51 (0=lossless, 28=default, 51=worst)
		// HEVC generally has better quality at the same CRF compared to H.264
		quality = 100 - (crf * 1.76) // 100-(28*1.76) ≈ 51 (default quality)
	} else if codec == "libvpx-vp9" {
		// For VP9, CRF 0-63 (0=lossless, 31=default, 63=worst)
		// Convert to 0-100 scale
		quality = 100 - (crf * 1.58) // 100-(31*1.58) ≈ 51 (default quality)
	}
	
	// Ensure quality is in 0-100 range
	if quality < 0 {
		quality = 0
	} else if quality > 100 {
		quality = 100
	}
	
	return quality
}

// Simple progress reporter interface for segment compression
type progressReporter interface {
	reportProgress(progress int)
}

// Implementation of progress reporter for segments
type segmentProgressTracker struct {
	segmentID    int
	progressChan chan<- int
}

func (spt *segmentProgressTracker) reportProgress(progress int) {
	// Combinar segmentID e progress em um único inteiro
	// Os primeiros 16 bits representam o segmentID, os últimos 16 bits representam o progresso
	combinedProgress := (spt.segmentID << 16) | progress
	spt.progressChan <- combinedProgress
}

func (vc *VideoCompressor) compressVideoWithTwoPass(inputFile, outputFile string, settings map[string]string, progress *util.ProgressTracker) error {
	vc.Logger.Debug("Starting two-pass compression")
	
	// Obter o caminho para o FFmpeg
	ffmpegInfo, err := util.FindFFmpeg()
	if err != nil {
		return fmt.Errorf("erro ao encontrar FFmpeg: %v", err)
	}
	ffmpegPath := ffmpegInfo.Path
	
	// Build FFmpeg arguments for both passes
	firstPassArgs := append(vc.BuildFFmpegArgs(inputFile, "NUL", settings), "-pass", "1", "-f", "null", "-")
	
	// Modify for macOS/Linux
	if runtime.GOOS != "windows" {
		firstPassArgs = append(vc.BuildFFmpegArgs(inputFile, "/dev/null", settings), "-pass", "1", "-f", "null", "/dev/null")
	}
	
	// Run first pass
	vc.Logger.Debug("Running first pass compression")
	cmd := exec.Command(ffmpegPath, firstPassArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg first pass error: %w\nOutput: %s", err, string(output))
	}
	
	// Run second pass
	secondPassArgs := append(vc.BuildFFmpegArgs(inputFile, outputFile, settings), "-pass", "2")
	vc.Logger.Debug("Running second pass compression")
	cmd = exec.Command(ffmpegPath, secondPassArgs...)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg second pass error: %w\nOutput: %s", err, string(output))
	}
	
	return nil
}

func (vc *VideoCompressor) compressVideoSegment(inputFile, outputFile string, startTime, duration float64, settings map[string]string) error {
	vc.Logger.Debug("Compressing segment: %s from %.2fs for %.2fs", inputFile, startTime, duration)
	
	// Obter o caminho para o FFmpeg
	ffmpegInfo, err := util.FindFFmpeg()
	if err != nil {
		return fmt.Errorf("erro ao encontrar FFmpeg: %v", err)
	}
	ffmpegPath := ffmpegInfo.Path
	
	// Add segment-specific arguments
	segmentArgs := []string{
		"-ss", fmt.Sprintf("%.3f", startTime),
		"-t", fmt.Sprintf("%.3f", duration),
	}
	
	// Build the FFmpeg command
	args := append(segmentArgs, vc.BuildFFmpegArgs(inputFile, outputFile, settings)...)
	
	// Execute command
	vc.Logger.Debug("Running FFmpeg segment command: %s %s", ffmpegPath, strings.Join(args, " "))
	cmd := exec.Command(ffmpegPath, args...)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to compress segment: %w\nOutput: %s", err, string(output))
	}
	
	return nil
} 