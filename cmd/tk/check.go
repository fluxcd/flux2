package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

var checkCmd = &cobra.Command{
	Use:   "check --pre",
	Short: "Check for potential problems",
	Long: `
The check command will perform a series of checks to validate that
the local environment and Kubernetes cluster are configured correctly.`,
	Example: `  check --pre`,
	RunE:    runCheckCmd,
}

var (
	kubeconfig string
	checkPre   bool
)

func init() {
	if home := homeDir(); home != "" {
		checkCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "", filepath.Join(home, ".kube", "config"),
			"path to the kubeconfig file")
	} else {
		checkCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "", "",
			"absolute path to the kubeconfig file")
	}
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

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
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
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Println(`✗`, "kubernetes client initialization failed", err.Error())
		return false
	}

	client, err := kubernetes.NewForConfig(config)
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
