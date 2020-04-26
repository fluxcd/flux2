package main

import (
	"time"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create commands",
}

var (
	interval time.Duration
)

func init() {
	createCmd.PersistentFlags().DurationVarP(&interval, "interval", "", time.Minute, "source sync interval")
	rootCmd.AddCommand(createCmd)
}
