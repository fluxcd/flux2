/*
Copyright 2020 The Flux CD contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	Long: `The check command will perform a series of checks to validate that
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

	logger.Actionf("checking prerequisites")
	checkFailed := false

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
		logger.Successf("prerequisites checks passed")
		return nil
	}

	logger.Actionf("checking controllers")
	if !componentsCheck() {
		checkFailed = true
	}
	if checkFailed {
		os.Exit(1)
	}
	logger.Successf("all checks passed")
	return nil
}

func kubectlCheck(ctx context.Context, version string) bool {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		logger.Failuref("kubectl not found")
		return false
	}

	command := "kubectl version --client --short | awk '{ print $3 }'"
	output, err := utils.execCommand(ctx, ModeCapture, command)
	if err != nil {
		logger.Failuref("kubectl version can't be determined")
		return false
	}

	v, err := semver.ParseTolerant(output)
	if err != nil {
		logger.Failuref("kubectl version can't be parsed")
		return false
	}

	rng, _ := semver.ParseRange(version)
	if !rng(v) {
		logger.Failuref("kubectl version must be %s", version)
		return false
	}

	logger.Successf("kubectl %s %s", v.String(), version)
	return true
}

func kubernetesCheck(version string) bool {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logger.Failuref("Kubernetes client initialization failed: %s", err.Error())
		return false
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Failuref("Kubernetes client initialization failed: %s", err.Error())
		return false
	}

	ver, err := client.Discovery().ServerVersion()
	if err != nil {
		logger.Failuref("Kubernetes API call failed: %s", err.Error())
		return false
	}

	v, err := semver.ParseTolerant(ver.String())
	if err != nil {
		logger.Failuref("Kubernetes version can't be determined")
		return false
	}

	rng, _ := semver.ParseRange(version)
	if !rng(v) {
		logger.Failuref("Kubernetes version must be %s", version)
		return false
	}

	logger.Successf("Kubernetes %s %s", v.String(), version)
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
			logger.Failuref("%s: %s", deployment, strings.TrimSuffix(output, "\n"))
			ok = false
		} else {
			logger.Successf("%s is healthy", deployment)
		}
	}
	return ok
}
