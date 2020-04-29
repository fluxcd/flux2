package main

import (
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize commands",
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
