// +build e2e

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fluxcd/flux2/internal/utils"
	"k8s.io/apimachinery/pkg/version"
)

func TestCheckPre(t *testing.T) {
	jsonOutput, err := utils.ExecKubectlCommand(context.TODO(), utils.ModeCapture, rootArgs.kubeconfig, rootArgs.kubecontext, "version", "--output", "json")
	if err != nil {
		t.Fatalf("Error running utils.ExecKubectlCommand: %v", err.Error())
	}

	var versions map[string]version.Info
	if err := json.Unmarshal([]byte(jsonOutput), &versions); err != nil {
		t.Fatalf("Error unmarshalling: %v", err.Error())
	}

	clientVersion := strings.TrimPrefix(versions["clientVersion"].GitVersion, "v")
	serverVersion := strings.TrimPrefix(versions["serverVersion"].GitVersion, "v")

	cmd := cmdTestCase{
		args:            "check --pre",
		testClusterMode: ExistingClusterMode,
		assert: assertGoldenTemplateFile("testdata/check/check_pre.golden", map[string]string{
			"clientVersion": clientVersion,
			"serverVersion": serverVersion,
		}),
	}
	cmd.runTestCmd(t)
}
