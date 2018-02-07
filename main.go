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
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/radovskyb/watcher"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
)

func createDeployment(chart_path string) {
	hc := helm.NewClient()
	fmt.Println(hc.ListReleases())
	fmt.Println("Loading chart...")
	chart, err := chartutil.Load(chart_path)
	if err != nil {
		log.Fatalln("Could not load chart", err)
	} else {
		fmt.Println(chart)
		fmt.Println("Installing chart")
		response, err := hc.InstallReleaseFromChart(chart, "default")
		if err != nil {
			log.Fatalln("failed to install chart", err)
		} else {
			fmt.Println(response)
		}
	}
}

func buildSampleImage(context_path string) {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("OMG CONTAINER: %s %s\n", container.ID[:10], container.Image)
	}

	//create a go ctx to watch for build progress

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	dockerFile := "Dockerfile"
	root_directory := context_path

	walkDirFn := func(path string, info os.FileInfo, err error) error {

		if info.IsDir() {
			return nil
		}

		new_path := path[len(root_directory)+1:]
		if len(new_path) == 0 {
			return nil
		}
		fmt.Println(new_path)

		//read all files, add em' to the tar
		aFile, err := os.Open(path)
		if err != nil {
			log.Fatal(err, " :unable to open "+path)
		}
		defer aFile.Close()

		h, err := tar.FileInfoHeader(info, new_path)
		if err != nil {
			fmt.Println("Couldnt create tar header ")
		} else {
			h.Name = new_path
			err = tw.WriteHeader(h)
			if err != nil {
				fmt.Println("Error writing tar header")
			}
		}

		length, err := io.Copy(tw, aFile)
		if err != nil {
			fmt.Println("Error coping file contents to tar")
		} else {
			fmt.Printf("Wrote tar contents of %s %d bytes\n", new_path, length)
		}
		return nil
	}

	filepath.Walk("samples", walkDirFn)

	dockerFileTarReader := bytes.NewReader(buf.Bytes())
	ctx := context.Background()
	imageBuildResponse, err := cli.ImageBuild(
		ctx,
		dockerFileTarReader,
		types.ImageBuildOptions{
			Tags:       []string{"albi/yolo"},
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
}

func main() {
	createDeployment("testchart")
	context_path := "samples"
	w := watcher.New()

	go func() {
		for {
			select {
			case event := <-w.Event:
				fmt.Println(event) // Print the event's info.
				buildSampleImage(context_path)
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	if err := w.AddRecursive(context_path); err != nil {
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
