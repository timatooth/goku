package cmd

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/fatih/color"
	"github.com/radovskyb/watcher"
	"github.com/spf13/cobra"
	GokuConfig "github.com/timatooth/goku/config"
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
)

// Check if goku managed release already been deployed
func ReleaseExists(hc *helm.Client, name string) bool {
	response, err := hc.ListReleases(helm.ReleaseListFilter(name))
	if err != nil {
		log.Fatalf("%s Could not list existing helm releases. Have you port forwarded the Tiller Pod?", err)
		panic("Can't contact Tiller")
	}
	return response.Count == 1
}

// Deploy - Create or update Helm release with chart & value overrides
func Deploy(chartName string, chartPath string, values map[string]interface{}) {
	vals, err := yaml.Marshal(values)
	if err != nil {
		panic("Could not marshal Chart value overrides")
	}

	//TODO find a cool way to autodetect kubectl context, and do this in the background?

	hc := helm.NewClient(helm.Host("127.0.0.1:44134"), helm.ConnectTimeout(5))
	log.Printf("Loading chart %s ...\n", chartPath)
	achart, err := chartutil.Load(chartPath)

	if err != nil {
		log.Fatalln("Could not load Helm chart", err)
	} else {
		releaseName := "goku-" + chartName
		if !ReleaseExists(hc, releaseName) {
			log.Printf("***Installing*** chart release %s... ", releaseName)
			_, err = hc.InstallReleaseFromChart(achart, "default", helm.ReleaseName(releaseName), helm.ValueOverrides(vals))
		} else {
			log.Printf("**Updating** existing chart release %s... ", releaseName)
			_, err = hc.UpdateReleaseFromChart(releaseName, achart, helm.UpdateValueOverrides(vals))
		}

		if err != nil {
			log.Fatalln("Failed to install/update Helm chart", err)
		} else {
			log.Println("Done")
		}
	}
}

// Build docker image inside local kubernetes node
func buildImage(contextPath string, imageName string, dockerFile string, tags []string) string {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	rootDirectory := contextPath

	walkDirFn := func(path string, info os.FileInfo, err error) error {

		if info.IsDir() {
			return nil
		}

		newPath := path[len(rootDirectory)+1:]
		if len(newPath) == 0 {
			return nil
		}

		aFile, err := os.Open(path)
		if err != nil {
			log.Fatal(err, " :unable to open "+path)
		}
		defer aFile.Close()

		h, err := tar.FileInfoHeader(info, filepath.ToSlash(newPath))
		if err != nil {
			log.Println("Couldn't create tar header ")
		} else {
			// We need to convert ToSlash if the OS is Windows
			// make sure the path slashes are around the right way!
			h.Name = filepath.ToSlash(newPath)
			err = tw.WriteHeader(h)
			if err != nil {
				log.Println("Error writing tar header")
			}
		}

		_, err = io.Copy(tw, aFile)
		if err != nil {
			log.Println("Error coping file contents to tar")
		}
		return nil
	}

	filepath.Walk(contextPath, walkDirFn)

	dockerFileTarReader := bytes.NewReader(buf.Bytes())
	ctx := context.Background()
	timeString := strconv.Itoa(int(time.Now().Unix()))
	tagName := imageName + ":" + timeString
	allTags := append(tags, tagName)
	imageBuildResponse, err := cli.ImageBuild(
		ctx,
		dockerFileTarReader,
		types.ImageBuildOptions{
			Tags:       allTags,
			Context:    dockerFileTarReader,
			Dockerfile: dockerFile,
			Remove:     true})

	if err != nil {
		log.Fatal(err, " :Unable to build docker image")
	}
	defer imageBuildResponse.Body.Close()
	defer color.Unset()
	color.Set(color.FgCyan)
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		log.Fatal(err, " :Unable to read image build response")
	}
	return tagName
}

//callback type called on file change event
type WatchChangeFn func()

func startWatcher(contextPath string, watchCallback WatchChangeFn) {
	w := watcher.New()

	go func() {
		for {
			select {
			case event := <-w.Event:
				log.Println(event)
				watchCallback()
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	if err := w.AddRecursive(contextPath); err != nil {
		log.Fatalln(err)
	}
	log.Println("Watching files for changes:")
	for path, f := range w.WatchedFiles() {
		log.Printf("%s: %s\n", path, f.Name())
	}
	//check source files every 100ms
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

func StartWatching(config *GokuConfig.GokuConfig) {
	var wg sync.WaitGroup

	for _, chart := range config.Charts {
		valueOverrides := make(map[string]interface{})
		for _, imageItem := range chart.Images {
			//if ContextPath is not given: use the watchPath (Path) instead
			dockerBuildContext := imageItem.Path
			if imageItem.ContextPath != "" {
				dockerBuildContext = imageItem.ContextPath
			}

			// if Dockerfile is not give assume its ContextPath/Dockerfile
			if imageItem.Dockerfile == "" {
				imageItem.Dockerfile = "Dockerfile"
			}

			//TODO this is an initial build/bootstrap on startup... to be removed?
			imageTag := buildImage(path.Join(config.BaseDir, dockerBuildContext), imageItem.Name, imageItem.Dockerfile, imageItem.Tags)
			valueOverrides[imageItem.ImageValueName] = imageTag

			// Go thread to watch each image's file structure
			// build and update chart on any file change.
			go func(watchPath string, name string, dockerFile, contextPath string, tags []string, imageValueName string) {
				startWatcher(path.Join(config.BaseDir, watchPath), func() {

					imageTag := buildImage(path.Join(config.BaseDir, contextPath), name, dockerFile, tags)
					valueOverrides[imageValueName] = imageTag
					Deploy(chart.Name, path.Join(config.BaseDir, chart.Path), valueOverrides)
				})
			}(imageItem.Path, imageItem.Name, imageItem.Dockerfile, dockerBuildContext, imageItem.Tags, imageItem.ImageValueName)

			wg.Add(1)
		}

		Deploy(chart.Name, path.Join(config.BaseDir, chart.Path), valueOverrides)
	}
	//block until all threads end
	wg.Wait()
}

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch goku managed containers for changes and redeploy to Kubernetes via Helm",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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

		//setup minikube docker-env

		//setup background connection to Tiller pod

		StartWatching(gokuConfig)
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// watchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// watchCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
