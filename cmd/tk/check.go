package main

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check requirements",
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	logAction("starting verification")
	checkFailed := false
	if !sshCheck() {
		checkFailed = true
	}

	if !kubectlCheck(ctx, ">=1.18.0") {
		checkFailed = true
	}

	if !kustomizeCheck(ctx, ">=3.5.0") {
		checkFailed = true
	}

	if checkPre {
		if checkFailed {
			os.Exit(1)
		}
		logSuccess("all prerequisites checks passed")
		return nil
	}

	if !kubernetesCheck(">=1.14.0") {
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

func kustomizeCheck(ctx context.Context, version string) bool {
	_, err := exec.LookPath("kustomize")
	if err != nil {
		logFailure("kustomize not found")
		return false
	}

	command := "kustomize version --short | awk '{ print $1 }' | cut -c2-"
	output, err := utils.execCommand(ctx, ModeCapture, command)
	if err != nil {
		logFailure("kustomize version can't be determined")
		return false
	}

	if strings.Contains(output, "kustomize/") {
		command = "kustomize version --short | awk '{ print $1 }' | cut -c12-"
		output, err = utils.execCommand(ctx, ModeCapture, command)
		if err != nil {
			logFailure("kustomize version can't be determined")
			return false
		}
	}

	v, err := semver.ParseTolerant(output)
	if err != nil {
		logFailure("kustomize version can't be parsed")
		return false
	}

	rng, _ := semver.ParseRange(version)
	if !rng(v) {
		logFailure("kustomize version must be %s", version)
		return false
	}

	logSuccess("kustomize %s %s", v.String(), version)
	return true
}

func kubernetesCheck(version string) bool {
	client, err := utils.kubeClient(kubeconfig)
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
