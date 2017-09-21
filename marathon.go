package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type marathonGroup struct {
	ID     string          `json:"id" yaml:"id"`
	Apps   []marathonApp   `json:"apps,omitempty" yaml:"apps"`
	Groups []marathonGroup `json:"groups,omitempty" yaml:"groups"`
}

type portDefinition struct {
	Port int64  `json:"port" yaml:"port"`
	Name string `json:"name" yaml:"name"`
}

type marathonApp struct {
	ID          string     `json:"id" yaml:"id"`
	Cmd         string     `json:"cmd,omitempty" yaml:"cmd"`
	Args        []string   `json:"args,omitempty" yaml:"args"`
	Instances   int64      `json:"instances" yaml:"instances"`
	CPUs        float64    `json:"cpus" yaml:"cpus"`
	Mem         int64      `json:"mem" yaml:"mem"`
	Disk        int64      `json:"disk,omitempty" yaml:"disk"`
	Constraints [][]string `json:"constraints" yaml:"constraints"`
	Fetch       []struct {
		URI        string `json:"uri,omitempty" yaml:"uri"`
		Executable bool   `json:"executable,omitempty" yaml:"executable"`
		Extract    bool   `json:"extract,omitempty" yaml:"extract"`
		Cache      bool   `json:"cache,omitempty" yaml:"cache"`
	} `json:"fetch,omitempty" yaml:"fetch"`
	URIs                       []string         `json:"uris,omitempty" yaml:"uris"`
	StoreURLs                  []string         `json:"storeUrls,omitempty" yaml:"storeUrls"`
	Ports                      []int64          `json:"ports,omitempty" yaml:"ports"`
	PortDefinitions            []portDefinition `json:"portDefinitions,omitempty" yaml:"portDefinitions"`
	RequirePorts               bool             `json:"requirePorts" yaml:"requirePorts"`
	BackoffSeconds             int64            `json:"backoffSeconds,omitempty" yaml:"backoffSeconds"`
	BackoffFactor              float64          `json:"backoffFactor,omitempty" yaml:"backoffFactor"`
	MaxLaunchDelaySeconds      int64            `json:"maxLaunchDelaySeconds,omitempty" yaml:"maxLaunchDelaySeconds"`
	TaskKillGracePeriodSeconds int64            `json:"taskKillGracePeriodSeconds,omitempty" yaml:"taskKillGracePeriodSeconds"`
	Container                  struct {
		Type    string `json:"type" yaml:"type"`
		Volumes []struct {
			ContainerPath string `json:"containerPath" yaml:"containerPath"`
			HostPath      string `json:"hostPath" yaml:"hostPath"`
			Mode          string `json:"mode"`
		} `json:"volumes" yaml:"volumes"`
		Docker *struct {
			Image      string `json:"image" yaml:"image"`
			Network    string `json:"network" yaml:"network"`
			Privileged bool   `json:"privileged" yaml:"privileged"`
			Parameters []struct {
				Key   string `json:"key" yaml:"key"`
				Value string `json:"value" yaml:"value"`
			} `json:"parameters" yaml:"parameters"`
			ForcePullImage bool `json:"forcePullImage" yaml:"forcePullImage"`
		} `json:"docker,omitempty" yaml:"docker"`
	} `json:"container" yaml:"container"`
	Env          map[string]string `json:"env,omitempty" yaml:"env"`
	Labels       map[string]string `json:"labels,omitempty" yaml:"labels"`
	Dependencies []string          `json:"dependencies,omitempty" yaml:"dependencies"`
	HealthChecks []struct {
		Protocol               string `json:"protocol,omitempty" yaml:"protocol"`
		Path                   string `json:"path,omitempty" yaml:"path"`
		GracePeriodSeconds     int64  `json:"gracePeriodSeconds,omitempty" yaml:"gracePeriodSeconds"`
		IntervalSeconds        int64  `json:"intervalSeconds,omitempty" yaml:"intervalSeconds"`
		PortIndex              int64  `json:"portIndex,omitempty" yaml:"portIndex"`
		TimeoutSeconds         int64  `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds"`
		MaxConsecutiveFailures int64  `json:"maxConsecutiveFailures,omitempty" yaml:"maxConsecutiveFailures"`
	} `json:"healthChecks,omitempty" yaml:"healthChecks"`
	UpgradeStrategy *struct {
		MinimumHealthCapacity *float64 `json:"minimumHealthCapacity,omitempty" yaml:"minimumHealthCapacity"`
		MaximumOverCapacity   *float64 `json:"maximumOverCapacity,omitempty" yaml:"maximumOverCapacity"`
	} `json:"upgradeStrategy,omitempty" yaml:"upgradeStrategy"`
	IPAddress *struct {
		Groups      []string          `json:"groups,omitempty" yaml:"groups"`
		Labels      map[string]string `json:"labels,omitempty" yaml:"labels"`
		NetworkName string            `json:"networkName,omitempty" yaml:"networkName"`
	} `json:"ipAddress,omitempty" yaml:"ipAddress"`
}

type marathonResult struct {
	Message string `json:"message"`
	Details []struct {
		Path   string   `json:"path"`
		Errors []string `json:"errors"`
	} `json:"details"`
	Version      string `json:"version"`
	DeploymentID string `json:"deploymentId"`
}

// marathonPrepare will read a YAML file, validate it and return a JSON
func marathonPrepare(f flags, conf config, vars fileVars) ([]byte, error) {

	// Build file path
	filePath := f.configDir + "/" + conf.Environments[f.env].Marathon.File

	// Read file into a template and parse
	fileData, err := fileLoad(filePath, vars)
	if err != nil {
		return nil, fmt.Errorf(
			"Unable to load '%s' Marathon file:\n%s",
			conf.Environments[f.env].Marathon.File,
			err,
		)
	}

	// Unmarshal YAML and validate
	group, err := marathonParseYAML(fileData)
	if err != nil {
		return nil, err
	}
	err = marathonValidate(group)
	if err != nil {
		return nil, err
	}

	// Marshal JSON
	marathonJSON, err := json.MarshalIndent(group, "", "    ")
	if err != nil {
		return nil, fmt.Errorf(
			"Error marshaling JSON: %s",
			err,
		)
	}

	return marathonJSON, nil

}

func marathonParseYAML(fileData []byte) (marathonGroup, error) {

	// Parse file YAML
	var group marathonGroup
	err := yaml.Unmarshal(fileData, &group)
	if err != nil {
		return marathonGroup{}, err
	}

	// Return group struct
	return group, nil

}

func marathonValidate(group marathonGroup) error {
	if group.ID == "" {
		return fmt.Errorf("App id '%s' invalid", group.ID)
	}
	for i, app := range group.Apps {
		if app.ID == "" {
			return fmt.Errorf(
				"App %d id invalid. Found: %s",
				i,
				app.ID,
			)
		}
		if app.Container.Type != "DOCKER" {
			return fmt.Errorf(
				"App %d container type must be docker. Found: %s",
				i,
				app.Container.Type,
			)
		}
		if app.Container.Docker.Image == "" {
			return fmt.Errorf(
				"App %d container docker image must not be empty",
				i,
			)
		}
		if !strings.Contains(app.Container.Docker.Image, ":") {
			return fmt.Errorf(
				"App %d container docker image '%s' must have a tag",
				i,
				app.Container.Docker.Image,
			)
		}
	}
	return nil
}

func marathonURL(conf config, force bool) string {
	url := fmt.Sprintf("https://%s/v2/groups", conf.Marathon.Host)
	if force {
		url = url + "?force=true"
	}

	return url
}

func marathonPush(conf config, jsonConfig []byte, force bool) (marathonResult, error) {
	// Prepare request
	u := marathonURL(conf, force)
	_, err := url.Parse(u)
	if err != nil {
		return marathonResult{}, fmt.Errorf(
			"Error parsing URL: %s",
			err,
		)
	}
	req, err := http.NewRequest("PUT", u, bytes.NewBuffer(jsonConfig))
	if err != nil {
		return marathonResult{}, fmt.Errorf(
			"Error building HTTP request: %s",
			err,
		)
	}
	if len(conf.Marathon.Headers) > 0 {
		req.Header = conf.Marathon.Headers
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return marathonResult{}, fmt.Errorf(
			"Error with PUT %s: %s",
			u,
			err,
		)
	}
	// Get response
	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return marathonResult{}, fmt.Errorf(
			"Error reading response: %s",
			err,
		)
	}
	// Parse response
	var result marathonResult
	if len(respBody) > 0 {
		err := json.Unmarshal(respBody, &result)
		if err != nil {
			return marathonResult{}, fmt.Errorf(
				"Error parsing response json: %s\nResponse:\n%s",
				err,
				respBody,
			)
		}
	}
	// Check response status
	if resp.StatusCode == 302 {
		return marathonResult{}, fmt.Errorf(
			"%s. Location: %+v\nResult: %+v",
			resp.Status,
			resp.Header.Get("Location"),
			result,
		)
	} else if resp.StatusCode == 422 {
		return marathonResult{}, fmt.Errorf(
			"%s\nResult: %+v\n\nConfig: %s",
			resp.Status,
			result,
			jsonConfig,
		)
	} else if resp.StatusCode != 200 {
		return marathonResult{}, fmt.Errorf(
			"%s\nResult: %+v",
			resp.Status,
			result,
		)
	}
	// Check result
	if result.DeploymentID == "" {
		return marathonResult{}, fmt.Errorf(
			"Deployment ID empty. Result: %+v",
			result,
		)
	}
	return result, nil
}
