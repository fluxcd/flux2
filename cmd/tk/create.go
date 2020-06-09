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
	export   bool
)

func init() {
	createCmd.PersistentFlags().DurationVarP(&interval, "interval", "", time.Minute, "source sync interval")
	createCmd.PersistentFlags().BoolVar(&export, "export", false, "export in yaml format to stdout")
	rootCmd.AddCommand(createCmd)
}
