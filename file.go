package main

import (
	"bytes"
	"io/ioutil"
	"text/template"
)

type fileVars struct {
	Images map[string]string
}

func fileLoad(path string, vars fileVars) ([]byte, error) {

	// Read file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Create template from file
	tpl, err := template.New("file").Parse(string(data))
	if err != nil {
		return nil, err
	}

	// Parse vars into template
	var buf bytes.Buffer
	err = tpl.Execute(&buf, vars)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil

}
