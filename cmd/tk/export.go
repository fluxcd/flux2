package main

import (
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export commands",
}

var (
	exportAll bool
)

func init() {
	exportCmd.PersistentFlags().BoolVar(&exportAll, "all", false, "select all resources")

	rootCmd.AddCommand(exportCmd)
}
