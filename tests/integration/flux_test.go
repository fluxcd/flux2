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

package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/fluxcd/test-infra/tftestenv"
)

func TestFluxInstallation(t *testing.T) {
	g := NewWithT(t)
	ctx := context.TODO()
	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, cfg.client, "flux-system", "flux-system")
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

	tests := []struct {
		name      string
		refType   string
		cloneType string
	}{
		{
			name:      "https-feature-branch",
			refType:   "branch",
			cloneType: "http",
		},
		{
			name:      "https-v1",
			refType:   "tag",
			cloneType: "http",
		},
		{
			name:      "ssh-feature-branch",
			refType:   "branch",
			cloneType: "ssh",
		},
		{
			name:      "ssh-v1",
			refType:   "tag",
			cloneType: "ssh",
		},
	}

	t.Log("Creating application sources")
	repo, err := cloneRepository(cfg.applicationRepository.http, branchName, cfg.pat)
	g.Expect(err).ToNot(HaveOccurred())

	repoDir := repo.Path()

	for _, tt := range tests {
		manifest, err := executeTemplate("testdata/cm.yaml", map[string]string{
			"ns": tt.name,
		})
		g.Expect(err).ToNot(HaveOccurred())
		err = tftestenv.RunCommand(ctx, repoDir, fmt.Sprintf("mkdir -p ./cloning-test/%s", tt.name), tftestenv.RunCommandOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		err = tftestenv.RunCommand(ctx, repoDir, fmt.Sprintf("echo '%s' > ./cloning-test/%s/configmap.yaml", manifest, tt.name), tftestenv.RunCommandOptions{})
		g.Expect(err).ToNot(HaveOccurred())
	}
	err = commitAndPushAll(ctx, repo, branchName, cfg.pat)
	g.Expect(err).ToNot(HaveOccurred())
	err = createTagAndPush(repo, branchName, tagName, cfg.pat)
	g.Expect(err).ToNot(HaveOccurred())

	t.Log("Verifying application-gitops namespaces")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := &sourcev1.GitRepositoryRef{
				Branch: branchName,
			}
			if tt.refType == "tag" {
				ref = &sourcev1.GitRepositoryRef{
					Tag: tagName,
				}
			}

			url := cfg.applicationRepository.http
			if tt.cloneType == "ssh" {
				url = cfg.applicationRepository.ssh
			}

			err := setupNamespace(ctx, tt.name, nsConfig{
				repoURL:    url,
				protocol:   tt.cloneType,
				objectName: tt.name,
				path:       fmt.Sprintf("./cloning-test/%s", tt.name),
				modifyGitSpec: func(spec *sourcev1.GitRepositorySpec) {
					spec.Reference = ref
				},
			})

			g.Expect(err).ToNot(HaveOccurred())

			// Wait for configmap to be deployed
			g.Eventually(func() bool {
				err := verifyGitAndKustomization(ctx, cfg.client, tt.name, tt.name)
				if err != nil {
					return false
				}
				nn := types.NamespacedName{Name: "foobar", Namespace: tt.name}
				cm := &corev1.ConfigMap{}
				err = cfg.client.Get(ctx, nn, cm)
				if err != nil {
					return false
				}
				return true
			}, 120*time.Second, 5*time.Second)
		})
	}
}
