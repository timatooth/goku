package cmd

import (
	"log"
	GokuConfig "github.com/timatooth/goku/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Checks config",
	Long: `Parses goku.yaml config file`,
	Run: func(cmd *cobra.Command, args []string) {
		var gokuConfig *GokuConfig.GokuConfig
		if len(args) < 1 {
			// look for goku.yaml in current dir
			log.Println("Looking for goku.yaml file in current directory")
			gokuConfig = GokuConfig.ReadConfig("goku.yaml")
		} else {
			log.Printf("Looking for goku.yaml file in %s\n", args[0])
			gokuConfig = GokuConfig.ReadConfig(args[0])
		}
		log.Print(gokuConfig)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
