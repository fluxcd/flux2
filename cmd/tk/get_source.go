package main

import (
	"github.com/spf13/cobra"
)

var getSourceCmd = &cobra.Command{
	Use:   "sources",
	Short: "Get sources commands",
}

func init() {
	getCmd.AddCommand(getSourceCmd)
}
