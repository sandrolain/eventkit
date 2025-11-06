package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "pgsqltool",
		Short: "PostgreSQL LISTEN/NOTIFY tester",
		Long:  "A simple PostgreSQL CLI with send and serve commands for LISTEN/NOTIFY.",
	}

	root.AddCommand(sendCommand(), serveCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
