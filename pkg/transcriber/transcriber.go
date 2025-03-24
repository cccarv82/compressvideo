// Package transcriber provides functionality to transcribe video audio to text
package transcriber

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"runtime"

	"github.com/cccarv82/compressvideo/pkg/util"
)

// TranscriptionOptions define as opções para transcrição
type TranscriptionOptions struct {
	// Idioma do áudio (default: "auto" para detecção automática)
	Language string
	
	// Formato de saída (txt, srt, vtt)
	OutputFormat string
	
	// Modelo a ser usado (tiny, base, small, medium, large)
	Model string
	
	// Mostrar timestamps
	ShowTimestamps bool
	
	// Forçar transcrição mesmo se o áudio não for detectado (útil no Windows)
	ForceAudio bool
}

// TranscriptionResult contém informações sobre o resultado da transcrição
type TranscriptionResult struct {
	// Arquivo de entrada
	InputFile string
	
	// Arquivo de saída da transcrição
	OutputFile string
	
	// Tempo de processamento
	ProcessingTime time.Duration
	
	// Idioma detectado
	DetectedLanguage string
	
	// Número de palavras transcritas
	WordCount int
	
	// Erro ocorrido (se houver)
	Error error
}

// Transcriber gerencia a transcrição de vídeos
type Transcriber struct {
	// Caminho para o executável do FFmpeg
	FFmpegPath string
	
	// Caminho para o executável do Whisper
	WhisperPath string
	
	// Logger para saída
	Logger *util.Logger
	
	// Diretório temporário para arquivos de processamento
	TempDir string
}

// NewTranscriber cria uma nova instância do Transcriber
func NewTranscriber(logger *util.Logger) (*Transcriber, error) {
	// Obter o caminho para o FFmpeg
	ffmpegInfo, err := util.FindFFmpeg()
	if err != nil {
		return nil, fmt.Errorf("erro ao encontrar FFmpeg: %v", err)
	}
	
	// Verificar se o Whisper está instalado
	whisperPath, err := findWhisper()
	if err != nil {
		return nil, fmt.Errorf("erro ao encontrar Whisper: %v", err)
	}
	
	// Criar diretório temporário
	tempDir := filepath.Join(os.TempDir(), "compressvideo-transcribe")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório temporário: %v", err)
	}
	
	return &Transcriber{
		FFmpegPath:  ffmpegInfo.Path,
		WhisperPath: whisperPath,
		Logger:      logger,
		TempDir:     tempDir,
	}, nil
}

// DefaultOptions retorna as opções padrão para transcrição
func DefaultOptions() TranscriptionOptions {
	return TranscriptionOptions{
		Language:      "auto",
		OutputFormat:  "txt",
		Model:         "base",
		ShowTimestamps: false,
		ForceAudio:     false,
	}
}

// findWhisper procura pelo executável do Whisper no sistema
func findWhisper() (string, error) {
	// Primeiro, verificar se whisper-cpp está instalado
	if path, err := exec.LookPath("whisper-cpp"); err == nil {
		return path, nil
	}
	
	// Verificar se whisper.cpp está instalado
	if path, err := exec.LookPath("whisper.cpp"); err == nil {
		return path, nil
	}
	
	// Verificar se whisper está instalado como comando Python
	if path, err := exec.LookPath("whisper"); err == nil {
		return path, nil
	}
	
	// Verificar o caminho personalizado no diretório .compressvideo
	homeDir, err := os.UserHomeDir()
	if err == nil {
		// Verificar whisper-cpp local
		whisperPath := filepath.Join(homeDir, ".compressvideo", "bin", "whisper-cpp")
		if runtime.GOOS == "windows" {
			whisperPath += ".exe"
		}
		if _, err := os.Stat(whisperPath); err == nil {
			return whisperPath, nil
		}
		
		// Verificar wrapper do whisper local
		whisperPath = filepath.Join(homeDir, ".compressvideo", "bin", "whisper")
		if runtime.GOOS == "windows" {
			whisperPath += ".bat"
		}
		if _, err := os.Stat(whisperPath); err == nil {
			return whisperPath, nil
		}
	}
	
	// Se chegou aqui, o Whisper não está instalado
	return "", fmt.Errorf("não foi possível encontrar o Whisper instalado. Use 'compressvideo repair-whisper' para instalar automaticamente")
}

// Transcribe transcreve o áudio de um vídeo para texto
func (t *Transcriber) Transcribe(
	inputFile, outputFile string, 
	options TranscriptionOptions, 
	progress *util.ProgressTracker) (*TranscriptionResult, error) {
	
	startTime := time.Now()
	
	// Se o arquivo de saída não foi especificado, usar o mesmo nome do arquivo de entrada
	// mas com a extensão .txt
	if outputFile == "" {
		ext := filepath.Ext(inputFile)
		baseName := strings.TrimSuffix(inputFile, ext)
		outputFile = baseName + ".txt"
	}
	
	// Verificar se o arquivo de saída já existe
	if _, err := os.Stat(outputFile); err == nil {
		// O arquivo existe, verificar se devemos sobrescrever
		return nil, fmt.Errorf("o arquivo de saída já existe")
	}
	
	// Verificar se estamos no Windows para fornecer instruções personalizadas
	if runtime.GOOS == "windows" && t.WhisperPath == "" {
		// No Windows, oferecer instruções específicas para resolver problemas comuns
		t.Logger.Warning("Não foi possível encontrar Whisper no Windows.")
		t.Logger.Info("Tente executar o comando abaixo com a opção para Windows:")
		t.Logger.Info("  compressvideo repair-whisper --ctranslate2")
		t.Logger.Info("Se continuar tendo problemas, instale manualmente com:")
		t.Logger.Info("  pip install -U whisper-ctranslate2")
		t.Logger.Info("Em seguida, reinicie o CompressVideo.")
		
		return nil, fmt.Errorf("Whisper não encontrado. Use 'compressvideo repair-whisper --ctranslate2' no Windows")
	}
	
	// Iniciar a transcrição
	t.Logger.Info("Transcrevendo vídeo: %s", inputFile)
	t.Logger.Info("Arquivo de saída: %s", outputFile)
	
	// Extrair o áudio do vídeo
	audioFile, err := t.extractAudio(inputFile, options, progress)
	if err != nil {
		// Verificar se é o erro específico de falta de áudio
		if strings.Contains(err.Error(), "não contém uma faixa de áudio") {
			t.Logger.Error("O vídeo não contém faixa de áudio!")
			t.Logger.Info("Para transcrever um vídeo, ele precisa conter uma faixa de áudio.")
			return nil, fmt.Errorf("o vídeo não contém uma faixa de áudio para transcrição. Por favor, use um vídeo que contenha áudio")
		}
		return nil, fmt.Errorf("erro ao extrair áudio: %w", err)
	}
	defer os.Remove(audioFile) // Limpar arquivo de áudio temporário
	
	// 2. Transcrever o áudio para texto
	transcription, detectedLang, wordCount, err := t.transcribeAudio(audioFile, options, progress)
	if err != nil {
		return nil, fmt.Errorf("erro ao transcrever áudio: %v", err)
	}
	
	// 3. Salvar a transcrição em um arquivo
	if err := os.WriteFile(outputFile, []byte(transcription), 0644); err != nil {
		return nil, fmt.Errorf("erro ao salvar transcrição: %v", err)
	}
	
	// 4. Criar e retornar o resultado
	result := &TranscriptionResult{
		InputFile:        inputFile,
		OutputFile:       outputFile,
		ProcessingTime:   time.Since(startTime),
		DetectedLanguage: detectedLang,
		WordCount:        wordCount,
	}
	
	return result, nil
}

// extractAudio extrai o áudio de um vídeo usando FFmpeg
func (t *Transcriber) extractAudio(videoFile string, options TranscriptionOptions, progress *util.ProgressTracker) (string, error) {
	t.Logger.Info("Extraindo áudio do vídeo...")
	
	// Verificar se o vídeo tem uma faixa de áudio (pular se ForceAudio estiver ativo)
	if !options.ForceAudio {
		hasAudio, err := t.checkForAudioStream(videoFile)
		if err != nil {
			// No Windows, tentar um método alternativo ou pular verificação
			if util.IsWindows() {
				t.Logger.Warning("Problema ao verificar faixa de áudio: %v", err)
				t.Logger.Info("Tentando método alternativo para verificar áudio...")
				
				// Tentar método alternativo usando FFmpeg
				hasAudioAlt, altErr := t.checkForAudioStreamAlternative(videoFile)
				if altErr != nil {
					t.Logger.Warning("Método alternativo também falhou: %v", altErr)
					t.Logger.Info("Assumindo que o vídeo tem áudio e tentando prosseguir...")
				} else if !hasAudioAlt {
					return "", fmt.Errorf("o vídeo não contém uma faixa de áudio para transcrição")
				}
				// Se o método alternativo funcionou e encontrou áudio, prosseguir
			} else {
				return "", fmt.Errorf("erro ao verificar faixa de áudio: %w", err)
			}
		} else if !hasAudio {
			return "", fmt.Errorf("o vídeo não contém uma faixa de áudio para transcrição")
		}
	} else {
		t.Logger.Info("Flag ForceAudio ativa - pulando verificação de áudio")
	}
	
	// Criar um arquivo temporário para o áudio
	audioFile, err := os.CreateTemp("", "audio-*.wav")
	if err != nil {
		return "", fmt.Errorf("erro ao criar arquivo temporário para áudio: %w", err)
	}
	audioFile.Close()
	
	// Garantir que o arquivo temporário tenha a extensão .wav
	tempAudioPath := audioFile.Name()
	if !strings.HasSuffix(tempAudioPath, ".wav") {
		os.Remove(tempAudioPath)
		tempAudioPath = tempAudioPath + ".wav"
	}
	
	// Extrair áudio do vídeo usando FFmpeg
	t.Logger.Info("Extraindo faixa de áudio do vídeo...")
	
	args := []string{
		"-y",              // Sobrescrever arquivos de saída sem perguntar
		"-i", videoFile,   // Arquivo de entrada
		"-vn",             // Sem vídeo
		"-acodec", "pcm_s16le", // Codec de áudio
		"-ar", "16000",    // Taxa de amostragem
		"-ac", "1",        // Mono
		tempAudioPath,     // Arquivo de saída
	}
	
	cmd := exec.Command(t.FFmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tempAudioPath) // Limpar arquivo temporário em caso de erro
		t.Logger.Error("Saída do FFmpeg: %s", string(output))
		return "", fmt.Errorf("erro ao extrair áudio: %w", err)
	}
	
	// Verificar se o arquivo de áudio foi criado e não está vazio
	fileInfo, err := os.Stat(tempAudioPath)
	if err != nil {
		return "", fmt.Errorf("erro ao verificar arquivo de áudio: %w", err)
	}
	
	if fileInfo.Size() == 0 {
		os.Remove(tempAudioPath)
		return "", fmt.Errorf("o arquivo de áudio extraído está vazio; o vídeo provavelmente não tem áudio")
	}
	
	t.Logger.Success("Áudio extraído com sucesso: %s (%.2f MB)",
		tempAudioPath, float64(fileInfo.Size())/(1024*1024))
	
	// Atualizar progresso
	if progress != nil {
		progress.Update(30) // 30% do progresso total após extrair o áudio
	}
	
	return tempAudioPath, nil
}

// checkForAudioStream verifica se o vídeo tem uma faixa de áudio
func (t *Transcriber) checkForAudioStream(videoFile string) (bool, error) {
	// Obter o caminho do ffprobe a partir do caminho do ffmpeg
	ffprobePath := t.FFmpegPath
	
	// Lidar melhor com extensões no Windows
	if util.IsWindows() {
		// No Windows, devemos tratar as extensões .exe
		if strings.HasSuffix(ffprobePath, "ffmpeg.exe") {
			ffprobePath = strings.TrimSuffix(ffprobePath, "ffmpeg.exe") + "ffprobe.exe"
		} else if strings.HasSuffix(ffprobePath, "ffmpeg") {
			// Talvez seja um caminho no WSL ou um arquivo sem extensão
			ffprobePath = strings.TrimSuffix(ffprobePath, "ffmpeg") + "ffprobe"
		}
	} else {
		// Em sistemas Unix, a extensão normalmente não é usada
		if strings.HasSuffix(ffprobePath, "ffmpeg") {
			ffprobePath = strings.TrimSuffix(ffprobePath, "ffmpeg") + "ffprobe"
		}
	}
	
	t.Logger.Debug("Usando ffprobe: %s", ffprobePath)
	
	// Usar FFprobe para listar todos os tipos de stream
	args := []string{
		"-v", "error",
		"-select_streams", "a",  // Selecionar apenas streams de áudio
		"-show_entries", "stream=codec_type",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoFile,
	}
	
	t.Logger.Debug("Executando comando ffprobe: %s %v", ffprobePath, args)
	cmd := exec.Command(ffprobePath, args...)
	output, err := cmd.CombinedOutput()
	
	// Log detalhado para depuração
	if err != nil {
		t.Logger.Warning("Não foi possível verificar a faixa de áudio com ffprobe: %v", err)
		if len(output) > 0 {
			t.Logger.Debug("Saída do ffprobe: %s", string(output))
		}
		return false, err
	}
	
	outputStr := string(output)
	t.Logger.Debug("Saída do ffprobe: %s", outputStr)
	
	// Se a saída contém "audio", então há pelo menos uma faixa de áudio
	hasAudio := strings.Contains(outputStr, "audio")
	if hasAudio {
		t.Logger.Info("Faixa de áudio detectada no vídeo")
	} else {
		t.Logger.Warning("Nenhuma faixa de áudio detectada no vídeo")
	}
	
	return hasAudio, nil
}

// checkForAudioStreamAlternative tenta extrair uma amostra pequena de áudio para verificar se o vídeo tem áudio
func (t *Transcriber) checkForAudioStreamAlternative(videoFile string) (bool, error) {
	t.Logger.Info("Usando método alternativo para verificar áudio...")
	
	// Criar arquivo temporário para amostra de áudio
	sampleFile, err := os.CreateTemp("", "audio-sample-*.wav")
	if err != nil {
		return false, fmt.Errorf("erro ao criar arquivo temporário para amostra de áudio: %w", err)
	}
	samplePath := sampleFile.Name()
	sampleFile.Close()
	
	// Tentar extrair apenas 1 segundo de áudio
	args := []string{
		"-y",              // Sobrescrever arquivos de saída sem perguntar
		"-i", videoFile,   // Arquivo de entrada
		"-vn",             // Sem vídeo
		"-acodec", "pcm_s16le", // Codec de áudio
		"-ar", "16000",    // Taxa de amostragem
		"-ac", "1",        // Mono
		"-t", "1",         // Extrair apenas 1 segundo
		samplePath,        // Arquivo de saída
	}
	
	cmd := exec.Command(t.FFmpegPath, args...)
	output, err := cmd.CombinedOutput()
	
	// Verificar se o arquivo foi criado e tem algum conteúdo
	defer os.Remove(samplePath) // Limpar arquivo temporário
	
	if err != nil {
		t.Logger.Warning("Erro ao extrair amostra de áudio: %v", err)
		t.Logger.Debug("Saída do FFmpeg: %s", string(output))
		return false, err
	}
	
	// Verificar se o arquivo tem conteúdo
	fileInfo, err := os.Stat(samplePath)
	if err != nil {
		return false, err
	}
	
	// Se o arquivo tem algum tamanho significativo, assumimos que tem áudio
	hasAudio := fileInfo.Size() > 44 // 44 bytes é o tamanho do cabeçalho WAV
	if hasAudio {
		t.Logger.Info("Áudio detectado via método alternativo")
	} else {
		t.Logger.Warning("Nenhum áudio detectado via método alternativo")
	}
	
	return hasAudio, nil
}

// transcribeAudio transcreve um arquivo de áudio para texto usando Whisper
func (t *Transcriber) transcribeAudio(audioFile string, options TranscriptionOptions, progress *util.ProgressTracker) (string, string, int, error) {
	t.Logger.Info("Transcrevendo áudio...")
	
	// Verificar se estamos no Windows
	isWindows := util.IsWindows()
	
	// No Windows, tentar o modo direto primeiro
	if isWindows {
		t.Logger.Info("Usando método direto para transcrição no Windows...")
		result, detectedLang, wordCount, err := t.transcribeDirectPython(audioFile, options, progress)
		if err == nil {
			return result, detectedLang, wordCount, nil
		}
		t.Logger.Warning("Método direto falhou: %v. Tentando abordagem normal...", err)
	}
	
	// Se não é Windows ou o método direto falhou, usar a abordagem padrão
	if strings.Contains(t.WhisperPath, "whisper-cpp") || strings.Contains(t.WhisperPath, "whisper.cpp") {
		return t.transcribeWithWhisperCPP(audioFile, options, progress)
	} else {
		return t.transcribeWithWhisperPython(audioFile, options, progress)
	}
}

// transcribeWithWhisperCPP transcreve usando a implementação C++ do Whisper
func (t *Transcriber) transcribeWithWhisperCPP(audioFile string, options TranscriptionOptions, progress *util.ProgressTracker) (string, string, int, error) {
	// Construir o comando para o whisper.cpp
	args := []string{
		audioFile,
		"--model", options.Model,
		"--output-txt",
	}
	
	// Adicionar idioma se não for auto
	if options.Language != "auto" {
		args = append(args, "--language", options.Language)
	}
	
	// Adicionar timestamps se solicitado
	if options.ShowTimestamps {
		args = append(args, "--print-timestamps")
	}
	
	// Executar o comando
	cmd := exec.Command(t.WhisperPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", 0, fmt.Errorf("erro ao transcrever com Whisper: %v\nOutput: %s", err, string(output))
	}
	
	// Extrair a transcrição do output
	transcription := string(output)
	
	// Identificar idioma detectado (simplificado, teria que ser adaptado conforme a saída real)
	detectedLang := "desconhecido"
	if strings.Contains(transcription, "Detected language:") {
		parts := strings.Split(transcription, "Detected language:")
		if len(parts) > 1 {
			langPart := strings.Split(parts[1], "\n")[0]
			detectedLang = strings.TrimSpace(langPart)
		}
	}
	
	// Contagem aproximada de palavras
	wordCount := len(strings.Fields(transcription))
	
	// Atualizar progresso
	if progress != nil {
		progress.Update(100) // 100% completo
	}
	
	return transcription, detectedLang, wordCount, nil
}

// transcribeWithWhisperPython transcreve usando a implementação Python do Whisper
func (t *Transcriber) transcribeWithWhisperPython(audioFile string, options TranscriptionOptions, progress *util.ProgressTracker) (string, string, int, error) {
	// Verificar se estamos no Windows
	isWindows := util.IsWindows()
	
	// Log detalhado para diagnóstico
	t.Logger.Debug("Usando WhisperPath: %s", t.WhisperPath)
	
	// Para o Windows, verificar se o arquivo é um .bat
	isBatchFile := isWindows && strings.HasSuffix(t.WhisperPath, ".bat")
	
	// Construir o comando para o whisper python
	args := []string{
		audioFile,
		"--model", options.Model,
		"--output_format", "txt",
	}
	
	// Adicionar idioma se não for auto
	if options.Language != "auto" {
		args = append(args, "--language", options.Language)
	}
	
	// Adicionar word timestamps se solicitado
	if options.ShowTimestamps {
		args = append(args, "--word_timestamps", "True")
	}
	
	// Log de comando para diagnóstico
	t.Logger.Debug("Comando a ser executado: %s %v", t.WhisperPath, args)
	
	var cmd *exec.Cmd
	var output []byte
	var err error
	
	if isWindows {
		if isBatchFile {
			// Para arquivos batch no Windows, precisamos usar cmd.exe
			t.Logger.Debug("Executando via cmd.exe porque o whisper é um arquivo .bat")
			cmdArgs := []string{"/C", t.WhisperPath}
			cmdArgs = append(cmdArgs, args...)
			cmd = exec.Command("cmd.exe", cmdArgs...)
		} else {
			// Para executáveis normais no Windows
			cmd = exec.Command(t.WhisperPath, args...)
		}
		
		// Definir o ambiente de execução para Windows
		// Para garantir que todas as variáveis de ambiente estejam disponíveis
		cmd.Env = os.Environ()
		
		// Tentar usar diretório de trabalho do executável para garantir 
		// que as DLLs e dependências sejam encontradas
		if !isBatchFile {
			cmd.Dir = filepath.Dir(t.WhisperPath)
		}
	} else {
		// Para sistemas Unix, executar normalmente
		cmd = exec.Command(t.WhisperPath, args...)
	}
	
	// Executar o comando
	t.Logger.Debug("Iniciando execução do Whisper...")
	output, err = cmd.CombinedOutput()
	
	// Log da saída para diagnóstico
	if len(output) > 0 {
		t.Logger.Debug("Saída do comando Whisper: %s", string(output))
	}
	
	if err != nil {
		errMsg := fmt.Sprintf("erro ao transcrever com Whisper Python: %v", err)
		
		// Adicionar mais informações no Windows para diagnóstico
		if isWindows {
			errMsg += fmt.Sprintf("\nWindows error code: %v", err)
			// Verificar se o erro é relacionado a permissões ou ambiente
			if exitErr, ok := err.(*exec.ExitError); ok {
				errMsg += fmt.Sprintf("\nExit code: 0x%x", exitErr.ExitCode())
			}
		}
		
		// Adicionar a saída para diagnóstico
		if len(output) > 0 {
			errMsg += fmt.Sprintf("\nOutput:\n%s", string(output))
		}
		
		return "", "", 0, fmt.Errorf(errMsg)
	}
	
	// O Whisper Python cria um arquivo de texto com o mesmo nome do arquivo de entrada
	outputTxtFile := strings.TrimSuffix(audioFile, filepath.Ext(audioFile)) + ".txt"
	t.Logger.Debug("Verificando arquivo de saída: %s", outputTxtFile)
	
	// Verificar se o arquivo de saída existe
	if _, err := os.Stat(outputTxtFile); os.IsNotExist(err) {
		t.Logger.Warning("Arquivo de saída não encontrado: %s", outputTxtFile)
		
		// Tentar encontrar o arquivo em outro local (whisper às vezes tem comportamentos diferentes)
		// Verificar no diretório atual
		currDir, _ := os.Getwd()
		altOutputFile := filepath.Join(currDir, filepath.Base(strings.TrimSuffix(audioFile, filepath.Ext(audioFile))+".txt"))
		t.Logger.Debug("Tentando arquivo alternativo: %s", altOutputFile)
		
		if _, err := os.Stat(altOutputFile); err == nil {
			outputTxtFile = altOutputFile
			t.Logger.Debug("Usando arquivo alternativo encontrado: %s", outputTxtFile)
		} else {
			// Se não encontrou, tentar usar a saída direta do comando
			t.Logger.Warning("Arquivo alternativo também não encontrado. Usando saída do comando.")
			// Criar arquivo temporário com a saída
			tempFile, err := os.CreateTemp("", "whisper-output-*.txt")
			if err == nil {
				defer tempFile.Close()
				tempFile.WriteString(string(output))
				outputTxtFile = tempFile.Name()
				t.Logger.Debug("Criado arquivo temporário com a saída: %s", outputTxtFile)
			} else {
				return "", "", 0, fmt.Errorf("não foi possível criar arquivo temporário para a saída: %v", err)
			}
		}
	}
	
	// Ler o arquivo de transcrição
	transcription, err := os.ReadFile(outputTxtFile)
	if err != nil {
		return "", "", 0, fmt.Errorf("erro ao ler arquivo de transcrição: %v", err)
	}
	
	// Limpar o arquivo temporário
	defer os.Remove(outputTxtFile)
	
	// Extrair idioma detectado (o Python whisper também mostra isso)
	detectedLang := "desconhecido"
	if strings.Contains(string(output), "Detected language:") {
		parts := strings.Split(string(output), "Detected language:")
		if len(parts) > 1 {
			langPart := strings.Split(parts[1], "\n")[0]
			detectedLang = strings.TrimSpace(langPart)
		}
	}
	
	// Contagem de palavras
	wordCount := len(strings.Fields(string(transcription)))
	
	// Atualizar progresso
	if progress != nil {
		progress.Update(100) // 100% completo
	}
	
	return string(transcription), detectedLang, wordCount, nil
}

// transcribeDirectPython tenta transcrever o áudio chamando o módulo Python diretamente
func (t *Transcriber) transcribeDirectPython(audioFile string, options TranscriptionOptions, progress *util.ProgressTracker) (string, string, int, error) {
	// Encontrar o Python através do método mais confiável
	pythonPath, err := findPythonDirect()
	if err != nil {
		return "", "", 0, fmt.Errorf("não foi possível encontrar o Python: %w", err)
	}
	
	t.Logger.Debug("Usando Python direto: %s", pythonPath)
	
	// Construir a linha de comando para chamar o whisper_ctranslate2 diretamente
	var args []string
	
	// Primeiro tentar com whisper_ctranslate2
	args = []string{
		"-c",
		fmt.Sprintf(`
import sys
try:
    import whisper_ctranslate2
    audio_file = %q
    model_name = %q
    
    # Configurar o modelo
    model = whisper_ctranslate2.load_model(%q)
    
    # Gerar transcrição
    result = model.transcribe(audio_file)
    
    # Gravar a saída
    output_file = %q
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(result["text"])
    
    # Imprimir informações para o log
    print(f"Detected language: {result.get('language', 'unknown')}")
    print(f"Transcription complete. Written to {output_file}")
    
except ImportError:
    print("whisper_ctranslate2 not found, will try alternative")
    sys.exit(2)
except Exception as e:
    print(f"Error during transcription: {e}")
    sys.exit(1)
`, audioFile, options.Model, options.Model, strings.TrimSuffix(audioFile, filepath.Ext(audioFile))+".txt"),
	}
	
	// Criar o comando
	cmd := exec.Command(pythonPath, args...)
	
	// Definir o ambiente apenas com variáveis essenciais para evitar conflitos
	cleanEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"SYSTEMROOT=" + os.Getenv("SYSTEMROOT"),
		"TEMP=" + os.Getenv("TEMP"),
		"TMP=" + os.Getenv("TMP"),
		"USERPROFILE=" + os.Getenv("USERPROFILE"),
		"PYTHONIOENCODING=utf-8",
	}
	
	cmd.Env = cleanEnv
	
	// Capturar saída
	t.Logger.Debug("Executando transcrição direta...")
	output, err := cmd.CombinedOutput()
	
	// Log da saída para diagnóstico
	outputStr := string(output)
	t.Logger.Debug("Saída da transcrição direta: %s", outputStr)
	
	if err != nil {
		// Se for o código 2, significa que whisper_ctranslate2 não está instalado
		// Vamos tentar com openai-whisper
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			t.Logger.Info("whisper_ctranslate2 não encontrado, tentando com openai-whisper...")
			return t.transcribeDirectWithOpenAIWhisper(pythonPath, audioFile, options, progress)
		}
		
		return "", "", 0, fmt.Errorf("erro na transcrição direta: %v\nOutput: %s", err, outputStr)
	}
	
	// Ler o arquivo de saída
	outputFile := strings.TrimSuffix(audioFile, filepath.Ext(audioFile)) + ".txt"
	transcription, err := os.ReadFile(outputFile)
	if err != nil {
		return "", "", 0, fmt.Errorf("erro ao ler transcrição: %w", err)
	}
	
	// Extrair idioma detectado
	detectedLang := "desconhecido"
	if strings.Contains(outputStr, "Detected language:") {
		parts := strings.Split(outputStr, "Detected language:")
		if len(parts) > 1 {
			langPart := strings.Split(parts[1], "\n")[0]
			detectedLang = strings.TrimSpace(langPart)
		}
	}
	
	// Contagem de palavras
	wordCount := len(strings.Fields(string(transcription)))
	
	// Atualizar progresso
	if progress != nil {
		progress.Update(100) // 100% completo
	}
	
	return string(transcription), detectedLang, wordCount, nil
}

// transcribeDirectWithOpenAIWhisper tenta transcrever usando o módulo openai-whisper diretamente
func (t *Transcriber) transcribeDirectWithOpenAIWhisper(pythonPath, audioFile string, options TranscriptionOptions, progress *util.ProgressTracker) (string, string, int, error) {
	// Script Python para usar openai-whisper diretamente
	args := []string{
		"-c",
		fmt.Sprintf(`
import sys
try:
    import whisper
    audio_file = %q
    model_name = %q
    language = %q
    
    # Carregar o modelo
    model = whisper.load_model(model_name)
    
    # Definir opções de decodificação
    decode_options = {}
    if language != "auto":
        decode_options["language"] = language
    
    # Gerar transcrição
    result = model.transcribe(audio_file, **decode_options)
    
    # Gravar a saída
    output_file = %q
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(result["text"])
    
    # Imprimir informações para o log
    print(f"Detected language: {result.get('language', 'unknown')}")
    print(f"Transcription complete. Written to {output_file}")
    
except ImportError:
    print("openai-whisper not found. Please install with: pip install openai-whisper")
    sys.exit(1)
except Exception as e:
    print(f"Error during transcription: {e}")
    sys.exit(1)
`, audioFile, options.Model, options.Language, strings.TrimSuffix(audioFile, filepath.Ext(audioFile))+".txt"),
	}
	
	// Criar o comando
	cmd := exec.Command(pythonPath, args...)
	
	// Definir o ambiente apenas com variáveis essenciais
	cleanEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"SYSTEMROOT=" + os.Getenv("SYSTEMROOT"),
		"TEMP=" + os.Getenv("TEMP"),
		"TMP=" + os.Getenv("TMP"),
		"USERPROFILE=" + os.Getenv("USERPROFILE"),
		"PYTHONIOENCODING=utf-8",
	}
	
	cmd.Env = cleanEnv
	
	// Capturar saída
	t.Logger.Debug("Executando transcrição direta com openai-whisper...")
	output, err := cmd.CombinedOutput()
	
	// Log da saída para diagnóstico
	outputStr := string(output)
	t.Logger.Debug("Saída da transcrição com openai-whisper: %s", outputStr)
	
	if err != nil {
		return "", "", 0, fmt.Errorf("erro na transcrição com openai-whisper: %v\nOutput: %s", err, outputStr)
	}
	
	// Ler o arquivo de saída
	outputFile := strings.TrimSuffix(audioFile, filepath.Ext(audioFile)) + ".txt"
	transcription, err := os.ReadFile(outputFile)
	if err != nil {
		return "", "", 0, fmt.Errorf("erro ao ler transcrição: %w", err)
	}
	
	// Extrair idioma detectado
	detectedLang := "desconhecido"
	if strings.Contains(outputStr, "Detected language:") {
		parts := strings.Split(outputStr, "Detected language:")
		if len(parts) > 1 {
			langPart := strings.Split(parts[1], "\n")[0]
			detectedLang = strings.TrimSpace(langPart)
		}
	}
	
	// Contagem de palavras
	wordCount := len(strings.Fields(string(transcription)))
	
	// Atualizar progresso
	if progress != nil {
		progress.Update(100) // 100% completo
	}
	
	return string(transcription), detectedLang, wordCount, nil
}

// findPythonDirect encontra o Python usando métodos mais robustos para o Windows
func findPythonDirect() (string, error) {
	// Primeiro, tentar os métodos de busca normais
	pythonPath, err := findPythonCommand()
	if err == nil && pythonPath != "" {
		return pythonPath, nil
	}
	
	// Se estamos no Windows, tentar caminhos comuns diretamente
	if util.IsWindows() {
		// Caminhos comuns onde o Python pode estar instalado
		commonPaths := []string{
			"C:\\Python310\\python.exe",
			"C:\\Python39\\python.exe",
			"C:\\Python38\\python.exe",
			"C:\\Python37\\python.exe",
			"C:\\Python36\\python.exe",
			"C:\\Program Files\\Python310\\python.exe",
			"C:\\Program Files\\Python39\\python.exe",
			"C:\\Program Files\\Python38\\python.exe",
			"C:\\Program Files (x86)\\Python310\\python.exe",
			"C:\\Program Files (x86)\\Python39\\python.exe",
			"C:\\Program Files (x86)\\Python38\\python.exe",
		}
		
		// Verificar cada caminho diretamente
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				// Testar se o Python é funcional
				cmd := exec.Command(path, "--version")
				if err := cmd.Run(); err == nil {
					return path, nil
				}
			}
		}
		
		// Verificar registro do Windows usando 'reg query'
		cmd := exec.Command("reg", "query", "HKEY_LOCAL_MACHINE\\SOFTWARE\\Python\\PythonCore", "/s")
		output, err := cmd.CombinedOutput()
		if err == nil {
			// Procurar por caminhos de instalação nos resultados
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "installpath") && strings.Contains(line, "REG_SZ") {
					parts := strings.SplitN(line, "REG_SZ", 2)
					if len(parts) > 1 {
						possiblePath := strings.TrimSpace(parts[1]) + "\\python.exe"
						if _, err := os.Stat(possiblePath); err == nil {
							return possiblePath, nil
						}
					}
				}
			}
		}
	}
	
	return "", fmt.Errorf("não foi possível encontrar o Python")
}

// DisplayTranscriptionResult exibe o resultado da transcrição no console
func (t *Transcriber) DisplayTranscriptionResult(result *TranscriptionResult) {
	if result == nil {
		return
	}
	
	t.Logger.Section("Resultado da Transcrição")
	t.Logger.Field("Arquivo de Entrada", "%s", result.InputFile)
	t.Logger.Field("Arquivo de Saída", "%s", result.OutputFile)
	t.Logger.Field("Tempo de Processamento", "%s", result.ProcessingTime.Round(time.Second))
	if result.DetectedLanguage != "" {
		t.Logger.Field("Idioma Detectado", "%s", result.DetectedLanguage)
	}
	t.Logger.Field("Número de Palavras", "%d palavras", result.WordCount)
	t.Logger.Info("\nTranscrição concluída com sucesso!")
} 