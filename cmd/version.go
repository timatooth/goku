package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Goku",
	Long:  `All software has versions. This is Gokus's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Goku v1.0.1")
	},
}
