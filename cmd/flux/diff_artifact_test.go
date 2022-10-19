//go:build unit
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
	"testing"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	_ "github.com/distribution/distribution/v3/registry/auth/htpasswd"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"github.com/phayes/freeport"
	ctrl "sigs.k8s.io/controller-runtime"
)

var dockerReg string

func setupRegistryServer(ctx context.Context) error {
	// Registry config
	config := &configuration.Configuration{}
	port, err := freeport.GetFreePort()
	if err != nil {
		return fmt.Errorf("failed to get free port: %s", err)
	}

	dockerReg = fmt.Sprintf("localhost:%d", port)
	config.HTTP.Addr = fmt.Sprintf("127.0.0.1:%d", port)
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}
	dockerRegistry, err := registry.NewRegistry(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create docker registry: %w", err)
	}

	// Start Docker registry
	go dockerRegistry.ListenAndServe()

	return nil
}

func TestDiffArtifact(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		argsTpl  string
		pushFile string
		diffFile string
		assert   assertFunc
	}{
		{
			name:     "should not fail if there is no diff",
			url:      "oci://%s/podinfo:1.0.0",
			argsTpl:  "diff artifact  %s --path=%s",
			pushFile: "./testdata/diff-artifact/deployment.yaml",
			diffFile: "./testdata/diff-artifact/deployment.yaml",
			assert:   assertGoldenFile("testdata/diff-artifact/success.golden"),
		},
		{
			name:     "should fail if there is a diff",
			url:      "oci://%s/podinfo:2.0.0",
			argsTpl:  "diff artifact %s --path=%s",
			pushFile: "./testdata/diff-artifact/deployment.yaml",
			diffFile: "./testdata/diff-artifact/deployment-diff.yaml",
			assert:   assertError("the remote artifact contents differs from the local one"),
		},
	}

	ctx := ctrl.SetupSignalHandler()
	err := setupRegistryServer(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to start docker registry: %s", err))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.url = fmt.Sprintf(tt.url, dockerReg)
			_, err := executeCommand("push artifact " + tt.url + " --path=" + tt.pushFile + " --source=test --revision=test")
			if err != nil {
				t.Fatalf(fmt.Errorf("failed to push image: %w", err).Error())
			}

			cmd := cmdTestCase{
				args:   fmt.Sprintf(tt.argsTpl, tt.url, tt.diffFile),
				assert: tt.assert,
			}
			cmd.runTestCmd(t)
		})
	}
}
