package ffmpeg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cccarv82/compressvideo/pkg/util"
)

// VideoInfo contains information about a video file
type VideoInfo struct {
	Width      int     // Width in pixels
	Height     int     // Height in pixels
	FrameRate  float64 // Frame rate in frames per second
	Duration   float64 // Duration in seconds
	BitRate    int64   // Bit rate in bits per second
	CodecName  string  // Codec name
	SizeBits   int64   // Size in bits
	Resolution string  // Resolution as a string (e.g. "1920x1080")
}

// EncodingSettings contains settings for encoding
type EncodingSettings struct {
	VideoCodec    string // Video codec to use (e.g. libx264)
	CRF           int    // Constant Rate Factor (quality)
	Preset        string // Compression preset (e.g. medium, slow)
	MaxWidth      int    // Maximum width for scaling
	MaxHeight     int    // Maximum height for scaling
	TargetBitrate int64  // Target bitrate in bits per second
}

// Options contains options for FFmpeg compression
type Options struct {
	Quality int    // Quality level (1-5, 1=max compression, 5=max quality)
	Preset  string // Preset (fast, balanced, thorough)
}

// FFmpeg represents an FFmpeg instance
type FFmpeg struct {
	InputFile   string    // Input file path
	OutputFile  string    // Output file path
	Options     *Options  // Compression options
	Logger      *util.Logger // Logger
}

// NewFFmpeg cria uma nova instância do FFmpeg
func NewFFmpeg(inputFile, outputFile string, options *Options, logger *util.Logger) *FFmpeg {
	if options == nil {
		options = DefaultOptions()
	}
	
	return &FFmpeg{
		InputFile:  inputFile,
		OutputFile: outputFile,
		Options:    options,
		Logger:     logger,
	}
}

// GetVideoInfo retrieves detailed information about a video file using ffprobe
func (f *FFmpeg) GetVideoInfo(filePath string) (*VideoFile, error) {
	f.Logger.Debug("Getting video info for: %s", filePath)

	// Obter o caminho para o FFprobe
	info, err := util.FindFFmpeg()
	if err != nil {
		return nil, fmt.Errorf("erro ao encontrar FFmpeg: %v", err)
	}
	
	ffprobePath := info.FFprobePath

	// Run ffprobe to get JSON output with all stream info
	cmd := exec.Command(
		ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	// Parse the JSON output
	var ffprobeOutput struct {
		Streams []map[string]interface{} `json:"streams"`
		Format  map[string]interface{}   `json:"format"`
	}

	err = json.Unmarshal(output, &ffprobeOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Create a new VideoFile
	videoFile := &VideoFile{
		Path:     filePath,
		Format:   filepath.Ext(filePath)[1:], // Remove the dot
		Metadata: make(map[string]string),
		AudioInfo: []AudioStreamInfo{},
	}

	// Extract duration and size from format section
	if durationStr, ok := ffprobeOutput.Format["duration"].(string); ok {
		videoFile.Duration, _ = strconv.ParseFloat(durationStr, 64)
	}
	if sizeStr, ok := ffprobeOutput.Format["size"].(string); ok {
		size, _ := strconv.ParseInt(sizeStr, 10, 64)
		videoFile.Size = size
	}
	if bitrateStr, ok := ffprobeOutput.Format["bit_rate"].(string); ok {
		bitrate, _ := strconv.ParseInt(bitrateStr, 10, 64)
		videoFile.BitRate = bitrate
	}

	// Process each stream
	for _, stream := range ffprobeOutput.Streams {
		streamType, ok := stream["codec_type"].(string)
		if !ok {
			continue
		}

		if streamType == "video" {
			// Extract video stream info
			videoInfo := VideoStreamInfo{}
			
			// Extract codec
			if codec, ok := stream["codec_name"].(string); ok {
				videoInfo.Codec = codec
			}
			
			// Extract dimensions
			if width, ok := stream["width"].(float64); ok {
				videoInfo.Width = int(width)
			}
			if height, ok := stream["height"].(float64); ok {
				videoInfo.Height = int(height)
			}
			
			// Extract framerate
			if fpsStr, ok := stream["r_frame_rate"].(string); ok {
				parts := strings.Split(fpsStr, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den != 0 {
						videoInfo.FPS = num / den
					}
				}
			}
			
			// Extract pixel format
			if pixFmt, ok := stream["pix_fmt"].(string); ok {
				videoInfo.PixelFormat = pixFmt
			}
			
			// Extract profile level
			if profile, ok := stream["profile"].(string); ok {
				videoInfo.ProfileLevel = profile
			}
			
			// Extract video bitrate
			if bitrateStr, ok := stream["bit_rate"].(string); ok {
				bitrate, _ := strconv.ParseInt(bitrateStr, 10, 64)
				videoInfo.BitRate = bitrate
			}
			
			// Check for B-frames using profile
			if profile, ok := stream["profile"].(string); ok {
				// Most profiles with B-frames have "High" in the name
				if strings.Contains(strings.ToLower(profile), "high") {
					videoInfo.HasBFrames = true
				}
			}
			
			// Check for HDR
			if tags, ok := stream["tags"].(map[string]interface{}); ok {
				if colorTransfer, ok := tags["color_transfer"].(string); ok {
					if strings.Contains(strings.ToLower(colorTransfer), "smpte2084") || 
					   strings.Contains(strings.ToLower(colorTransfer), "arib-std-b67") {
						videoInfo.IsHDR = true
					}
				}
			}
			
			videoFile.VideoInfo = videoInfo
			
		} else if streamType == "audio" {
			// Extract audio stream info
			audioInfo := AudioStreamInfo{}
			
			// Extract index
			if index, ok := stream["index"].(float64); ok {
				audioInfo.Index = int(index)
			}
			
			// Extract codec
			if codec, ok := stream["codec_name"].(string); ok {
				audioInfo.Codec = codec
			}
			
			// Extract channels
			if channels, ok := stream["channels"].(float64); ok {
				audioInfo.Channels = int(channels)
			}
			
			// Extract sample rate
			if sampleRateStr, ok := stream["sample_rate"].(string); ok {
				sampleRate, _ := strconv.Atoi(sampleRateStr)
				audioInfo.SampleRate = sampleRate
			}
			
			// Extract bitrate
			if bitrateStr, ok := stream["bit_rate"].(string); ok {
				bitrate, _ := strconv.ParseInt(bitrateStr, 10, 64)
				audioInfo.BitRate = bitrate
			}
			
			// Extract language
			if tags, ok := stream["tags"].(map[string]interface{}); ok {
				if language, ok := tags["language"].(string); ok {
					audioInfo.Language = language
				}
			}
			
			videoFile.AudioInfo = append(videoFile.AudioInfo, audioInfo)
		}
	}

	f.Logger.Debug("Video info extraction completed for: %s", filePath)
	return videoFile, nil
}

// ExecuteCommand runs an FFmpeg command with the given arguments
func (f *FFmpeg) ExecuteCommand(args []string) ([]byte, error) {
	// Obter o caminho para o FFmpeg
	info, err := util.FindFFmpeg()
	if err != nil {
		return nil, fmt.Errorf("erro ao encontrar FFmpeg: %v", err)
	}
	
	ffmpegPath := info.Path
	
	f.Logger.Debug("Executing FFmpeg command: %s %s", ffmpegPath, strings.Join(args, " "))
	
	cmd := exec.Command(ffmpegPath, args...)
	return cmd.CombinedOutput()
}

// DetectSceneChanges analyzes a video to detect scene changes
func (f *FFmpeg) DetectSceneChanges(filePath string, threshold float64) ([]float64, error) {
	f.Logger.Debug("Detecting scene changes in: %s", filePath)
	
	// If threshold not specified, use a default value
	if threshold <= 0 {
		threshold = 0.4 // Default threshold
	}
	
	// Use FFmpeg's scene detection filter
	args := []string{
		"-i", filePath,
		"-vf", fmt.Sprintf("select='gt(scene,%f)',metadata=print", threshold),
		"-f", "null",
		"-",
	}
	
	output, err := f.ExecuteCommand(args)
	if err != nil {
		return nil, fmt.Errorf("scene detection failed: %w", err)
	}
	
	// Parse the output to extract scene change timecodes
	sceneChanges := []float64{}
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "scene:") {
			parts := strings.Split(line, "scene:")
			if len(parts) >= 2 {
				sceneValue, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				if err == nil && sceneValue >= threshold {
					// Extract timestamp (this is an approximation, might need refinement)
					timeParts := strings.Split(line, "pts_time:")
					if len(timeParts) >= 2 {
						timeStr := strings.Split(timeParts[1], " ")[0]
						timestamp, err := strconv.ParseFloat(timeStr, 64)
						if err == nil {
							sceneChanges = append(sceneChanges, timestamp)
						}
					}
				}
			}
		}
	}
	
	f.Logger.Debug("Detected %d scene changes", len(sceneChanges))
	return sceneChanges, nil
}

// CalculateFrameComplexity estimates the complexity of video frames
func (f *FFmpeg) CalculateFrameComplexity(filePath string) (float64, error) {
	f.Logger.Debug("Calculating frame complexity for: %s", filePath)
	
	// Use FFmpeg to extract frames and calculate complexity
	// This is a simplified approach using FFmpeg filters
	args := []string{
		"-i", filePath,
		"-vf", "select='eq(pict_type,I)',signalstats=stat=variance",
		"-f", "null",
		"-",
	}
	
	output, err := f.ExecuteCommand(args)
	if err != nil {
		return 0, fmt.Errorf("frame complexity analysis failed: %w", err)
	}
	
	// Parse the output to find the average variance (complexity)
	lines := strings.Split(string(output), "\n")
	var totalVariance float64
	var count int
	
	for _, line := range lines {
		if strings.Contains(line, "variance:") {
			parts := strings.Split(line, "variance:")
			if len(parts) >= 2 {
				variance, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				if err == nil {
					totalVariance += variance
					count++
				}
			}
		}
	}
	
	if count == 0 {
		return 0, fmt.Errorf("no frames analyzed for complexity")
	}
	
	averageComplexity := totalVariance / float64(count)
	f.Logger.Debug("Average frame complexity: %f", averageComplexity)
	
	return averageComplexity, nil
}

// Execute executes the FFmpeg command
func (ffmpeg *FFmpeg) Execute() error {
	// Verificar se o FFmpeg está instalado
	ffmpegInfo, err := util.FindFFmpeg()
	if err != nil {
		ffmpeg.Logger.Error("Falha ao verificar FFmpeg: %v", err)
		ffmpeg.Logger.Info("Tente executar 'compressvideo repair-ffmpeg' para corrigir problemas com o FFmpeg")
		return fmt.Errorf("falha ao inicializar FFmpeg: %v", err)
	}
	
	// Caminho para o FFmpeg
	ffmpegPath := ffmpegInfo.Path
	
	// Obter informações do vídeo original
	video, err := ffmpeg.GetVideoInfo(ffmpeg.InputFile)
	if err != nil {
		// No Windows, erros com código hexadecimal podem indicar problemas com o FFprobe
		if strings.Contains(err.Error(), "0x") || strings.Contains(err.Error(), "exit status") {
			ffmpeg.Logger.Error("Erro ao executar FFprobe. Isso pode indicar um problema com a instalação do FFmpeg.")
			ffmpeg.Logger.Info("Tente executar 'compressvideo repair-ffmpeg' para corrigir problemas com o FFmpeg")
		}
		return fmt.Errorf("falha ao obter informações do vídeo: %v", err)
	}

	// Criar uma estrutura VideoInfo a partir do VideoFile para compatibilidade
	videoInfo := &VideoInfo{
		Width:      video.VideoInfo.Width,
		Height:     video.VideoInfo.Height,
		FrameRate:  video.VideoInfo.FPS,
		Duration:   video.Duration,
		BitRate:    video.BitRate,
		CodecName:  video.VideoInfo.Codec,
		Resolution: fmt.Sprintf("%dx%d", video.VideoInfo.Width, video.VideoInfo.Height),
	}
	
	// Exibir informações do vídeo
	ffmpeg.Logger.Info("Informações do vídeo:")
	ffmpeg.Logger.Info("  Resolução: %dx%d", videoInfo.Width, videoInfo.Height)
	ffmpeg.Logger.Info("  Codec: %s", videoInfo.CodecName)
	ffmpeg.Logger.Info("  Duração: %.2f segundos", videoInfo.Duration)
	if videoInfo.BitRate > 0 {
		ffmpeg.Logger.Info("  Bitrate: %.2f Mbps", float64(videoInfo.BitRate)/1024/1024)
	}
	
	// Calcular configurações de compressão com base na qualidade
	settings := ffmpeg.calculateEncodingSettings(videoInfo, ffmpeg.Options.Quality)

	// Exibir configurações de compressão
	ffmpeg.Logger.Info("Configurações de compressão:")
	ffmpeg.Logger.Info("  Codec: %s", settings.VideoCodec)
	ffmpeg.Logger.Info("  CRF: %d", settings.CRF)
	ffmpeg.Logger.Info("  Preset: %s", settings.Preset)
	if settings.MaxWidth > 0 || settings.MaxHeight > 0 {
		ffmpeg.Logger.Info("  Escala: %dx%d", settings.MaxWidth, settings.MaxHeight)
	}
	if settings.TargetBitrate > 0 {
		ffmpeg.Logger.Info("  Bitrate: %.2f Mbps", float64(settings.TargetBitrate)/1024/1024)
	} else {
		ffmpeg.Logger.Info("  Bitrate: Automático (controlado pelo CRF)")
	}
	
	// Iniciar compressão
	ffmpeg.Logger.Info("Iniciando compressão do vídeo...")
	
	// Construir comando FFmpeg
	args := ffmpeg.buildFFmpegCommand(settings)

	// Add input and output files
	inputArgs := []string{"-i", ffmpeg.InputFile}
	args = append(inputArgs, args...)
	args = append(args, "-y", ffmpeg.OutputFile)

	// Log the full command for debugging
	ffmpeg.Logger.Debug("Executando comando FFmpeg: %s %s", ffmpegPath, strings.Join(args, " "))

	// Execute the command with progress monitoring
	cmd := exec.Command(ffmpegPath, args...)
	
	// Configurar buffers para stdout e stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	
	// Configurar captura de stderr para progresso
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("falha ao capturar saída de erro: %v", err)
	}

	// Iniciar comando
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("falha ao iniciar FFmpeg: %v", err)
	}

	// Mostrar progresso
	progressTracker := util.NewProgressTrackerWithOptions(util.ProgressTrackerOptions{
		Total:          100, // Total de 100% em vez da duração
		Description:    "Comprimindo vídeo",
		Logger:         ffmpeg.Logger,
		ShowPercentage: true,
		ShowSpeed:      false,
	})

	// Ler stderr para mostrar progresso
	buf := make([]byte, 2048)
	lastProgress := int64(0)
	
	// Capturar a saída de stderr para mostrar o progresso e armazenar em stderrBuf
	go func() {
		for {
			n, err := stderr.Read(buf)
			if n == 0 || err != nil {
				break
			}
			
			output := string(buf[:n])
			stderrBuf.WriteString(output)
			
			// Procurar por informações de tempo em qualquer parte da saída
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
						currentTime := parseFFmpegTime(timeStr)
						
						if currentTime > 0 && videoInfo.Duration > 0 {
							// Calcular percentual em vez de usar o tempo diretamente
							percentComplete := int64((currentTime / videoInfo.Duration) * 100.0)
							if percentComplete > 100 {
								percentComplete = 100
							}
							
							// Só atualizar se houver mudança significativa ou for o final
							if percentComplete > lastProgress || percentComplete >= 100 {
								progressTracker.Update(percentComplete)
								lastProgress = percentComplete
							}
						}
					}
				}
			}
			
			// Mostrar erro detalhado em modo verbose
			if ffmpeg.Logger.IsVerbose() && strings.Contains(strings.ToLower(output), "error") {
				ffmpeg.Logger.Debug("FFmpeg: %s", output)
			}
		}
	}()
	
	// Aguardar comando finalizar
	err = cmd.Wait()
	if err != nil {
		// Extrair a mensagem de erro da saída de stderr
		errorMsg := stderrBuf.String()
		return fmt.Errorf("FFmpeg falhou: %v\nDetalhes: %s", err, errorMsg)
	}
	
	progressTracker.Finish()
	
	return nil
}

// DefaultOptions returns default options
func DefaultOptions() *Options {
	return &Options{
		Quality: 3,        // Balanced quality by default
		Preset:  "balanced", // Balanced preset by default
	}
}

// Analisar tempo do FFmpeg (formato HH:MM:SS.ms)
func parseFFmpegTime(timeStr string) float64 {
	// Remover qualquer espaço em branco
	timeStr = strings.TrimSpace(timeStr)
	
	// Verificar se é um formato válido
	if len(timeStr) < 8 { // Pelo menos HH:MM:SS
		return 0
	}
	
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0
	}
	
	hours, err1 := strconv.ParseFloat(parts[0], 64)
	minutes, err2 := strconv.ParseFloat(parts[1], 64)
	
	// Verificar se a parte dos segundos contém ponto decimal
	secondsStr := parts[2]
	
	// Limitar a parte de segundos até 2 casas decimais para evitar problemas de parsing
	if idx := strings.Index(secondsStr, "."); idx >= 0 {
		if len(secondsStr) > idx+3 {
			secondsStr = secondsStr[:idx+3] // Manter até 2 casas decimais
		}
	}
	
	seconds, err3 := strconv.ParseFloat(secondsStr, 64)
	
	// Verificar se houve algum erro no parsing
	if err1 != nil || err2 != nil || err3 != nil {
		return 0
	}
	
	return hours*3600 + minutes*60 + seconds
}

// Calcula as configurações ideais para codificação
func (ffmpeg *FFmpeg) calculateEncodingSettings(video *VideoInfo, quality int) *EncodingSettings {
	settings := &EncodingSettings{}
	
	// Codec padrão é sempre H.264 por compatibilidade
	settings.VideoCodec = "libx264"
	
	// Controle de qualidade baseado na opção escolhida (1-5)
	switch quality {
	case 1: // Máxima compressão
		settings.CRF = 28
		settings.Preset = "faster"
		settings.TargetBitrate = 1000000 // 1 Mbps
		// Para máxima compressão, podemos reduzir a resolução
		if video.Height > 720 {
			settings.MaxHeight = 720
			// Não usar 0 para a largura para evitar problemas com o FFmpeg
			// Calcular a largura proporcional com base na altura original e na nova altura
			if video.Width > 0 && video.Height > 0 {
				// Calcular a proporção e aplicar à nova altura
				aspectRatio := float64(video.Width) / float64(video.Height)
				settings.MaxWidth = int(float64(settings.MaxHeight) * aspectRatio)
				// Garantir que a largura seja um número par (requisito de alguns codecs)
				settings.MaxWidth = settings.MaxWidth - (settings.MaxWidth % 2)
			} else {
				// Fallback seguro se não tivermos dimensões válidas
				settings.MaxWidth = 1280 // Largura comum para 720p (proporção 16:9)
			}
		}
	case 2: // Alta compressão
		settings.CRF = 26
		settings.Preset = "medium"
		settings.TargetBitrate = 2000000 // 2 Mbps
		if video.Height > 1080 {
			settings.MaxHeight = 1080
			// Calcular a largura proporcional da mesma forma que para qualidade 1
			if video.Width > 0 && video.Height > 0 {
				aspectRatio := float64(video.Width) / float64(video.Height)
				settings.MaxWidth = int(float64(settings.MaxHeight) * aspectRatio)
				// Garantir que a largura seja um número par
				settings.MaxWidth = settings.MaxWidth - (settings.MaxWidth % 2)
			} else {
				// Fallback seguro
				settings.MaxWidth = 1920 // Largura comum para 1080p (proporção 16:9)
			}
		}
	case 3: // Balanceado (padrão)
		settings.CRF = 23
		settings.Preset = "medium"
		settings.TargetBitrate = 0 // Deixar o CRF controlar
	case 4: // Alta qualidade
		settings.CRF = 20
		settings.Preset = "slow"
		settings.TargetBitrate = 0
	case 5: // Máxima qualidade
		settings.CRF = 18
		settings.Preset = "slow"
		settings.TargetBitrate = 0
	default: // Caso valor fora do intervalo
		settings.CRF = 23
		settings.Preset = "medium"
		settings.TargetBitrate = 0
	}
	
	// Adaptar preset conforme opção do usuário
	switch ffmpeg.Options.Preset {
	case "fast":
		settings.Preset = "faster"
	case "thorough":
		settings.Preset = "slow"
		// Se estivermos usando bitrate alvo com qualidade baixa e preset thorough,
		// é melhor remover o bitrate alvo para evitar conflitos
		if quality <= 2 {
			settings.TargetBitrate = 0 // Deixar o CRF controlar a qualidade
		}
	}
	
	return settings
}

// Constrói o comando para o FFmpeg
func (ffmpeg *FFmpeg) buildFFmpegCommand(settings *EncodingSettings) []string {
	// Remover o arquivo de entrada e saída, já que serão adicionados separadamente na função Execute
	args := []string{
		"-c:v", settings.VideoCodec,
		"-crf", fmt.Sprintf("%d", settings.CRF),
		"-preset", settings.Preset,
	}
	
	// Adicionar controle de bitrate se especificado
	if settings.TargetBitrate > 0 {
		args = append(args, "-maxrate", fmt.Sprintf("%d", settings.TargetBitrate))
		args = append(args, "-bufsize", fmt.Sprintf("%d", settings.TargetBitrate*2))
	}
	
	// Adicionar escala se especificada - garantir que não criamos filtros inválidos
	if settings.MaxWidth > 0 || settings.MaxHeight > 0 {
		// Verificar se temos valores válidos para as dimensões
		if settings.MaxWidth <= 0 {
			settings.MaxWidth = -2  // Usar -2 para manter proporção
		}
		
		if settings.MaxHeight <= 0 {
			settings.MaxHeight = -2  // Usar -2 para manter proporção
		}
		
		// Se ambas as dimensões são -2, isso é inválido, definir um padrão
		if settings.MaxWidth == -2 && settings.MaxHeight == -2 {
			settings.MaxWidth = 1280
			settings.MaxHeight = 720
		}
		
		// Aplicar o filtro de escala
		scaleFilter := fmt.Sprintf("scale=%d:%d", settings.MaxWidth, settings.MaxHeight)
		args = append(args, "-vf", scaleFilter)
	}
	
	// Copiar áudio
	args = append(args, "-c:a", "aac", "-b:a", "128k")
	
	// Configurações de performance
	args = append(args, "-movflags", "+faststart")
	
	return args
} 