package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "redistool",
		Short: "Redis client tester",
		Long:  "A simple Redis CLI with send and serve commands for channels and streams.",
	}

	root.AddCommand(sendCommand(), serveCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
