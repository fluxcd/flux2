package main

import (
	"github.com/spf13/cobra"
)

var syncSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Synchronize source commands",
}

func init() {
	syncCmd.AddCommand(syncSourceCmd)
}
