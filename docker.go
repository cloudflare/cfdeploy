package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"text/template"
)

type dockerImage struct {
	Repository string
	Name       string
	Tag        string
}

func (i *dockerImage) Validate() error {
	if i.Repository == "" {
		return fmt.Errorf("Image repository cannot be blank")
	}
	if strings.Contains(i.Repository, "/") {
		return fmt.Errorf("Image repository cannot contain forward slashes")
	}
	if i.Name == "" {
		return fmt.Errorf("Image name cannot be blank")
	}
	if i.Name[0:1] == "/" {
		return fmt.Errorf("Image name cannot start with forward slash")
	}
	if i.Name[len(i.Name)-1:] == "/" {
		return fmt.Errorf("Image name cannot end with forward slash")
	}
	if i.Tag == "" {
		return fmt.Errorf("Image tag cannot be blank")
	}
	if strings.Contains(i.Tag, ":") {
		return fmt.Errorf("Image tag cannot contain colon")
	}
	return nil
}

func (i *dockerImage) String() string {
	return i.Repository + "/" + i.Name + ":" + i.Tag
}

var dockerTagVars struct {
	GitBranch   string
	GitRevCount string
	GitRevShort string
}

// dockerTag renders a tag template
func dockerTag(tagTemplate string) (string, error) {
	// Lazy load git vars
	var err error
	var out []byte
	if strings.Contains(tagTemplate, ".GitBranch") && dockerTagVars.GitBranch == "" {
		/* #nosec */
		out, err = exec.Command("git", "symbolic-ref", "--short", "HEAD").Output()
		if err != nil {
			return "", err
		}
		dockerTagVars.GitBranch = strings.TrimSpace(string(out))
	}
	if strings.Contains(tagTemplate, ".GitRevCount") && dockerTagVars.GitRevCount == "" {
		/* #nosec */
		out, err = exec.Command("git", "rev-list", "--count", "HEAD").Output()
		if err != nil {
			return "", err
		}
		dockerTagVars.GitRevCount = strings.TrimSpace(string(out))
	}
	if strings.Contains(tagTemplate, ".GitRevShort") && dockerTagVars.GitRevShort == "" {
		/* #nosec */
		out, err = exec.Command("git", "rev-parse", "--short", "HEAD").Output()
		if err != nil {
			return "", err
		}
		dockerTagVars.GitRevShort = strings.TrimSpace(string(out))
	}
	// Render template
	t := template.Must(template.New("tagTemplate").Parse(tagTemplate))
	var buf bytes.Buffer
	err = t.Execute(&buf, dockerTagVars)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// dockerImageList compiles a list of dockerImage's from a given config struct
func dockerImageList(c config, e string) (images map[string]dockerImage, err error) {
	images = map[string]dockerImage{}
	for imageKey, envImage := range c.Environments[e].Images {
		image := dockerImage{}
		// Add repository
		if envImage.Repository != "" {
			image.Repository = envImage.Repository
		} else if c.Image.Repository != "" {
			image.Repository = c.Image.Repository
		} else {
			err = fmt.Errorf(
				"Could not find image repository in config for %s",
				imageKey,
			)
			return
		}
		// Add name
		if envImage.Name != "" {
			image.Name = envImage.Name
		} else if c.Image.Name != "" {
			image.Name = c.Image.Name
		} else {
			err = fmt.Errorf(
				"Could not find image name in config for %s",
				imageKey,
			)
			return
		}
		// Add tag
		if envImage.TagTemplate != "" {
			image.Tag, err = dockerTag(envImage.TagTemplate)
		} else if c.Image.TagTemplate != "" {
			image.Tag, err = dockerTag(c.Image.TagTemplate)
		} else {
			err = fmt.Errorf(
				"Could not find image tag in config for %s",
				imageKey,
			)
		}
		if err != nil {
			return
		}
		// Append image
		images[imageKey] = image
	}
	return
}

func dockerCheckImage(image dockerImage) error {

	// Validate image fields
	err := image.Validate()
	if err != nil {
		return err
	}

	// Attempt to verify image exists without auth
	authHeader, err := dockerGetImage(image.Repository, image.Name, image.Tag, "")
	if err != nil {
		return err
	}

	// Return if auth not required
	if authHeader == "" {
		return nil
	}

	// Auth required, so get a signed token (with grant)
	var authToken string
	authToken, err = dockerGetToken(authHeader)
	if err != nil {
		return err
	}

	// Verify image exists with auth
	_, err = dockerGetImage(image.Repository, image.Name, image.Tag, authToken)
	return err

}

// dockerGetImage will query the image manifest to verify an image exists.
// if the response is a 401 and contains a Www-Authenticate header,
// it will be returned in authHeader. if an error occurs, err will be returned.
// if the image is found, authHeader and err will be empty.
func dockerGetImage(imageRepo, imageName, imageTag, token string) (authHeader string, err error) {
	// Build registry URL for image/tag
	url := fmt.Sprintf(
		"https://%s/v2/%s/manifests/%s",
		imageRepo,
		imageName,
		imageTag,
	)
	// Build request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err = fmt.Errorf("Error building request: %s", err)
		return
	}
	req.Header = http.Header{
		"Content-Type": []string{
			"application/json; charset=utf-8",
		},
	}
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	// Make request
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	// Check if auth is required
	if resp.StatusCode == 401 {
		if token != "" {
			err = fmt.Errorf(
				"HTTP response should not be 401 when token is provided",
			)
			return
		}
		// Get WWW-Authenticate header
		authHeader = resp.Header.Get("Www-Authenticate")
		if authHeader == "" {
			err = fmt.Errorf(
				"Expected 401 response to contain Www-Authenticate error",
			)
		}
		return
	}
	// Check if image/tag not found
	if resp.StatusCode == 404 {
		err = fmt.Errorf(
			"Docker image/tag (%s/%s:%s) not found",
			imageRepo,
			imageName,
			imageTag,
		)
		return
	}
	// Check request was valid
	if resp.StatusCode != 200 {
		err = fmt.Errorf(
			"Response code not 200, got: %d",
			resp.StatusCode,
		)
		return
	}
	// Get response
	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close() // #nosec G104
	if err != nil {
		err = fmt.Errorf(
			"Error reading response: %s",
			err,
		)
		return
	}
	// Parse response
	var searchResult struct {
		Name  string
		Tag   string
		Error struct {
			Code    string
			Message string
			Detail  struct {
				Type   string
				Name   string
				Action string
			}
		}
	}
	err = json.Unmarshal(respBody, &searchResult)
	if err != nil {
		err = fmt.Errorf("Error parsing response json: %s", err)
		return
	}
	// Check response values
	if searchResult.Error.Code != "" {
		err = fmt.Errorf(
			"GET %s\nRegistry search error (%s): %s",
			url,
			searchResult.Error.Code,
			searchResult.Error.Message,
		)
		return
	}
	if searchResult.Name == "" || searchResult.Tag == "" {
		err = fmt.Errorf(
			"GET %s\nImage name/tag invalid: %+v",
			url,
			searchResult,
		)
	}
	return
}

// dockerGetToken will get a secure Docker registry token
func dockerGetToken(authHeader string) (string, error) {
	// Extract auth realm/service/scope from auth header
	var realm, service, scope string
	wwwAuthTrimmed := strings.TrimPrefix(authHeader, "Bearer ")
	wwwAuthSplit := strings.Split(string(wwwAuthTrimmed), ",")
	for _, wwwAuthPart := range wwwAuthSplit {
		wwwAuth := strings.Trim(wwwAuthPart, " ")
		if wwwAuth == "" {
			continue
		}
		wwwAuthSplit := strings.Split(wwwAuth, "=")
		if len(wwwAuthSplit) != 2 {
			continue
		}
		value := strings.TrimSpace(strings.Trim(wwwAuthSplit[1], "\""))
		switch wwwAuthSplit[0] {
		case "realm":
			realm = value
		case "service":
			service = value
		case "scope":
			scope = value
		}
	}
	if realm == "" || service == "" || scope == "" {
		return "", fmt.Errorf(
			"Realm, service or scope empty (realm: '%s', service: '%s', scope '%s')",
			realm,
			service,
			scope,
		)
	}
	// Build auth URL
	reqURL, err := url.Parse(realm)
	if err != nil {
		return "", fmt.Errorf(
			"Error parsing realm URL: %s",
			err,
		)
	}
	reqQuery := url.Values{}
	reqQuery.Add("service", service)
	reqQuery.Add("scope", scope)
	reqURL.RawQuery = reqQuery.Encode()
	authURL := reqURL.String()
	// Request auth token
	resp, err := http.Get(authURL) // #nosec G107
	if err != nil {
		return "", fmt.Errorf(
			"GET %s\n%s",
			authURL,
			err,
		)
	}
	// Get response
	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close() // #nosec G104
	if err != nil {
		return "", fmt.Errorf(
			"GET %s\nError reading response: %s",
			authURL,
			err,
		)
	}
	// Parse response
	var respObject struct {
		Token string
	}
	err = json.Unmarshal(respBody, &respObject)
	if err != nil {
		return "", fmt.Errorf(
			"GET %s\nError parsing json: %s",
			authURL,
			err,
		)
	}
	// Check token valid
	if respObject.Token == "" {
		return "", fmt.Errorf(
			"GET %s\nAuth token invalid. Response: %+v",
			authURL,
			respObject,
		)
	}
	return respObject.Token, nil
}
