package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "pubsubtool",
		Short: "Google Pub/Sub client tester",
		Long:  "A simple Google Cloud Pub/Sub CLI with send and serve commands.",
	}

	root.AddCommand(sendCommand(), serveCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
