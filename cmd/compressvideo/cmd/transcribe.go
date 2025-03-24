package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cccarv82/compressvideo/pkg/transcriber"
	"github.com/cccarv82/compressvideo/pkg/util"
	"github.com/spf13/cobra"
)

var (
	// Opções específicas de transcrição
	language      string // Idioma do vídeo (auto = detectar automaticamente)
	modelSize     string // Tamanho do modelo (tiny, base, small, medium, large)
	showTimestamp bool   // Mostrar timestamps para cada segmento
)

// transcribeCmd representa o comando transcribe
var transcribeCmd = &cobra.Command{
	Use:   "transcribe",
	Short: "Transcreve o áudio de um vídeo para texto",
	Long: `Transcreve o áudio de um vídeo para texto usando reconhecimento de fala.

Este comando extrai o áudio do vídeo e usa o modelo Whisper da OpenAI para
transcrevê-lo para texto. A transcrição é salva em um arquivo de texto.

Exemplos:
  compressvideo transcribe -i video.mp4
  compressvideo transcribe -i video.mp4 -o transcript.txt
  compressvideo transcribe -i video.mp4 -l pt -m small`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTranscription()
	},
}

func init() {
	rootCmd.AddCommand(transcribeCmd)

	// Adiciona flags específicas para transcrição
	transcribeCmd.Flags().StringVarP(&language, "language", "l", "auto", "Idioma do vídeo (auto = detectar automaticamente)")
	transcribeCmd.Flags().StringVarP(&modelSize, "model", "m", "base", "Tamanho do modelo (tiny, base, small, medium, large)")
	transcribeCmd.Flags().BoolVarP(&showTimestamp, "timestamps", "s", false, "Mostrar timestamps para cada segmento")
	
	// Re-define algumas flags do comando raiz
	transcribeCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input video file (required)")
	transcribeCmd.MarkFlagRequired("input")
	transcribeCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output text file (default: input.txt)")
	transcribeCmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite output file if it exists")
	transcribeCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
}

// runTranscription executa o processo de transcrição
func runTranscription() error {
	// Configure logger
	logger = util.NewLogger(verbose)
	logger.Title("CompressVideo - Transcrição de Vídeo")

	// Validar flags
	if err := validateTranscriptionFlags(); err != nil {
		return err
	}
	
	// Se o arquivo de saída não foi especificado, usar o mesmo nome do arquivo de entrada com extensão .txt
	if outputFile == "" {
		dir := filepath.Dir(inputFile)
		base := filepath.Base(inputFile)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		outputFile = filepath.Join(dir, base+".txt")
	}

	// Verificar se o arquivo de entrada é um vídeo
	if !isVideoFile(inputFile) {
		return fmt.Errorf("o arquivo de entrada não parece ser um vídeo válido: %s", inputFile)
	}
	
	// Verificar se o arquivo de saída já existe (a menos que force esteja definido)
	if _, err := os.Stat(outputFile); err == nil && !force {
		return fmt.Errorf("o arquivo de saída já existe (use -f para sobrescrever): %s", outputFile)
	}
	
	logger.Section("Transcrição de Vídeo")
	logger.Field("Arquivo de Entrada", "%s", inputFile)
	logger.Field("Arquivo de Saída", "%s", outputFile)
	logger.Field("Idioma", "%s", language)
	logger.Field("Modelo", "%s", modelSize)
	
	// Inicializar o transcriber
	transcribeInst, err := transcriber.NewTranscriber(logger)
	if err != nil {
		// Se for um erro de "Whisper não encontrado", sugerir o comando repair-whisper
		if strings.Contains(err.Error(), "não foi possível encontrar o Whisper") {
			logger.Error("Whisper não encontrado.")
			logger.Info("Execute o comando abaixo para instalar o Whisper automaticamente:")
			logger.Info("  compressvideo repair-whisper")
			return err
		}
		return fmt.Errorf("falha ao inicializar transcriber: %w", err)
	}
	
	// Configurar opções de transcrição
	options := transcriber.DefaultOptions()
	options.Language = language
	options.Model = modelSize
	options.ShowTimestamps = showTimestamp
	
	// Criar um tracker de progresso
	progressOptions := util.ProgressTrackerOptions{
		Total:          100,
		Description:    "Transcrevendo vídeo",
		Logger:         logger,
		ShowPercentage: true,
		ShowSpeed:      true,
	}
	progressBar := util.NewProgressTrackerWithOptions(progressOptions)
	
	// Iniciar a transcrição
	logger.Info("Iniciando transcrição do vídeo...")
	startTime := time.Now()
	
	result, err := transcribeInst.Transcribe(inputFile, outputFile, options, progressBar)
	if err != nil {
		progressBar.Finish()
		return fmt.Errorf("erro na transcrição: %w", err)
	}
	
	// Garantir que a barra de progresso termine
	progressBar.Finish()
	
	// Exibir relatório
	transcribeInst.DisplayTranscriptionResult(result)
	
	// Exibir uma mensagem amigável de conclusão
	processingTime := time.Since(startTime).Round(time.Second)
	
	logger.Success("Transcrição concluída com sucesso!")
	logger.Info("Arquivo %s transcrito em %s, gerando %d palavras", 
		filepath.Base(inputFile), 
		processingTime,
		result.WordCount)
	
	return nil
}

// validateTranscriptionFlags valida as flags para o comando de transcrição
func validateTranscriptionFlags() error {
	// Validar se o arquivo de entrada existe
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("arquivo de entrada não existe: %s", inputFile)
	}
	
	// Validar tamanho do modelo
	validModels := map[string]bool{
		"tiny":   true,
		"base":   true,
		"small":  true,
		"medium": true,
		"large":  true,
	}
	
	if !validModels[modelSize] {
		return fmt.Errorf("tamanho de modelo inválido: %s. Use um dos seguintes: tiny, base, small, medium, large", modelSize)
	}
	
	return nil
} 