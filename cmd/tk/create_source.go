package main

import (
	"github.com/spf13/cobra"
)

var createSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Create source commands",
}

func init() {
	createCmd.AddCommand(createSourceCmd)
}
