package main

import (
	"github-actions-exporter/cmd/github-actions-exporter/cmd"
	"os"
)

func main() {
	rootCmd := cmd.GetRootCmd(os.Args[1:])

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
