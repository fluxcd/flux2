// +build unit

package main

import (
	"testing"
)

func TestVersion(t *testing.T) {
	cmd := cmdTestCase{
		args:            "--version",
		testClusterMode: TestEnvClusterMode,
		assert:          assertGoldenValue("flux version 0.0.0-dev.0\n"),
	}
	cmd.runTestCmd(t)
}
