package main

import (
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete commands",
}

var (
	deleteSilent bool
)

func init() {
	deleteCmd.PersistentFlags().BoolVarP(&deleteSilent, "silent", "", false,
		"delete resource without asking for confirmation")

	rootCmd.AddCommand(deleteCmd)
}
