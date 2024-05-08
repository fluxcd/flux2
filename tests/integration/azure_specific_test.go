//go:build azure
// +build azure

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
	"log"
	"strings"
	"testing"
	"time"

	giturls "github.com/chainguard-dev/git-urls"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notiv1 "github.com/fluxcd/notification-controller/api/v1"
	notiv1beta3 "github.com/fluxcd/notification-controller/api/v1beta3"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

func TestAzureDevOpsCommitStatus(t *testing.T) {
	g := NewWithT(t)

	ctx := context.TODO()
	branchName := "commit-status"
	testID := branchName + randStringRunes(5)
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: foobar`

	repoUrl := getTransportURL(cfg.applicationRepository)
	tmpDir := t.TempDir()
	c, err := getRepository(ctx, tmpDir, repoUrl, defaultBranch, cfg.defaultAuthOpts)
	g.Expect(err).ToNot(HaveOccurred())
	files := make(map[string]io.Reader)
	files["configmap.yaml"] = strings.NewReader(manifest)
	err = commitAndPushAll(ctx, c, files, branchName)
	g.Expect(err).ToNot(HaveOccurred())

	modifyKsSpec := func(spec *kustomizev1.KustomizationSpec) {
		spec.HealthChecks = []meta.NamespacedObjectKindReference{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "foobar",
				Namespace:  testID,
			},
		}
	}
	err = setUpFluxConfig(ctx, testID, nsConfig{
		ref: &sourcev1.GitRepositoryRef{
			Branch: branchName,
		},
		repoURL:      repoUrl,
		path:         "./",
		modifyKsSpec: modifyKsSpec,
	})
	g.Expect(err).ToNot(HaveOccurred())
	t.Cleanup(func() {
		err := tearDownFluxConfig(ctx, testID)
		if err != nil {
			log.Printf("failed to delete resources in '%s' namespace: %s", testID, err)
		}
	})

	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, testEnv, testID, testID)
		if err != nil {
			return false
		}
		return true
	}, testTimeout, testInterval)

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azuredevops-token",
			Namespace: testID,
		},
		StringData: map[string]string{
			"token": cfg.gitPat,
		},
	}
	g.Expect(testEnv.Create(ctx, &secret)).To(Succeed())
	defer testEnv.Delete(ctx, &secret)

	provider := notiv1beta3.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azuredevops",
			Namespace: testID,
		},
		Spec: notiv1beta3.ProviderSpec{
			Type:    "azuredevops",
			Address: repoUrl,
			SecretRef: &meta.LocalObjectReference{
				Name: "azuredevops-token",
			},
		},
	}
	g.Expect(testEnv.Create(ctx, &provider)).To(Succeed())
	defer testEnv.Delete(ctx, &provider)

	alert := notiv1beta3.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azuredevops",
			Namespace: testID,
		},
		Spec: notiv1beta3.AlertSpec{
			ProviderRef: meta.LocalObjectReference{
				Name: provider.Name,
			},
			EventSources: []notiv1.CrossNamespaceObjectReference{
				{
					Kind:      "Kustomization",
					Name:      testID,
					Namespace: testID,
				},
			},
		},
	}
	g.Expect(testEnv.Create(ctx, &alert)).To(Succeed())
	defer testEnv.Delete(ctx, &alert)

	url, err := ParseAzureDevopsURL(repoUrl)
	g.Expect(err).ToNot(HaveOccurred())

	rev, err := c.Head()
	g.Expect(err).ToNot(HaveOccurred())

	connection := azuredevops.NewPatConnection(url.OrgURL, cfg.gitPat)
	client, err := git.NewClient(ctx, connection)
	g.Expect(err).ToNot(HaveOccurred())
	getArgs := git.GetStatusesArgs{
		Project:      &url.Project,
		RepositoryId: &url.Repo,
		CommitId:     &rev,
	}
	g.Eventually(func() bool {
		statuses, err := client.GetStatuses(ctx, getArgs)
		if err != nil {
			return false
		}
		if len(*statuses) != 1 {
			return false
		}
		return true
	}, 500*time.Second, 5*time.Second)
}

type AzureDevOpsURL struct {
	OrgURL  string
	Project string
	Repo    string
}

// TODO(somtochiama): move this into fluxcd/pkg and reuse in NC
func ParseAzureDevopsURL(s string) (AzureDevOpsURL, error) {
	var args AzureDevOpsURL
	u, err := giturls.Parse(s)
	if err != nil {
		return args, nil
	}

	scheme := u.Scheme
	if u.Scheme == "ssh" {
		scheme = "https"
	}

	id := strings.TrimLeft(u.Path, "/")
	id = strings.TrimSuffix(id, ".git")

	comp := strings.Split(id, "/")
	if len(comp) != 4 {
		return args, fmt.Errorf("invalid repository id %q", id)
	}

	args = AzureDevOpsURL{
		OrgURL:  fmt.Sprintf("%s://%s/%s", scheme, u.Host, comp[0]),
		Project: comp[1],
		Repo:    comp[3],
	}

	return args, nil

}
