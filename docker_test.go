package main

import (
	"strings"
	"testing"
)

func TestDockerTag(t *testing.T) {
	tests := []struct {
		tagTemplate string
	}{
		{
			tagTemplate: "{{ .GitBranch }}",
		},
		{
			tagTemplate: "{{ .GitRevCount }}",
		},
		{
			tagTemplate: "{{ .GitRevShort }}",
		},
	}
	for i, test := range tests {
		tag, err := dockerTag(test.tagTemplate)
		if err != nil {
			t.Errorf("(%d) Unexpected error: %s", i, err)
		}
		if tag == "" {
			t.Errorf("(%d) Tag should not be empty", i)
		}
		if strings.Contains(tag, "\n") {
			t.Errorf("(%d) Tag should not contain newline characters", i)
		}
	}
}

func TestDockerImageList(t *testing.T) {
	c := config{
		Image: configImage{
			Repository:  "index.docker.io",
			TagTemplate: "latest",
		},
		Environments: map[string]configEnvironment{
			"prod": configEnvironment{
				Images: map[string]configImage{
					"hello": configImage{
						Name: "library/hello-world",
					},
				},
			},
		},
	}
	images, err := dockerImageList(c, "prod")
	if err != nil {
		t.Errorf("Unexpected error getting Docker image list: %s", err)
	}
	if len(images) != 1 {
		t.Fatalf("Expected image list length = 1, got: %d", len(images))
	}
	if _, ok := images["hello"]; !ok {
		t.Fatalf(
			"Expected 1st image key = '%s'",
			"hello",
		)
	}
	if images["hello"].Repository != "index.docker.io" {
		t.Errorf(
			"Expected 1st image repository = '%s', got: '%s",
			"index.docker.io",
			images["hello"].Repository,
		)
	}
	if images["hello"].Name != "library/hello-world" {
		t.Errorf(
			"Expected 1st image name = '%s', got: '%s",
			"library/hello-world",
			images["hello"].Name,
		)
	}
	if images["hello"].Tag != "latest" {
		t.Errorf(
			"Expected 1st image tag = '%s', got: '%s",
			"latest",
			images["hello"].Tag,
		)
	}
}

func TestDockerCheckImage(t *testing.T) {
	tests := []struct {
		image dockerImage
		err   string
	}{
		// TODO: Find a secure docker registry that supports anonymous tokens
		// {
		// 	image: dockerImage{
		// 		Repository: "index.docker.io",
		// 		Name:       "secure-image",
		// 		Tag:        "b65349dad81",
		// 	},
		// 	err: "",
		// },
		{
			image: dockerImage{
				Repository: "index.docker.io",
				Name:       "library/hello-world",
				Tag:        "latest",
			},
			err: "",
		},
		{
			image: dockerImage{
				Repository: "index.docker.io",
				Name:       "library/hello-world",
				Tag:        "this-tag-does-not-exist",
			},
			err: "Docker image/tag (index.docker.io/library/hello-world:this-tag-does-not-exist) not found",
		},
		{
			image: dockerImage{},
			err:   "Image repository cannot be blank",
		},
		{
			image: dockerImage{
				Repository: "index.docker.io",
			},
			err: "Image name cannot be blank",
		},
		{
			image: dockerImage{
				Repository: "index.docker.io",
				Name:       "library/hello-world",
			},
			err: "Image tag cannot be blank",
		},
	}
	for i, test := range tests {
		e := dockerCheckImage(test.image)
		if e != nil && test.err == "" {
			t.Errorf("(%d) Unexpected error: %s", i, e)
		} else if e == nil && test.err != "" {
			t.Errorf("(%d) Expected error '%s' but no error occured", i, test.err)
		} else if e != nil && e.Error() != test.err {
			t.Errorf("(%d) Expected error '%s' but got '%s'", i, test.err, e)
		}
	}
}
