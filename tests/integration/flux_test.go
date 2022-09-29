/*
Copyright 2023 The Flux authors

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

package integration

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/fluxcd/pkg/git"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestFluxInstallation(t *testing.T) {
	g := NewWithT(t)
	ctx := context.TODO()
	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, testEnv.Client, "flux-system", "flux-system")
		if err != nil {
			return false
		}
		return true
	}, 60*time.Second, 5*time.Second)
}

func TestRepositoryCloning(t *testing.T) {
	ctx := context.TODO()
	branchName := "feature/branch"
	tagName := "v1"

	g := NewWithT(t)

	type testStruct struct {
		name      string
		refType   string
		cloneType git.TransportType
	}

	tests := []testStruct{
		{
			name:      "ssh-feature-branch",
			refType:   "branch",
			cloneType: git.SSH,
		},
		{
			name:      "ssh-v1",
			refType:   "tag",
			cloneType: git.SSH,
		},
	}

	// Not all cloud providers have repositories that support authentication with an accessToken
	// we don't run http tests for these.
	if cfg.gitPat != "" {
		httpTests := []testStruct{
			{
				name:      "https-feature-branch",
				refType:   "branch",
				cloneType: git.HTTP,
			},
			{
				name:      "https-v1",
				refType:   "tag",
				cloneType: git.HTTP,
			},
		}

		tests = append(tests, httpTests...)
	}

	t.Log("Creating application sources")
	url := getTransportURL(cfg.applicationRepository)
	tmpDir := t.TempDir()
	client, err := getRepository(ctx, tmpDir, url, defaultBranch, cfg.defaultAuthOpts)
	g.Expect(err).ToNot(HaveOccurred())

	files := make(map[string]io.Reader)
	for _, tt := range tests {
		manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: foobar
    `
		name := fmt.Sprintf("cloning-test/%s/configmap.yaml", tt.name)
		files[name] = strings.NewReader(manifest)
	}

	err = commitAndPushAll(ctx, client, files, branchName)
	g.Expect(err).ToNot(HaveOccurred())
	err = createTagAndPush(ctx, client, branchName, tagName)
	g.Expect(err).ToNot(HaveOccurred())

	t.Log("Verifying application-gitops namespaces")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			ref := &sourcev1.GitRepositoryRef{
				Branch: branchName,
			}
			if tt.refType == "tag" {
				ref = &sourcev1.GitRepositoryRef{
					Tag: tagName,
				}
			}

			url := cfg.applicationRepository.http
			if tt.cloneType == git.SSH {
				url = cfg.applicationRepository.ssh
			}

			testID := fmt.Sprintf("%s-%s", tt.name, randStringRunes(5))
			err := setUpFluxConfig(ctx, testID, nsConfig{
				repoURL:    url,
				ref:        ref,
				protocol:   tt.cloneType,
				objectName: testID,
				path:       fmt.Sprintf("./cloning-test/%s", tt.name),
			})
			g.Expect(err).ToNot(HaveOccurred())
			t.Cleanup(func() {
				err := tearDownFluxConfig(ctx, testID)
				if err != nil {
					t.Logf("failed to delete resources in '%s' namespace: %s", tt.name, err)
				}
			})

			g.Eventually(func() bool {
				err := verifyGitAndKustomization(ctx, testEnv.Client, testID, testID)
				if err != nil {
					return false
				}
				return true
			}, 120*time.Second, 5*time.Second).Should(BeTrue())

			// Wait for configmap to be deployed
			g.Eventually(func() bool {
				nn := types.NamespacedName{Name: "foobar", Namespace: testID}
				cm := &corev1.ConfigMap{}
				err = testEnv.Get(ctx, nn, cm)
				if err != nil {
					return false
				}

				return true
			}, 120*time.Second, 5*time.Second).Should(BeTrue())
		})
	}
}
