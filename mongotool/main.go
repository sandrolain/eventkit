package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "mongotool",
		Short: "MongoDB testing tool",
		Long:  "A CLI tool for testing MongoDB connections and operations. Supports insert and changestream operations.",
	}

	root.AddCommand(sendCommand(), serveCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
