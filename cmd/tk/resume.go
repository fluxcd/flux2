package main

import (
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume commands",
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
