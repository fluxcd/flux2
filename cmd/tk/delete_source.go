package main

import (
	"github.com/spf13/cobra"
)

var deleteSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Delete sources commands",
}

func init() {
	deleteCmd.AddCommand(deleteSourceCmd)
}
