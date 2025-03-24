package cmd

import (
	"fmt"
	"runtime"

	"github.com/cccarv82/compressvideo/pkg/transcriber"
	"github.com/cccarv82/compressvideo/pkg/util"
	"github.com/spf13/cobra"
)

var (
	// Flag para forçar a instalação da versão ctranslate2 do Whisper (mais leve)
	forceCtranslate2 bool
)

// repairWhisperCmd representa o comando repair-whisper
var repairWhisperCmd = &cobra.Command{
	Use:   "repair-whisper",
	Short: "Instala ou atualiza o Whisper para transcrição de áudio",
	Long: `Instala ou atualiza o Whisper para transcrição de áudio.

Este comando verifica a instalação do Whisper e o instala automaticamente se não estiver presente.
Whisper é o motor de reconhecimento de fala usado para transcrever o áudio dos vídeos.

No Windows, é recomendado usar a opção --ctranslate2 para instalar uma versão mais leve
que não requer dependências de compilação.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRepairWhisper()
	},
}

func init() {
	rootCmd.AddCommand(repairWhisperCmd)
	
	// Flags
	repairWhisperCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Mostrar saída detalhada durante a instalação")
	
	// No Windows, adicionar opção para instalar ctranslate2 (mais fácil)
	if runtime.GOOS == "windows" {
		repairWhisperCmd.Flags().BoolVar(&forceCtranslate2, "ctranslate2", true, "Instalar versão mais leve do Whisper (recomendado para Windows)")
	} else {
		repairWhisperCmd.Flags().BoolVar(&forceCtranslate2, "ctranslate2", false, "Instalar versão mais leve do Whisper")
	}
}

// runRepairWhisper executa o processo de instalação do Whisper
func runRepairWhisper() error {
	// Configure logger
	logger = util.NewLogger(verbose)
	logger.Title("CompressVideo - Instalação do Whisper")

	// Criar o instalador
	installer := transcriber.NewWhisperInstaller(logger)

	// Verificar e instalar o Whisper
	logger.Section("Verificando instalação do Whisper")
	isInstalled, err := installer.CheckWhisperInstallation()
	if err != nil {
		return fmt.Errorf("erro ao verificar instalação do Whisper: %w", err)
	}

	if isInstalled {
		logger.Success("Whisper já está instalado!")
		return nil
	}

	logger.Info("Whisper não encontrado. Iniciando instalação...")
	
	// Criar uma barra de progresso
	progressOptions := util.ProgressTrackerOptions{
		Total:          100,
		Description:    "Instalando Whisper",
		Logger:         logger,
		ShowPercentage: true,
		ShowSpeed:      true,
	}
	progressBar := util.NewProgressTrackerWithOptions(progressOptions)

	// Instalar o Whisper
	var installErr error
	if forceCtranslate2 {
		// Se a flag forceCtranslate2 estiver ativa, tentar instalar a versão ctranslate2
		logger.Info("Tentando instalar versão mais leve (ctranslate2)...")
		
		// No Windows, mostrar instruções específicas
		if runtime.GOOS == "windows" {
			logger.Info("Para instalação manual no Windows, execute:")
			logger.Info("pip install -U whisper-ctranslate2")
		}
		
		installErr = installer.InstallWhisperCtranslate2(progressBar)
	} else {
		// Caso contrário, usar a instalação padrão
		installErr = installer.InstallWhisper(progressBar)
	}
	
	progressBar.Finish()
	
	if installErr != nil {
		return fmt.Errorf("falha ao instalar o Whisper: %w", installErr)
	}

	logger.Success("Whisper instalado com sucesso!")
	return nil
} 