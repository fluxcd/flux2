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
	"encoding/json"
	"strings"
	"testing"

	"github.com/fluxcd/flux2/internal/utils"
	"k8s.io/apimachinery/pkg/version"
)

func TestCheckPre(t *testing.T) {
	jsonOutput, err := utils.ExecKubectlCommand(context.TODO(), utils.ModeCapture, *kubeconfigArgs.KubeConfig, *kubeconfigArgs.Context, "version", "--output", "json")
	if err != nil {
		t.Fatalf("Error running utils.ExecKubectlCommand: %v", err.Error())
	}

	var versions map[string]version.Info
	if err := json.Unmarshal([]byte(jsonOutput), &versions); err != nil {
		t.Fatalf("Error unmarshalling: %v", err.Error())
	}

	serverVersion := strings.TrimPrefix(versions["serverVersion"].GitVersion, "v")

	cmd := cmdTestCase{
		args: "check --pre",
		assert: assertGoldenTemplateFile("testdata/check/check_pre.golden", map[string]string{
			"serverVersion": serverVersion,
		}),
	}
	cmd.runTestCmd(t)
}
