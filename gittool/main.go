package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "gittool",
		Short: "Git source tester",
		Long:  "A simple Git CLI with only a send command that commits and pushes periodically.",
	}

	root.AddCommand(sendCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
