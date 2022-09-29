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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	automationv1beta1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	reflectorv1beta2 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

func TestImageRepositoryAndAutomation(t *testing.T) {
	g := NewWithT(t)
	ctx := context.TODO()
	branchName := "image-repository"
	testID := branchName + "-" + randStringRunes(5)
	imageURL := fmt.Sprintf("%s/podinfo", cfg.testRegistry)

	manifest := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: podinfo
  namespace: %[1]s
spec:
  selector:
    matchLabels:
      app: podinfo
  template:
    metadata:
      labels:
        app: podinfo
    spec:
      containers:
      - name: podinfod
        image: %[2]s:%[3]s # {"$imagepolicy": "%[1]s:podinfo"}
        readinessProbe:
          exec:
            command:
            - podcli
            - check
            - http
            - localhost:9898/readyz
          initialDelaySeconds: 5
          timeoutSeconds: 5
`, testID, imageURL, oldPodinfoVersion)

	repoUrl := getTransportURL(cfg.applicationRepository)
	client, err := getRepository(ctx, t.TempDir(), repoUrl, defaultBranch, cfg.defaultAuthOpts)
	g.Expect(err).ToNot(HaveOccurred())
	files := make(map[string]io.Reader)
	files[testID+"/podinfo.yaml"] = strings.NewReader(manifest)

	err = commitAndPushAll(ctx, client, files, branchName)
	g.Expect(err).ToNot(HaveOccurred())

	err = setUpFluxConfig(ctx, testID, nsConfig{
		repoURL: repoUrl,
		path:    testID,
		ref: &sourcev1.GitRepositoryRef{
			Branch: branchName,
		},
	})
	g.Expect(err).ToNot(HaveOccurred())
	t.Cleanup(func() {
		err := tearDownFluxConfig(ctx, testID)
		if err != nil {
			t.Logf("failed to delete resources in '%s' namespace: %s", testID, err)
		}
	})

	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, testEnv.Client, testID, testID)
		if err != nil {
			return false
		}
		return true
	}, testTimeout, testInterval).Should(BeTrue())

	imageRepository := reflectorv1beta2.ImageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podinfo",
			Namespace: testID,
		},
		Spec: reflectorv1beta2.ImageRepositorySpec{
			Image: imageURL,
			Interval: metav1.Duration{
				Duration: 1 * time.Minute,
			},
			Provider: infraOpts.Provider,
		},
	}
	g.Expect(testEnv.Create(ctx, &imageRepository)).To(Succeed())
	defer testEnv.Delete(ctx, &imageRepository)

	imagePolicy := reflectorv1beta2.ImagePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podinfo",
			Namespace: testID,
		},
		Spec: reflectorv1beta2.ImagePolicySpec{
			ImageRepositoryRef: meta.NamespacedObjectReference{
				Name: imageRepository.Name,
			},
			Policy: reflectorv1beta2.ImagePolicyChoice{
				SemVer: &reflectorv1beta2.SemVerPolicy{
					Range: "6.0.x",
				},
			},
		},
	}
	g.Expect(testEnv.Create(ctx, &imagePolicy)).To(Succeed())
	defer testEnv.Delete(ctx, &imagePolicy)

	imageAutomation := automationv1beta1.ImageUpdateAutomation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podinfo",
			Namespace: testID,
		},
		Spec: automationv1beta1.ImageUpdateAutomationSpec{
			Interval: metav1.Duration{
				Duration: 1 * time.Minute,
			},
			SourceRef: automationv1beta1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
				Name: testID,
			},
			GitSpec: &automationv1beta1.GitSpec{
				Checkout: &automationv1beta1.GitCheckoutSpec{
					Reference: sourcev1.GitRepositoryRef{
						Branch: branchName,
					},
				},
				Commit: automationv1beta1.CommitSpec{
					Author: automationv1beta1.CommitUser{
						Email: "imageautomation@example.com",
						Name:  "imageautomation",
					},
				},
			},
			Update: &automationv1beta1.UpdateStrategy{
				Path:     testID,
				Strategy: automationv1beta1.UpdateStrategySetters,
			},
		},
	}
	g.Expect(testEnv.Create(ctx, &imageAutomation)).To(Succeed())
	defer testEnv.Delete(ctx, &imageAutomation)

	// Wait for image repository to be ready
	g.Eventually(func() bool {
		client, err := getRepository(ctx, t.TempDir(), repoUrl, branchName, cfg.defaultAuthOpts)
		if err != nil {
			return false
		}

		b, err := os.ReadFile(filepath.Join(client.Path(), testID, "podinfo.yaml"))
		if err != nil {
			return false
		}
		if bytes.Contains(b, []byte(newPodinfoVersion)) == false {
			return false
		}
		return true
	}, testTimeout, testInterval).Should(BeTrue())
}
