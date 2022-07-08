/*
Copyright 2022 The Flux authors

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

package oci

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	_ "github.com/distribution/distribution/v3/registry/auth/htpasswd"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"github.com/phayes/freeport"
	"golang.org/x/crypto/bcrypt"
)

const (
	testRegistryHtpasswdFileBasename = "authtest.htpasswd"
	testRegistryUsername             = "myuser"
	testRegistryPassword             = "mypass"
)

var (
	dockerReg string
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func setupRegistryServer(ctx context.Context) error {
	// Create a temporary workspace directory for the registry
	workspaceDir, err := os.MkdirTemp("", "registry-test-")
	if err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// create htpasswd file (w BCrypt, which is required)
	pwBytes, err := bcrypt.GenerateFromPassword([]byte(testRegistryPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to generate password: %s", err)
	}

	htpasswdPath := filepath.Join(workspaceDir, testRegistryHtpasswdFileBasename)
	err = ioutil.WriteFile(htpasswdPath, []byte(fmt.Sprintf("%s:%s\n", testRegistryUsername, string(pwBytes))), 0644)
	if err != nil {
		return fmt.Errorf("failed to create htpasswd file: %s", err)
	}

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

func TestMain(m *testing.M) {
	ctx := ctrl.SetupSignalHandler()

	err := setupRegistryServer(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to start docker registry: %s", err))
	}

	code := m.Run()

	os.Exit(code)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
