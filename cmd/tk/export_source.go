package main

import (
	"github.com/spf13/cobra"
)

var exportSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Export source commands",
}

var (
	exportSourceWithCred bool
)

func init() {
	exportSourceCmd.PersistentFlags().BoolVar(&exportSourceWithCred, "with-credentials", false, "include credential secrets")

	exportCmd.AddCommand(exportSourceCmd)
}
