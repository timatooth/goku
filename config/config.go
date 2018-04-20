package config

import (
	"log"
	"path"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

// goku.yaml structure
type GokuConfig struct {
	// Helm charts
	Charts []struct {
		// Vanity name of the chart
		Name string `yaml:"name"`
		// Location of the chart relative to the goku.yaml file BaseDir
		Path string `yaml:"path"`
		// Map image, name, helm template value names for overriding
		Images []struct {
			// The value name which must exist in the helm chart templates
			ImageValueName string `yaml:"imageValueName"`
			Name           string `yaml:"name"`
			// Path for Goku to watch for changes. Used as the default docker ContextPath
			Path string `yaml:"path"`
			// Optional extra tags to apply to the image
			Tags []string `yaml:"tags"`
			// Optionl set a different Docker build context Path from the watch Path.
			ContextPath string `yaml:"contextPath"`
			// Optional custom path to Dockerfile. Must be below the ContextPath
			Dockerfile string `yaml:"dockerfile"`
		} `yaml:"images"`
	} `yaml:"charts"`
	Tools map[string] map[string] string `yaml:"tools"`
	// The base path relative to goku.yaml where all paths are built from
	BaseDir string
}

// Configuration read from goku.yaml file
func ReadConfig(configPath string) *GokuConfig {
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic("Could not read: " + configPath)
	}
	gokuConfig := GokuConfig{}
	err = yaml.Unmarshal(configData, &gokuConfig)

	if err != nil {
		log.Fatalf("yaml error: %v", err)
	}

	//set the BaseDir so every path is relative to the Gokufile
	gokuConfig.BaseDir = path.Dir(configPath)

	return &gokuConfig
}
