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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/test-infra/tftestenv"
)

func TestKeyVaultSops(t *testing.T) {
	g := NewWithT(t)
	ctx := context.TODO()
	name := "key-vault-sops"
	repoUrl := cfg.applicationRepository.http
	secretYaml, err := executeTemplate("testdata/secret.yaml", map[string]string{
		"ns": name,
	})

	c, err := cloneRepository(repoUrl, name, cfg.pat)
	tmpDir := c.Path()
	err = tftestenv.RunCommand(ctx, tmpDir, "mkdir -p ./key-vault-sops", tftestenv.RunCommandOptions{})
	g.Expect(err).ToNot(HaveOccurred())
	err = tftestenv.RunCommand(ctx, tmpDir, fmt.Sprintf("echo \"%s\" > ./key-vault-sops/secret.enc.yaml", secretYaml), tftestenv.RunCommandOptions{})
	g.Expect(err).ToNot(HaveOccurred())
	err = tftestenv.RunCommand(ctx, tmpDir, fmt.Sprintf("sops --encrypt --encrypted-regex '^(data|stringData)$' %s --in-place ./key-vault-sops/secret.enc.yaml", cfg.sopsArgs), tftestenv.RunCommandOptions{})
	g.Expect(err).ToNot(HaveOccurred())
	err = commitAndPushAll(ctx, c, name, "Add sops files")
	g.Expect(err).ToNot(HaveOccurred())

	if cfg.sopsSecretData != nil {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sops-keys",
				Namespace: name,
			},
		}

		_, err = controllerutil.CreateOrUpdate(ctx, testEnv.Client, &secret, func() error {
			secret.StringData = cfg.sopsSecretData
			return nil
		})

		g.Expect(err).ToNot(HaveOccurred())
	}

	modifyKsSpec := func(spec *kustomizev1.KustomizationSpec) {
		spec.Decryption = &kustomizev1.Decryption{
			Provider: "sops",
		}
		if cfg.sopsSecretData != nil {
			spec.Decryption.SecretRef = &meta.LocalObjectReference{
				Name: "sops-keys",
			}
		}
	}
	err = setupNamespace(ctx, name, nsConfig{
		repoURL:      repoUrl,
		path:         "./key-vault-sops",
		modifyKsSpec: modifyKsSpec,
	})
	g.Expect(err).ToNot(HaveOccurred())

	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, testEnv.Client, name, name)
		if err != nil {
			return false
		}
		nn := types.NamespacedName{Name: "test", Namespace: name}
		secret := &corev1.Secret{}
		err = testEnv.Client.Get(ctx, nn, secret)
		if err != nil {
			return false
		}

		if string(secret.Data["foo"]) == "bar" {
			return true
		}

		return false
	}, 120*time.Second, 5*time.Second)
}
