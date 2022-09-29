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
	"bytes"
	"context"
	b64 "encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	automationv1beta1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	reflectorv1beta1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

func TestImageRepositoryAndAutomation(t *testing.T) {
	g := NewWithT(t)
	ctx := context.TODO()
	name := "image-repository-acr"
	repoUrl := cfg.applicationRepository.http
	oldVersion := "1.0.0"
	newVersion := "1.0.1"

	imageURL := fmt.Sprintf("%s/container/podinfo", cfg.dockerCred.url)
	// push the podinfo image to the container registry
	err := pushImagesFromURL(imageURL, "ghcr.io/stefanprodan/podinfo", []string{oldVersion, newVersion})
	g.Expect(err).ToNot(HaveOccurred())

	tmpl := map[string]string{
		"ns":      name,
		"name":    name,
		"version": oldVersion,
		"url":     cfg.dockerCred.url,
	}
	manifest, err := executeTemplate("testdata/automation-deploy.yaml", tmpl)

	c, err := cloneRepository(repoUrl, name, cfg.pat)
	g.Expect(err).ToNot(HaveOccurred())
	repoDir := c.Path()
	err = addFile(repoDir, "podinfo.yaml", manifest)
	g.Expect(err).ToNot(HaveOccurred())
	err = commitAndPushAll(ctx, c, name, cfg.pat)
	g.Expect(err).ToNot(HaveOccurred())

	err = setupNamespace(ctx, name, nsConfig{
		repoURL: repoUrl,
		path:    "./",
	})
	g.Expect(err).ToNot(HaveOccurred())

	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, testEnv.Client, name, name)
		if err != nil {
			return false
		}
		return true
	}, 60*time.Second, 5*time.Second)

	acrSecret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "acr-docker", Namespace: name}}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &acrSecret, func() error {
		acrSecret.Type = corev1.SecretTypeDockerConfigJson
		acrSecret.StringData = map[string]string{
			".dockerconfigjson": fmt.Sprintf(`
        {
          "auths": {
            "%s": {
              "auth": "%s"
            }
          }
        }
        `, cfg.dockerCred.url, b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.dockerCred.username, cfg.dockerCred.password)))),
		}
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())

	imageRepository := reflectorv1beta1.ImageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podinfo",
			Namespace: name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &imageRepository, func() error {
		imageRepository.Spec = reflectorv1beta1.ImageRepositorySpec{
			Image: fmt.Sprintf("%s/container/podinfo", cfg.dockerCred.url),
			Interval: metav1.Duration{
				Duration: 1 * time.Minute,
			},
		}
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())

	imagePolicy := reflectorv1beta1.ImagePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podinfo",
			Namespace: name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &imagePolicy, func() error {
		imagePolicy.Spec = reflectorv1beta1.ImagePolicySpec{
			ImageRepositoryRef: meta.NamespacedObjectReference{
				Name: imageRepository.Name,
			},
			Policy: reflectorv1beta1.ImagePolicyChoice{
				SemVer: &reflectorv1beta1.SemVerPolicy{
					Range: "1.0.x",
				},
			},
		}
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())

	imageAutomation := automationv1beta1.ImageUpdateAutomation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podinfo",
			Namespace: name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &imageAutomation, func() error {
		imageAutomation.Spec = automationv1beta1.ImageUpdateAutomationSpec{
			Interval: metav1.Duration{
				Duration: 1 * time.Minute,
			},
			SourceRef: automationv1beta1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
				Name: name,
			},
			GitSpec: &automationv1beta1.GitSpec{
				Checkout: &automationv1beta1.GitCheckoutSpec{
					Reference: sourcev1.GitRepositoryRef{
						Branch: name,
					},
				},
				Commit: automationv1beta1.CommitSpec{
					Author: automationv1beta1.CommitUser{
						Email: "imageautomation@example.com",
						Name:  "imageautomation",
					},
				},
			},
		}
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Wait for image repository to be ready
	g.Eventually(func() bool {
		c, err := cloneRepository(repoUrl, name, cfg.pat)
		if err != nil {
			return false
		}

		repoDir := c.Path()
		b, err := os.ReadFile(filepath.Join(repoDir, "podinfo.yaml"))
		if err != nil {
			return false
		}
		if bytes.Contains(b, []byte(newVersion)) == false {
			return false
		}
		return true
	}, 120*time.Second, 5*time.Second)
}
