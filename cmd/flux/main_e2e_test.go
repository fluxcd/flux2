// +build e2e

package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/fluxcd/flux2/internal/utils"
)

func TestMain(m *testing.M) {
	// Ensure tests print consistent timestamps regardless of timezone
	os.Setenv("TZ", "UTC")

	// Install Flux
	km, err := NewTestEnvKubeManager(ExistingClusterMode)
	if err != nil {
		panic(fmt.Errorf("error creating kube manager: '%w'", err))
	}
	rootCtx.kubeManager = km
	output, err := executeCommand("install --components-extra=image-reflector-controller,image-automation-controller")
	if err != nil {
		panic(fmt.Errorf("install falied: %s error:'%w'", output, err))
	}

	// Run tests
	code := m.Run()

	// Uninstall Flux
	output, err = executeCommand("uninstall -s --keep-namespace")
	if err != nil {
		panic(fmt.Errorf("uninstall falied: %s error:'%w'", output, err))
	}

	// Delete namespace and wait for finalisation
	kubectlArgs := []string{"delete", "namespace", "flux-system"}
	_, err = utils.ExecKubectlCommand(context.TODO(), utils.ModeStderrOS, rootArgs.kubeconfig, rootArgs.kubecontext, kubectlArgs...)
	if err != nil {
		panic(fmt.Errorf("delete namespace error:'%w'", err))
	}

	os.Exit(code)
}

func setupTestNamespace(namespace string) (func(), error) {
	kubectlArgs := []string{"create", "namespace", namespace}
	_, err := utils.ExecKubectlCommand(context.TODO(), utils.ModeStderrOS, rootArgs.kubeconfig, rootArgs.kubecontext, kubectlArgs...)
	if err != nil {
		return nil, err
	}

	return func() {
		kubectlArgs := []string{"delete", "namespace", namespace}
		utils.ExecKubectlCommand(context.TODO(), utils.ModeCapture, rootArgs.kubeconfig, rootArgs.kubecontext, kubectlArgs...)
	}, nil
}
