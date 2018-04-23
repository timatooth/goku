package cmd

import (
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start minikube, install helm charts",
	Long:  `First time start of minikube and installs helm charts`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("start called")

		// get home dir
		usr, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		gokuBinPath := path.Join(usr.HomeDir, ".goku/bin")
		log.Println(gokuBinPath)

		//start minikube
		command := exec.Command(path.Join(gokuBinPath, "minikube"), "start", "--cpus", "4", "--memory", "8086")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err = command.Run()
		if err != nil {
			log.Fatalf("Run failed with %s\n", err)
		}

		// enable ingress
		command = exec.Command(path.Join(gokuBinPath, "minikube"), "addons", "enable", "ingress")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err = command.Run()
		if err != nil {
			log.Fatalf("Run failed with %s\n", err)
		}

		//enable heapster
		command = exec.Command(path.Join(gokuBinPath, "minikube"), "addons", "enable", "heapster")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err = command.Run()
		if err != nil {
			log.Fatalf("Run failed with %s\n", err)
		}

		//install tiller
		command = exec.Command(path.Join(gokuBinPath, "helm"), "init")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err = command.Run()
		if err != nil {
			log.Fatalf("Run failed with %s\n", err)
		}

	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
