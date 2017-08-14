package main

import (
	"fmt"
)

func promptConfirm(prompt string) bool {
	var s string
	for {
		fmt.Printf("%s (y/n): ", prompt)
		fmt.Scanln(&s)
		switch s {
		case "Yes", "yes", "y", "Y":
			return true
		case "No", "no", "n", "N":
			return false
		}
	}
}
