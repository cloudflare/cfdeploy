package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type flags struct {
	env              string
	configFile       string
	configPath       string
	configDir        string
	marathonHost     string
	marathonCurlOpts string
	marathonForce    bool
	skipPrompt       bool
	verbose          bool
}

func (f *flags) parse() (err error) {

	// Parse flags
	flag.StringVar(&f.env, "e", "", "Environment (e.g. \"prod\")")
	flag.StringVar(&f.configFile, "f", "deploy.yaml", "Config File")
	flag.StringVar(&f.marathonHost, "marathon.host", "", "Marathon Host (e.g. \"www.example.com\"")
	flag.StringVar(&f.marathonCurlOpts, "marathon.curlopts", "", "Marathon cURL options (e.g. '-H \"OauthEmail: no-reply@cloudflare.com\"'). Note: only -H is currently supported.")
	flag.BoolVar(&f.marathonForce, "marathon.force", false, "Add the ?force=true to the Marathon request")
	flag.BoolVar(&f.skipPrompt, "y", false, "Skip confirmation prompt")
	flag.BoolVar(&f.verbose, "v", false, "Verbose mode e.g. dump Marathon config")
	flag.Parse()

	// Validate flags
	if f.env == "" || f.configFile == "" {
		flag.Usage()
		os.Exit(1)
	}
	f.configPath, err = filepath.Abs(f.configFile)
	if err != nil {
		return fmt.Errorf("Error parsing config file path: %s", err)
	}
	_, err = os.Stat(f.configPath)
	if err != nil {
		return fmt.Errorf("Invalid config file path '%s': %s", f.configFile, err)
	}
	f.configDir = filepath.Dir(f.configPath)
	if f.marathonHost != "" && strings.Contains(f.marathonHost, "/") {
		return fmt.Errorf(
			"Marathon hostname cannot contain forward slash. Found: %s",
			f.marathonHost,
		)
	}

	return

}
