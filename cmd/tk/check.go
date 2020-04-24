package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check --pre",
	Short: "Check for potential problems",
	Long: `
The check command will perform a series of checks to validate that
the local environment is configured correctly.`,
	Example: `  check --pre`,
	RunE:    runCheckCmd,
}

var (
	checkPre bool
)

func init() {
	checkCmd.Flags().BoolVarP(&checkPre, "pre", "", false,
		"only run pre-installation checks")

	rootCmd.AddCommand(checkCmd)
}

func runCheckCmd(cmd *cobra.Command, args []string) error {
	if !checkLocal() {
		os.Exit(1)
	}
	if checkPre {
		fmt.Println(`✔`, "all prerequisites checks passed")
		return nil
	}

	if !checkRemote() {
		os.Exit(1)
	} else {
		fmt.Println(`✔`, "all checks passed")
	}
	return nil
}

func checkLocal() bool {
	ok := true
	for _, cmd := range []string{"kubectl", "kustomize"} {
		_, err := exec.LookPath(cmd)
		if err != nil {
			fmt.Println(`✗`, cmd, "not found")
			ok = false
		} else {
			fmt.Println(`✔`, cmd, "found")
		}
	}
	return ok
}

func checkRemote() bool {
	client, err := NewKubernetesClient()
	if err != nil {
		fmt.Println(`✗`, "kubernetes client initialization failed", err.Error())
		return false
	}

	ver, err := client.Discovery().ServerVersion()
	if err != nil {
		fmt.Println(`✗`, "kubernetes API call failed", err.Error())
		return false
	}

	fmt.Println(`✔`, "kubernetes version", ver.String())
	return true
}
