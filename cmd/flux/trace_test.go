// +build unit

package main

import (
	"testing"
)

func TestTraceNoArgs(t *testing.T) {
	cmd := cmdTestCase{
		args:            "trace",
		testClusterMode: TestEnvClusterMode,
		assert:          assertError("object name is required"),
	}
	cmd.runTestCmd(t)
}

func TestTraceDeployment(t *testing.T) {
	cmd := cmdTestCase{
		args:            "trace podinfo -n podinfo --kind deployment --api-version=apps/v1",
		testClusterMode: TestEnvClusterMode,
		assert:          assertGoldenFile("testdata/trace/deployment.golden"),
		objectFile:      "testdata/trace/deployment.yaml",
	}
	cmd.runTestCmd(t)
}

func TestTraceHelmRelease(t *testing.T) {
	cmd := cmdTestCase{
		args:            "trace podinfo -n podinfo --kind HelmRelease --api-version=helm.toolkit.fluxcd.io/v2beta1",
		testClusterMode: TestEnvClusterMode,
		assert:          assertGoldenFile("testdata/trace/helmrelease.golden"),
		objectFile:      "testdata/trace/helmrelease.yaml",
	}
	cmd.runTestCmd(t)
}
