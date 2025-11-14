package main

import (
	"os"

	"github.com/suchpuppet/kubectl-mc/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
