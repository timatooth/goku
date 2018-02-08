package main

import (
	"fmt"
	"log"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
)

func createDeployment(chartPath string, imageID string) {
	hc := helm.NewClient(helm.Host("127.0.0.1:44134"))
	myTestChart, err := chartutil.Load(chartPath)

	if err != nil {
		log.Fatalln("Could not load chart", err)
	} else {
		fmt.Println("Installing chart")

		response, err := hc.InstallReleaseFromChart(myTestChart, "default")
		if err != nil {
			log.Fatalln("Failed to Install chart", err)
		} else {
			fmt.Println(response)
		}
	}
}

func main() {
	createDeployment("/Users/sullivt/dev/src/albi_cloud/k8s/albi-api/", "hello-world")
}
