package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check requirements and installation",
	Long: `
The check command will perform a series of checks to validate that
the local environment is configured correctly and if the installed components are healthy.`,
	Example: `  # Run pre-installation checks
  check --pre

  # Run installation checks
  check
`,
	RunE: runCheckCmd,
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	logAction("checking prerequisites")
	checkFailed := false
	if !sshCheck() {
		checkFailed = true
	}

	if !kubectlCheck(ctx, ">=1.18.0") {
		checkFailed = true
	}

	if !kubernetesCheck(">=1.14.0") {
		checkFailed = true
	}

	if checkPre {
		if checkFailed {
			os.Exit(1)
		}
		logSuccess("prerequisites checks passed")
		return nil
	}

	logAction("checking controllers")
	if !componentsCheck() {
		checkFailed = true
	}
	if checkFailed {
		os.Exit(1)
	}
	logSuccess("all checks passed")
	return nil
}

func sshCheck() bool {
	ok := true
	for _, cmd := range []string{"ssh-keygen", "ssh-keyscan"} {
		_, err := exec.LookPath(cmd)
		if err != nil {
			logFailure("%s not found", cmd)
			ok = false
		} else {
			logSuccess("%s found", cmd)
		}
	}

	return ok
}

func kubectlCheck(ctx context.Context, version string) bool {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		logFailure("kubectl not found")
		return false
	}

	command := "kubectl version --client --short | awk '{ print $3 }'"
	output, err := utils.execCommand(ctx, ModeCapture, command)
	if err != nil {
		logFailure("kubectl version can't be determined")
		return false
	}

	v, err := semver.ParseTolerant(output)
	if err != nil {
		logFailure("kubectl version can't be parsed")
		return false
	}

	rng, _ := semver.ParseRange(version)
	if !rng(v) {
		logFailure("kubectl version must be %s", version)
		return false
	}

	logSuccess("kubectl %s %s", v.String(), version)
	return true
}

func kubernetesCheck(version string) bool {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logFailure("kubernetes client initialization failed: %s", err.Error())
		return false
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logFailure("kubernetes client initialization failed: %s", err.Error())
		return false
	}

	ver, err := client.Discovery().ServerVersion()
	if err != nil {
		logFailure("kubernetes API call failed %s", err.Error())
		return false
	}

	v, err := semver.ParseTolerant(ver.String())
	if err != nil {
		logFailure("kubernetes version can't be determined")
		return false
	}

	rng, _ := semver.ParseRange(version)
	if !rng(v) {
		logFailure("kubernetes version must be %s", version)
		return false
	}

	logSuccess("kubernetes %s %s", v.String(), version)
	return true
}

func componentsCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ok := true
	for _, deployment := range components {
		command := fmt.Sprintf("kubectl -n %s rollout status deployment %s --timeout=%s",
			namespace, deployment, timeout.String())
		if output, err := utils.execCommand(ctx, ModeCapture, command); err != nil {
			logFailure("%s: %s", deployment, strings.TrimSuffix(output, "\n"))
			ok = false
		} else {
			logSuccess("%s is healthy", deployment)
		}
	}
	return ok
}
