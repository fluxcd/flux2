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
	"os"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/fluxcd/test-infra/tftestenv"
)

func TestKeyVaultSops(t *testing.T) {
	g := NewWithT(t)
	ctx := context.TODO()
	branchName := "key-vault"
	testID := branchName + "-" + randStringRunes(5)
	secretYaml := `apiVersion: v1
kind: Secret
metadata:
  name: "test"
stringData:
  foo: "bar"`

	repoUrl := getTransportURL(cfg.applicationRepository)
	tmpDir := t.TempDir()
	client, err := getRepository(ctx, tmpDir, repoUrl, defaultBranch, cfg.defaultAuthOpts)
	g.Expect(err).ToNot(HaveOccurred())

	dir := client.Path() + "/key-vault-sops"
	g.Expect(os.Mkdir(dir, 0o700)).To(Succeed())

	filename := dir + "secret.enc.yaml"
	f, err := os.Create(filename)
	g.Expect(err).ToNot(HaveOccurred())
	defer f.Close()

	_, err = f.Write([]byte(secretYaml))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(f.Sync()).To(Succeed())

	err = tftestenv.RunCommand(ctx, client.Path(),
		fmt.Sprintf("sops --encrypt --encrypted-regex '^(data|stringData)$' %s --in-place %s", cfg.sopsArgs, filename),
		tftestenv.RunCommandOptions{})
	g.Expect(err).ToNot(HaveOccurred())

	r, err := os.Open(filename)
	g.Expect(err).ToNot(HaveOccurred())

	files := make(map[string]io.Reader)
	files["key-vault-sops/secret.enc.yaml"] = r
	err = commitAndPushAll(ctx, client, files, branchName)
	g.Expect(err).ToNot(HaveOccurred())

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

	err = setUpFluxConfig(ctx, testID, nsConfig{
		ref: &sourcev1.GitRepositoryRef{
			Branch: branchName,
		},
		repoURL:      repoUrl,
		path:         "./key-vault-sops",
		modifyKsSpec: modifyKsSpec,
		protocol:     cfg.defaultGitTransport,
	})
	g.Expect(err).ToNot(HaveOccurred())
	t.Cleanup(func() {
		err := tearDownFluxConfig(ctx, testID)
		if err != nil {
			log.Printf("failed to delete resources in '%s' namespace", testID)
		}
	})

	if cfg.sopsSecretData != nil {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sops-keys",
				Namespace: testID,
			},
			StringData: cfg.sopsSecretData,
		}
		g.Expect(testEnv.Create(ctx, &secret)).To(Succeed())
		defer testEnv.Delete(ctx, &secret)
	}

	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, testEnv.Client, testID, testID)
		if err != nil {
			return false
		}
		nn := types.NamespacedName{Name: "test", Namespace: testID}
		secret := &corev1.Secret{}
		err = testEnv.Get(ctx, nn, secret)
		if err != nil {
			return false
		}

		if string(secret.Data["foo"]) == "bar" {
			return true
		}

		return false
	}, testTimeout, testInterval).Should(BeTrue())
}
