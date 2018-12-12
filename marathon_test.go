package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

const marathonExampleYAML = `
id: /path/to/apps
apps:
  - id: svc
    cpus: 0.2
    mem: 512
    instances: 1
    ports:
      - 0
    container:
      type: DOCKER
      docker:
        image: index.docker.io/library/hello-world
        network: HOST
        parameters:
          - key: log-driver
            value: journald
      volumes:
        - containerPath: /run/pald
          hostPath: /run/pald
          mode: RO
    env:
      SOME_ENV_VAR: some-env-var-value
    labels:
      some_label: some_label_value
    healthChecks:
      - protocol: HTTP
        path: /_healthcheck
        command:
          value: foo
        gracePeriodSeconds: 3
        intervalSeconds: 10
        portIndex: 0
        timeoutSeconds: 10
        maxConsecutiveFailures: 3
`

func TestMarathonParseYAML(t *testing.T) {
	app, err := marathonParseYAML([]byte(marathonExampleYAML))
	if err != nil {
		t.Errorf("Error parsing Marathon YAML: %s", err)
	}
	expect := "/path/to/apps"
	if app.ID != expect {
		t.Errorf(
			"Error parsing Marathon YAML. Expect id '%s', got '%s'",
			expect,
			app.ID,
		)
	}
}

func TestMarathonValidate(t *testing.T) {
	exampleAppInvalid, _ := marathonParseYAML([]byte(marathonExampleYAML))
	exampleAppValid, _ := marathonParseYAML([]byte(marathonExampleYAML))
	exampleAppValid.Apps[0].Container.Docker.Image = "index.docker.io/library/hello-world:latest"
	tests := []struct {
		app marathonGroup
		err string
	}{
		{
			app: exampleAppValid,
			err: "",
		},
		{
			app: marathonGroup{},
			err: "App id '' invalid",
		},
		{
			app: exampleAppInvalid,
			err: "App 0 container docker image 'index.docker.io/library/hello-world' must have a tag",
		},
	}
	for i, test := range tests {
		e := marathonValidate(test.app)
		if e != nil && test.err == "" {
			t.Errorf("(%d) Unexpected error: %s", i, e)
		} else if e == nil && test.err != "" {
			t.Errorf("(%d) Expected error '%s' but no error occurred", i, test.err)
		} else if e != nil && e.Error() != test.err {
			t.Errorf("(%d) Expected error '%s' but got '%s'", i, test.err, e)
		}
	}
}

func TestMarathonParseYAMLPorts(t *testing.T) {
	tests := []struct {
		yaml         string
		expectError  bool
		validateFunc func(*marathonGroup) bool
	}{
		{
			yaml:         "apps: [{ports: [0, 0]}]",
			validateFunc: func(g *marathonGroup) bool { return len(g.Apps[0].Ports) == 2 },
		},
		{
			yaml:         "apps: [{portDefinitions: [{port: 0, name: metrics}, {port: 0, name: pprof}]}]",
			validateFunc: func(g *marathonGroup) bool { return g.Apps[0].PortDefinitions[0].Name == "metrics" },
		},
		{
			yaml: "apps: [{portDefinitions: [{port: 0}, {port: 0, name: pprof}]}]",
			validateFunc: func(g *marathonGroup) bool {
				return g.Apps[0].PortDefinitions[0].Name == "" && g.Apps[0].PortDefinitions[1].Name == "pprof"
			},
		},
		{
			yaml:         "apps: [{portDefinitions: [0, 0]}]",
			expectError:  true,
			validateFunc: nil,
		},
	}
	for i, test := range tests {
		config, err := marathonParseYAML([]byte(test.yaml))
		switch {
		case err != nil && test.expectError:
		case err == nil && test.expectError:
			t.Errorf("expected error, didn't get it")
		case err != nil:
			t.Errorf("unexpected error in test %v: %v", i, err)
		case !test.validateFunc(&config):
			t.Errorf("invalid result for test %v: %#v", i, config)
		}
	}
}

func TestMarathonYAMLtoJSON(t *testing.T) {
	tests := []struct {
		yaml []byte
		json []byte
	}{
		{
			yaml: []byte(``),
			json: []byte(`{"id":""}`),
		},
		{
			yaml: []byte(`apps: [{ports: [0, 0]}]`),
			json: []byte(`{"id":"","apps":[{"id":"","instances":0,"cpus":0,"mem":0,"constraints":null,"ports":[0,0],"requirePorts":false,"container":{"type":"","volumes":null}}]}`),
		},
		// validate that portDefinition doesn't create empty name fields.
		{
			yaml: []byte(`apps: [{portDefinitions: [{port: 0}]}]`),
			json: []byte(`{"id":"","apps":[{"id":"","instances":0,"cpus":0,"mem":0,"constraints":null,"portDefinitions":[{"port":0}],"requirePorts":false,"container":{"type":"","volumes":null}}]}`),
		},
		{
			yaml: []byte(`apps: [{portDefinitions: [{port: 0, name: metrics}, {port: 0, name: pprof}]}]`),
			json: []byte(`{"id":"","apps":[{"id":"","instances":0,"cpus":0,"mem":0,"constraints":null,"portDefinitions":[{"port":0,"name":"metrics"},{"port":0,"name":"pprof"}],"requirePorts":false,"container":{"type":"","volumes":null}}]}`),
		},
		{
			yaml: []byte(`apps: [{healthChecks: [{protocol: COMMAND, command: {value: foo}, gracePeriodSeconds: 2, intervalSeconds: 2, portIndex: 1, timeoutSeconds: 2, maxConsecutiveFailures: 2 }]}]`),
			json: []byte(`{"id":"","apps":[{"id":"","instances":0,"cpus":0,"mem":0,"constraints":null,"requirePorts":false,"container":{"type":"","volumes":null},"healthChecks":[{"protocol":"COMMAND","command":{"value":"foo"},"gracePeriodSeconds":2,"intervalSeconds":2,"portIndex":1,"timeoutSeconds":2,"maxConsecutiveFailures":2}]}]}`),
		},
	}
	for i, test := range tests {
		config, err := marathonParseYAML(test.yaml)
		if err != nil {
			t.Fatalf("(%d) Unexpected error parsing YAML: %s", i, err)
		}
		marathonJSON, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("(%d) Unexpected error marshaling JSON: %s", i, err)
		}
		if !bytes.Equal(marathonJSON, test.json) {
			t.Fatalf(
				"(%d) JSON mismatch.\nExpected:\t%s\nGot:\t\t%s\n", i,
				test.json,
				marathonJSON,
			)
		}
	}
}
