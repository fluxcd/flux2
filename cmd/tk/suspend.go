package main

import (
	"github.com/spf13/cobra"
)

var suspendCmd = &cobra.Command{
	Use:   "suspend",
	Short: "Suspend commands",
}

func init() {
	rootCmd.AddCommand(suspendCmd)
}
