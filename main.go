package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/radovskyb/watcher"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
)

func createDeployment(chartPath string, imageTag string) {
	// for testing: kubectl -n kube-system port-forward tiller-deploy-7777bff5d-7j5x4 44134
	//TODO find a cool way to autodetect kubectl context, and do this in the background?

	hc := helm.NewClient(helm.Host("127.0.0.1:44134"))
	fmt.Println("Loading chart...")
	achart, err := chartutil.Load(chartPath)

	if err != nil {
		log.Fatalln("Could not load chart", err)
	} else {
		fmt.Println("installing chart")
		//response, err := hc.InstallReleaseFromChart(achart, "default", helm.ValueOverrides([]byte("image: "+imageTag)))
		response, err := hc.UpdateReleaseFromChart("orderly-stoat", achart, helm.UpdateValueOverrides([]byte("image: "+imageTag)))
		if err != nil {
			log.Fatalln("Failed to Install chart", err)
		} else {
			fmt.Println(response)
		}
	}
}

func buildSampleImage(contextPath string) {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	//create a go ctx to watch for build progress
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
		fmt.Println(newPath)

		//read all files, add em' to the tar
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

		length, err := io.Copy(tw, aFile)
		if err != nil {
			fmt.Println("Error coping file contents to tar")
		} else {
			fmt.Printf("Wrote tar contents of %s %d bytes\n", newPath, length)
		}
		return nil
	}

	filepath.Walk("samples", walkDirFn)

	dockerFileTarReader := bytes.NewReader(buf.Bytes())
	ctx := context.Background()
	timeString := strconv.Itoa(int(time.Now().Unix()))
	tagName := "goku:" + timeString
	fmt.Println("Tag is " + tagName)
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
	createDeployment("testchart", tagName)
}

// watch for FS changes and build docker image, deploy to k8s using helm.
func main() {
	contextPath := "samples"
	w := watcher.New()

	go func() {
		for {
			select {
			case event := <-w.Event:
				fmt.Println(event)
				buildSampleImage(contextPath)
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
