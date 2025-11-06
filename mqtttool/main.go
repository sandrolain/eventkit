package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "mqtttool",
		Short: "MQTT client/server tester",
		Long:  "A simple MQTT client/server CLI with send and serve commands.",
	}

	root.AddCommand(sendCommand(), serveCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
