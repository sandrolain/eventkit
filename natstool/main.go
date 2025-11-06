package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "natstool",
		Short: "NATS client tester",
		Long:  "A simple NATS CLI with send and serve commands (supports JetStream).",
	}

	root.AddCommand(sendCommand(), serveCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
