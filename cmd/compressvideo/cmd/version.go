package cmd

import (
	"fmt"

	"github.com/cccarv82/compressvideo/pkg/util"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version information",
	Long:  `Display the version information of CompressVideo.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(util.GetVersionInfo())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
} 