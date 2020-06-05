package main

import (
	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap commands",
}

var (
	bootstrapVersion string
)

func init() {
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapVersion, "version", "master", "toolkit tag or branch")

	rootCmd.AddCommand(bootstrapCmd)
}
