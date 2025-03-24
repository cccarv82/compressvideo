package cmd

import (
	"fmt"

	"github.com/cccarv82/compressvideo/pkg/transcriber"
	"github.com/cccarv82/compressvideo/pkg/util"
	"github.com/spf13/cobra"
)

// repairWhisperCmd representa o comando repair-whisper
var repairWhisperCmd = &cobra.Command{
	Use:   "repair-whisper",
	Short: "Instala ou atualiza o Whisper para transcrição de áudio",
	Long: `Instala ou atualiza o Whisper para transcrição de áudio.

Este comando verifica a instalação do Whisper e o instala automaticamente se não estiver presente.
Whisper é o motor de reconhecimento de fala usado para transcrever o áudio dos vídeos.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRepairWhisper()
	},
}

func init() {
	rootCmd.AddCommand(repairWhisperCmd)
	
	// Flags
	repairWhisperCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Mostrar saída detalhada durante a instalação")
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
	err = installer.InstallWhisper(progressBar)
	progressBar.Finish()
	
	if err != nil {
		return fmt.Errorf("falha ao instalar o Whisper: %w", err)
	}

	logger.Success("Whisper instalado com sucesso!")
	return nil
} 