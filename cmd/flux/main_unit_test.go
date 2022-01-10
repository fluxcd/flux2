// +build unit

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"testing"
)

// The test environment is long running process shared between tests, initialized
// by a `TestMain` function depending on how the test is involved and which tests
// are a part of the build.
var testEnv *testEnvKubeManager

func TestMain(m *testing.M) {
	// Ensure tests print consistent timestamps regardless of timezone
	os.Setenv("TZ", "UTC")

	// Creating the test env manager sets rootArgs client flags
	km, err := NewTestEnvKubeManager(TestEnvClusterMode)
	if err != nil {
		panic(fmt.Errorf("error creating kube manager: '%w'", err))
	}
	testEnv = km
	// rootArgs.kubeconfig = testEnv.kubeConfigPath
	kubeconfigArgs.KubeConfig = &testEnv.kubeConfigPath

	// Run tests
	code := m.Run()

	km.Stop()

	os.Exit(code)
}

func setupTestNamespace(namespace string, t *testing.T) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	err := testEnv.client.Create(context.Background(), ns)
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}
	t.Cleanup(func() {
		_ = testEnv.client.Delete(context.Background(), ns)
	})
}
