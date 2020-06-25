/*
Copyright 2020 The Flux CD contributors.

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
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap toolkit components",
	Long:  "The bootstrap sub-commands bootstrap the toolkit components on the targeted Git provider.",
}

var (
	bootstrapVersion string
)

const (
	bootstrapBranch                = "master"
	bootstrapInstallManifest       = "toolkit-components.yaml"
	bootstrapSourceManifest        = "toolkit-source.yaml"
	bootstrapKustomizationManifest = "toolkit-kustomization.yaml"
)

func init() {
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapVersion, "version", "master", "toolkit tag or branch")

	rootCmd.AddCommand(bootstrapCmd)
}

func generateInstallManifests(targetPath, namespace, tmpDir string) (string, error) {
	tkDir := path.Join(tmpDir, ".tk")
	defer os.RemoveAll(tkDir)

	if err := os.MkdirAll(tkDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("generating manifests failed: %w", err)
	}

	if err := genInstallManifests(bootstrapVersion, namespace, components, tkDir); err != nil {
		return "", fmt.Errorf("generating manifests failed: %w", err)
	}

	manifestsDir := path.Join(tmpDir, targetPath, namespace)
	if err := os.MkdirAll(manifestsDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("generating manifests failed: %w", err)
	}

	manifest := path.Join(manifestsDir, bootstrapInstallManifest)
	if err := buildKustomization(tkDir, manifest); err != nil {
		return "", fmt.Errorf("build kustomization failed: %w", err)
	}

	return manifest, nil
}

func applyInstallManifests(ctx context.Context, manifestPath string, components []string) error {
	command := fmt.Sprintf("kubectl apply -f %s", manifestPath)
	if _, err := utils.execCommand(ctx, ModeOS, command); err != nil {
		return fmt.Errorf("install failed")
	}

	for _, deployment := range components {
		command = fmt.Sprintf("kubectl -n %s rollout status deployment %s --timeout=%s",
			namespace, deployment, timeout.String())
		if _, err := utils.execCommand(ctx, ModeOS, command); err != nil {
			return fmt.Errorf("install failed")
		}
	}
	return nil
}

func generateSyncManifests(url, name, namespace, targetPath, tmpDir string, interval time.Duration) error {
	gvk := sourcev1.GroupVersion.WithKind("GitRepository")
	gitRepository := sourcev1.GitRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: sourcev1.GitRepositorySpec{
			URL: url,
			Interval: metav1.Duration{
				Duration: interval,
			},
			Reference: &sourcev1.GitRepositoryRef{
				Branch: "master",
			},
			SecretRef: &corev1.LocalObjectReference{
				Name: name,
			},
		},
	}

	gitData, err := yaml.Marshal(gitRepository)
	if err != nil {
		return err
	}

	if err := utils.writeFile(string(gitData), filepath.Join(tmpDir, targetPath, namespace, bootstrapSourceManifest)); err != nil {
		return err
	}

	gvk = kustomizev1.GroupVersion.WithKind("Kustomization")
	emptyAPIGroup := ""
	kustomization := kustomizev1.Kustomization{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kustomizev1.KustomizationSpec{
			Interval: metav1.Duration{
				Duration: 10 * time.Minute,
			},
			Path:  fmt.Sprintf("./%s", strings.TrimPrefix(targetPath, "./")),
			Prune: true,
			SourceRef: corev1.TypedLocalObjectReference{
				APIGroup: &emptyAPIGroup,
				Kind:     "GitRepository",
				Name:     name,
			},
		},
	}

	ksData, err := yaml.Marshal(kustomization)
	if err != nil {
		return err
	}

	if err := utils.writeFile(string(ksData), filepath.Join(tmpDir, targetPath, namespace, bootstrapKustomizationManifest)); err != nil {
		return err
	}

	return nil
}

func applySyncManifests(ctx context.Context, kubeClient client.Client, name, namespace, targetPath, tmpDir string) error {
	command := fmt.Sprintf("kubectl apply -f %s", filepath.Join(tmpDir, targetPath, namespace))
	if _, err := utils.execCommand(ctx, ModeStderrOS, command); err != nil {
		return err
	}

	logger.Waitingf("waiting for cluster sync")

	if err := wait.PollImmediate(pollInterval, timeout,
		isGitRepositoryReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	if err := wait.PollImmediate(pollInterval, timeout,
		isKustomizationReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	return nil
}

func shouldInstallManifests(ctx context.Context, kubeClient client.Client, namespace string) bool {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      namespace,
	}
	var kustomization kustomizev1.Kustomization
	if err := kubeClient.Get(ctx, namespacedName, &kustomization); err != nil {
		return true
	}

	return kustomization.Status.LastAppliedRevision == ""
}

func shouldCreateDeployKey(ctx context.Context, kubeClient client.Client, namespace string) bool {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      namespace,
	}

	var existing corev1.Secret
	if err := kubeClient.Get(ctx, namespacedName, &existing); err != nil {
		return true
	}
	return false
}

func generateDeployKey(ctx context.Context, kubeClient client.Client, url *url.URL, namespace string) (string, error) {
	pair, err := generateKeyPair(ctx)
	if err != nil {
		return "", err
	}

	hostKey, err := scanHostKey(ctx, url)
	if err != nil {
		return "", err
	}

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"identity":     string(pair.PrivateKey),
			"identity.pub": string(pair.PublicKey),
			"known_hosts":  string(hostKey),
		},
	}
	if err := upsertSecret(ctx, kubeClient, secret); err != nil {
		return "", err
	}

	return string(pair.PublicKey), nil
}
