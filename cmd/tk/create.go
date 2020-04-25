package main

import (
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create commands",
}

var (
	interval string
)

func init() {
	createCmd.PersistentFlags().StringVar(&interval, "interval", "1m", "source sync interval")
	rootCmd.AddCommand(createCmd)
}
