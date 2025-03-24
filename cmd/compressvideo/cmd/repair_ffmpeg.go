package cmd

import (
	"fmt"

	"github.com/cccarv82/compressvideo/pkg/util"
	"github.com/spf13/cobra"
)

// repairFFmpegCmd representa o comando repair-ffmpeg
var repairFFmpegCmd = &cobra.Command{
	Use:   "repair-ffmpeg",
	Short: "Repair FFmpeg installation",
	Long: `Repair FFmpeg installation if it is missing or corrupted.

This command will check for an existing FFmpeg installation and,
if not found or corrupted, will download and install FFmpeg
automatically for your platform.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRepairFFmpeg()
	},
}

func init() {
	rootCmd.AddCommand(repairFFmpegCmd)
	
	// Flags
	repairFFmpegCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
}

// runRepairFFmpeg executa o processo de instalação do FFmpeg
func runRepairFFmpeg() error {
	// Configure logger
	logger = util.NewLogger(verbose)
	logger.Title("CompressVideo - FFmpeg Installation")

	// Verificar e instalar o FFmpeg
	logger.Section("Checking FFmpeg Installation")
	
	ffmpegInfo, err := util.RepairFFmpeg(logger)
	if err != nil {
		return fmt.Errorf("error installing FFmpeg: %w", err)
	}

	logger.Success("FFmpeg installation successful!")
	logger.Info("FFmpeg path: %s", ffmpegInfo.Path)
	logger.Info("FFprobe path: %s", ffmpegInfo.FFprobePath)
	logger.Info("FFmpeg version: %s", ffmpegInfo.Version)
	
	return nil
} 