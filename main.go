package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/radovskyb/watcher"
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
)

// Check if goku managed release already been deployed
func ReleaseExists(hc *helm.Client, name string) bool {
	response, err := hc.ListReleases(helm.ReleaseListFilter(name))
	if err != nil {
		panic("Could not list existing helm releases")
	}
	return response.Count == 1
}

// Create or update Helm release with chart & value overrides
func Deploy(chartName string, chartPath string, values map[string]interface{}) {
	vals, err := yaml.Marshal(values)
	if err != nil {
		panic("could not marshal value overrides")
	}

	//TODO find a cool way to autodetect kubectl context, and do this in the background?

	hc := helm.NewClient(helm.Host("127.0.0.1:44134"))
	fmt.Println("Loading chart...")
	achart, err := chartutil.Load(chartPath)

	if err != nil {
		log.Fatalln("Could not load chart", err)
	} else {
		releaseName := "goku-" + chartName
		if !ReleaseExists(hc, releaseName) {
			fmt.Printf("***Installing*** chart release %s... ", releaseName)
			_, err = hc.InstallReleaseFromChart(achart, "default", helm.ReleaseName(releaseName), helm.ValueOverrides(vals))
		} else {
			fmt.Printf("**Updating** existing chart release %s... ", releaseName)
			_, err = hc.UpdateReleaseFromChart(releaseName, achart, helm.UpdateValueOverrides(vals))
		}

		if err != nil {
			log.Fatalln("Failed to Install chart", err)
		} else {
			fmt.Println("Done\n")
		}
	}
}

// Build docker image inside local kubernetes node
func buildImage(contextPath string, imageName string) string {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	dockerFile := "Dockerfile"
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

		h, err := tar.FileInfoHeader(info, newPath)
		if err != nil {
			fmt.Println("Couldnt create tar header ")
		} else {
			h.Name = newPath
			err = tw.WriteHeader(h)
			if err != nil {
				fmt.Println("Error writing tar header")
			}
		}

		_, err = io.Copy(tw, aFile)
		if err != nil {
			fmt.Println("Error coping file contents to tar")
		}
		return nil
	}

	filepath.Walk(contextPath, walkDirFn)

	dockerFileTarReader := bytes.NewReader(buf.Bytes())
	ctx := context.Background()
	timeString := strconv.Itoa(int(time.Now().Unix()))
	tagName := imageName + ":" + timeString
	imageBuildResponse, err := cli.ImageBuild(
		ctx,
		dockerFileTarReader,
		types.ImageBuildOptions{
			Tags:       []string{tagName},
			Context:    dockerFileTarReader,
			Dockerfile: dockerFile,
			Remove:     true})

	if err != nil {
		log.Fatal(err, " :unable to build docker image")
	}
	defer imageBuildResponse.Body.Close()
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		log.Fatal(err, " :unable to read image build response")
	}
	return tagName
}

type WatchChangeFn func()

func startWatcher(contextPath string, watchCallback WatchChangeFn) {
	w := watcher.New()

	go func() {
		for {
			select {
			case event := <-w.Event:
				fmt.Println(event)
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
	fmt.Println("Watching files for changes:")
	for path, f := range w.WatchedFiles() {
		fmt.Printf("%s: %s\n", path, f.Name())
	}

	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

// goku.yaml structure
type GokuConfig struct {
	Charts []struct {
		Name   string `yaml:"name"`
		Path   string `yaml:"path"`
		Images []struct {
			ImageValueName string `yaml:"imageValueName"`
			Name           string `yaml:"name"`
			Path           string `yaml:"path"`
		} `yaml:"images"`
	} `yaml:"charts"`
}

// Configuration read from goku.yaml file
func ReadConfig() *GokuConfig {
	configData, err := ioutil.ReadFile("examples/goku.yaml")
	if err != nil {
		panic("Could not read goku.yaml")
	}
	gokuConfig := GokuConfig{}
	err = yaml.Unmarshal(configData, &gokuConfig)

	if err != nil {
		log.Fatalf("yaml error: %v", err)
	}

	return &gokuConfig
}

// watch for FS changes and build docker image, deploy to k8s using helm.
func main() {
	var wg sync.WaitGroup
	gokuConfig := ReadConfig()

	for _, chart := range gokuConfig.Charts {
		valueOverrides := make(map[string]interface{})
		for _, imageItem := range chart.Images {

			//initial bootstrap...
			imageTag := buildImage("examples/"+imageItem.Path, imageItem.Name)
			valueOverrides[imageItem.ImageValueName] = imageTag

			// Go thread to watch each image's file structure
			// build and update chart on any file change.
			go func(path string, name string, imageValueName string) {
				startWatcher("examples/"+path, func() {
					imageTag := buildImage("examples/"+path, name)
					valueOverrides[imageValueName] = imageTag
					Deploy(chart.Name, "examples/"+chart.Path, valueOverrides)
				})
			}(imageItem.Path, imageItem.Name, imageItem.ImageValueName)

			wg.Add(1)
		}

		Deploy(chart.Name, "examples/"+chart.Path, valueOverrides)
		wg.Wait()
	}
}
