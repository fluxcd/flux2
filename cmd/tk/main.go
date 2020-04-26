package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var VERSION = "0.0.1"

var rootCmd = &cobra.Command{
	Use:           "tk",
	Short:         "Command line utility for assembling Kubernetes CD pipelines",
	Long:          `Command line utility for assembling Kubernetes CD pipelines.`,
	Version:       VERSION,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var (
	kubeconfig string
	namespace  string
	timeout    time.Duration
	verbose    bool
	utils      Utils
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "", "gitops-system",
		"the namespace scope for this operation")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "", 5*time.Minute,
		"timeout for this operation")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false,
		"print generated objects")
}

func main() {
	log.SetFlags(0)
	generateDocs()
	kubeconfigFlag()
	if err := rootCmd.Execute(); err != nil {
		logFailure("%v", err)
		os.Exit(1)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func logAction(format string, a ...interface{}) {
	fmt.Println(`✚`, fmt.Sprintf(format, a...))
}

func logSuccess(format string, a ...interface{}) {
	fmt.Println(`✔`, fmt.Sprintf(format, a...))
}

func logFailure(format string, a ...interface{}) {
	fmt.Println(`✗`, fmt.Sprintf(format, a...))
}

func generateDocs() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "docgen" {
		rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "", "~/.kube/config",
			"path to the kubeconfig file")
		err := doc.GenMarkdownTree(rootCmd, "./docs/cmd")
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
}

func kubeconfigFlag() {
	if home := homeDir(); home != "" {
		rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "", filepath.Join(home, ".kube", "config"),
			"path to the kubeconfig file")
	} else {
		rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "", "",
			"absolute path to the kubeconfig file")
	}
}
