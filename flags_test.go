package main

import (
	"flag"
)

var integration bool

func init() {
	flag.BoolVar(&integration, "integration", false, "Run integration tests")
	flag.Parse()
}
