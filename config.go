package main

import (
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"net/http"
	"strings"
)

type config struct {
	Marathon     configMarathon               `yaml:"marathon"`
	Image        configImage                  `yaml:"image"`
	Environments map[string]configEnvironment `yaml:"environments"`
}

type configMarathon struct {
	Host    string `yaml:"host"`
	Headers http.Header
}

type configImage struct {
	Repository  string `yaml:"repository"`
	Name        string `yaml:"name"`
	TagTemplate string `yaml:"tagTemplate"`
}

type configEnvironment struct {
	Marathon struct {
		File string `yaml:"file"`
	} `yaml:"marathon"`
	Images map[string]configImage
}

func configLoad(fileData []byte, flags flags) (config, error) {

	// Parse file YAML
	var c config
	err := yaml.Unmarshal(fileData, &c)
	if err != nil {
		return config{}, err
	}

	// Check environment exists
	if _, ok := c.Environments[flags.env]; !ok {
		return config{}, fmt.Errorf("Environment %s not found in config")
	}

	// Override marathon host if provided
	if flags.marathonHost != "" {
		c.Marathon.Host = flags.marathonHost
	}

	// Parse marathon headers if provided
	if flags.marathonCurlOpts != "" {
		c.Marathon.Headers = http.Header{}
		curlOpts := strings.Split(flags.marathonCurlOpts, "-H")
		for _, curlOpt := range curlOpts {
			curlOpt = strings.TrimSpace(strings.Replace(curlOpt, "\"", "", 2))
			if curlOpt == "" {
				continue
			}
			curlOptSplit := strings.Split(curlOpt, ": ")
			if len(curlOptSplit) == 2 {
				c.Marathon.Headers.Add(curlOptSplit[0], curlOptSplit[1])
			}
		}
	}

	// Return config struct
	return c, nil

}
