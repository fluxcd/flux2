// +build unit

package main

import (
	"fmt"
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
	rootArgs.kubeconfig = testEnv.kubeConfigPath

	// Run tests
	code := m.Run()

	km.Stop()

	os.Exit(code)
}
