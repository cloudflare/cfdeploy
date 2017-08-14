package main

import (
	"fmt"
	"io/ioutil"
	"log"
)

func main() {

	// Parse & validate flags
	flags := flags{}
	if err := flags.parse(); err != nil {
		log.Fatalf(err.Error())
	}

	// Read config file
	configData, err := ioutil.ReadFile(flags.configPath)
	if err != nil {
		log.Fatalf("Error reading config file '%s': %s\n", flags.configPath, err)
	}

	// Load config
	conf, err := configLoad(configData, flags)
	if err != nil {
		log.Fatalf("Error parsing config file: %s\n", err)
	}

	// Print config
	fmt.Printf("Environment: %s\n", flags.env)
	fmt.Printf("Config File: %s (%s)\n", flags.configFile, flags.configPath)

	// Get docker images and check they exist
	images, err := dockerImageList(conf, flags.env)
	if err != nil {
		log.Fatalf("Unable to verify docker images exists: %s\n", err)
	}
	vars := fileVars{Images: map[string]string{}}
	for key, image := range images {
		err := dockerCheckImage(image)
		if err != nil {
			log.Fatalf("Unable to verify docker images exists: %s\n", err)
		}
		vars.Images[key] = image.String()
	}

	// Print images
	fmt.Printf("Images:\n")
	for key, image := range vars.Images {
		fmt.Printf("* %s = %s\n", key, image)
	}

	// Check if deploy target is Marathon
	if conf.Marathon.Host != "" {

		// Prepare Marathon JSON config
		jsonConfig, err := marathonPrepare(flags, conf, vars)
		if err != nil {
			log.Fatalf("Error loading Marathon file: %s", err)
		}

		// Print info
		fmt.Printf(
			"Marathon File: %s\n",
			conf.Environments[flags.env].Marathon.File,
		)
		fmt.Printf("Marathon URL: %s\n", marathonURL(conf, flags.marathonForce))
		if len(conf.Marathon.Headers) > 0 {
			fmt.Printf("Marathon Headers:\n")
			for key, values := range conf.Marathon.Headers {
				for _, value := range values {
					if key == "Oauthaccesstoken" {
						value = "[hidden]"
					}
					fmt.Printf("* %s = %s\n", key, value)
				}
			}
		}
		if flags.verbose {
			fmt.Printf("Marathon Config: %s\n", jsonConfig)
		}

		// Confirm we should send request
		if !flags.skipPrompt && !promptConfirm("Deploy?") {
			log.Fatalf("Deployment cancelled")
		}

		// Deploy JSON config to Marathon
		result, err := marathonPush(
			conf,
			jsonConfig,
			flags.marathonForce,
		)
		if err != nil {
			log.Fatalf("Marathon deploy error:\n%s\n", err)
		}
		log.Printf("Deployed to marathon:\n%+v\n", result)

	} else {
		log.Fatalf("Deploy target unknown. Valid options: Marathon\n")
	}

}
