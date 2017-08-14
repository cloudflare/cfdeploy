package main

import (
	"testing"
)

const ConfigExample = `
marathon:
  host: www.example.com
image:
  repository: index.docker.io
  tagTemplate: "{{ .GitRevCount }}-{{ .GitRevShort }}"

environments:
  prod:
    marathon:
      file: prod.yaml
    images:
      hello:
        name: library/hello-world
  staging:
    marathon:
      file: staging.yaml
    images:
      hello:
        name: library/hello-world
`

func TestConfigLoad(t *testing.T) {

	tests := []struct {
		Data                  string
		Flags                 flags
		ExpectMarathonHost    string
		ExpectMarathonHeaders map[string]string
		ExpectError           string
	}{
		// Basic case
		{
			Data: ConfigExample,
			Flags: flags{
				env: "prod",
			},
			ExpectMarathonHost:    "www.example.com",
			ExpectMarathonHeaders: map[string]string{},
		},
		// Override marathon host
		{
			Data: ConfigExample,
			Flags: flags{
				env:          "prod",
				marathonHost: "test.example.com",
			},
			ExpectMarathonHost:    "test.example.com",
			ExpectMarathonHeaders: map[string]string{},
		},
		// Override marathon curl opts with single header
		{
			Data: ConfigExample,
			Flags: flags{
				env:              "prod",
				marathonCurlOpts: "-H \"OauthEmail: no-reply@cloudflare.com\"",
			},
			ExpectMarathonHost: "www.example.com",
			ExpectMarathonHeaders: map[string]string{
				"OauthEmail": "no-reply@cloudflare.com",
			},
		},
		// Override marathon curl opts with multiple headers
		{
			Data: ConfigExample,
			Flags: flags{
				env:              "prod",
				marathonCurlOpts: "-H \"OauthEmail: no-reply@cloudflare.com\" -H \"OauthExpires: 1501617700\"",
			},
			ExpectMarathonHost: "www.example.com",
			ExpectMarathonHeaders: map[string]string{
				"OauthEmail":   "no-reply@cloudflare.com",
				"OauthExpires": "1501617700",
			},
		},
	}

	for i, test := range tests {

		// Test loading
		config, err := configLoad(
			[]byte(test.Data),
			test.Flags,
		)
		if test.ExpectError != "" && err == nil {
			t.Errorf(
				"(%d) Expected error loading config YAML but did not get: %s",
				i,
				test.ExpectError,
			)
		} else if test.ExpectError == "" && err != nil {
			t.Errorf("(%d) Unexpected error loading config YAML: %s", i, err)
		} else if err != nil && test.ExpectError != err.Error() {
			t.Errorf(
				"(%d) Unexpected error loading config YAML.\nExpected: %s\nGot: %s\n",
				i,
				test.ExpectError,
				err,
			)
		}

		// Test marathon host override
		if config.Marathon.Host != test.ExpectMarathonHost {
			t.Errorf(
				"(%d) Error parsing config YAML. Expect marathon.host '%s', got '%s'",
				i,
				test.ExpectMarathonHost,
				config.Marathon.Host,
			)
		}

		// Test marathon headers
		for hKey, hVal := range test.ExpectMarathonHeaders {
			got := config.Marathon.Headers.Get(hKey)
			if got != hVal {
				t.Errorf(
					"(%d) Error parsing Marathon headers. Expect '%s' = '%s', got '%s'",
					i,
					hKey,
					hVal,
					got,
				)
			}
		}

	}
}
