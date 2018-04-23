package cmd

import (
	"log"
	"path"
	//"io"
	"os"
	"os/user"
	"os/exec"
	"github.com/spf13/cobra"
)

// helmCmd represents the helm command
var helmCmd = &cobra.Command{
	Use:   "helm",
	Short: "Setup helm",
	Long: `Installs tiller & initial charts`,
	Run: func(cmd *cobra.Command, args []string) {
		// get home dir
		usr, err := user.Current()
    if err != nil {
        log.Fatal( err )
    }
		gokuBinPath := path.Join(usr.HomeDir, ".goku/bin")
    log.Println( gokuBinPath)

		log.Println("Installing Tiller (Helm Server) into Minikube")
		// install tiller
		command := exec.Command(path.Join(gokuBinPath, "helm"), "init")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err = command.Run()
		if err != nil {
			log.Fatalf("Run failed with %s\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(helmCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// helmCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// helmCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
