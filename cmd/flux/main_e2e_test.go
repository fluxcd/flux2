//go:build e2e
// +build e2e

/*
Copyright 2021 The Flux authors

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
	"testing"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

func TestMain(m *testing.M) {
	log.SetLogger(logr.New(log.NullLogSink{}))

	// Ensure tests print consistent timestamps regardless of timezone
	os.Setenv("TZ", "UTC")

	testEnv, err := NewTestEnvKubeManager(ExistingClusterMode)
	if err != nil {
		panic(fmt.Errorf("error creating kube manager: '%w'", err))
	}
	kubeconfigArgs.KubeConfig = &testEnv.kubeConfigPath

	// Install Flux.
	output, err := executeCommand("install --components-extra=image-reflector-controller,image-automation-controller")
	if err != nil {
		panic(fmt.Errorf("install failed: %s error:'%w'", output, err))
	}

	// Run tests
	code := m.Run()

	// Uninstall Flux
	output, err = executeCommand("uninstall -s --keep-namespace")
	if err != nil {
		panic(fmt.Errorf("uninstall failed: %s error:'%w'", output, err))
	}

	// Delete namespace and wait for finalisation
	kubectlArgs := []string{"delete", "namespace", "flux-system"}
	_, err = utils.ExecKubectlCommand(context.TODO(), utils.ModeStderrOS, *kubeconfigArgs.KubeConfig, *kubeconfigArgs.Context, kubectlArgs...)
	if err != nil {
		panic(fmt.Errorf("delete namespace error:'%w'", err))
	}

	testEnv.Stop()

	os.Exit(code)
}

func execSetupTestNamespace(namespace string) (func(), error) {
	kubectlArgs := []string{"create", "namespace", namespace}
	_, err := utils.ExecKubectlCommand(context.TODO(), utils.ModeStderrOS, *kubeconfigArgs.KubeConfig, *kubeconfigArgs.Context, kubectlArgs...)
	if err != nil {
		return nil, err
	}

	return func() {
		kubectlArgs := []string{"delete", "namespace", namespace}
		utils.ExecKubectlCommand(context.TODO(), utils.ModeCapture, *kubeconfigArgs.KubeConfig, *kubeconfigArgs.Context, kubectlArgs...)
	}, nil
}
