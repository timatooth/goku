package cmd

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"

	"github.com/mholt/archiver"
	"github.com/spf13/cobra"
	GokuConfig "github.com/timatooth/goku/config"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initilise goku config & download all toos",
	Long: `Downloads all necessary Go tools for local Kubernetes development such as:
	* kubectl
	* minikube
	* helm

	Installs them to your $HOME/.goku/bin.

	Your should add ~/.goku/bin to your $PATH
	`,
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

		//log.Println(gokuConfig)

		// get home dir
		usr, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		log.Println(usr.HomeDir)

		//create $HOME/.goku/bin if it does not exist
		gokuBinPath := path.Join(usr.HomeDir, ".goku/bin")
		if _, err := os.Stat(gokuBinPath); os.IsNotExist(err) {
			log.Println("creating", gokuBinPath)
			os.MkdirAll(gokuBinPath, os.ModePerm)
		} else {
			log.Println(gokuBinPath, "exists")
		}

		// get os type
		platform := runtime.GOOS
		log.Println("You are running on", platform)
		//download binaries
		for key, url := range gokuConfig.Tools {
			location := path.Join(gokuBinPath, key)
			log.Printf("Downloading tool: %s from %s to %s", key, url[platform], location)

			err := DownloadFile(location, url[platform])
			if err != nil {
				panic(err)
			}

			//edge case, if .tar.gz extract contents
			if filepath.Ext(url[platform]) == ".gz" {
				gokuExtractingStagePath := path.Join(usr.HomeDir, ".goku/bin/stage")
				log.Println("Extracting tar.gz")

				//TarGz
				log.Printf("Extracting %s to %s\n", location, gokuExtractingStagePath)
				err = archiver.TarGz.Open(location, gokuExtractingStagePath)
				if err != nil {
					log.Fatalf("Error extracting targz file %s \n", err)
				}

				// a naieve approach, flatten the directory structure of the staged extration to ~/.goku/bin
				//err := os.Rename(originalPath, newPath)
				filepath.Walk(gokuExtractingStagePath, visit)
			}

			// chmod
			log.Printf("Making %s executable\n", location)
			err = os.Chmod(location, 0700)
		}

		// print instructions to user on $PATH setup
		log.Printf("Now add %s to your OS $PATH environment variable", gokuBinPath)
	},
}

func visit(inputPath string, f os.FileInfo, err error) error {
	// get home dir
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	gokuBinPath := path.Join(usr.HomeDir, ".goku/bin")
	//flatten
	err = os.Rename(inputPath, path.Join(gokuBinPath, filepath.Base(inputPath)))
	log.Printf("Visited: %s\n", inputPath)
	return nil
}

func DownloadFile(filepath string, url string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
