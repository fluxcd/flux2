package main

import (
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get commands",
}

func init() {
	rootCmd.AddCommand(getCmd)
}
