package main

import (
	"github.com/spf13/cobra"
)

var exportSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Export source commands",
}

func init() {
	exportCmd.AddCommand(exportSourceCmd)
}
