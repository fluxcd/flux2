/*
Copyright 2020 The Flux authors

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
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"

	"github.com/fluxcd/flux2/internal/flags"
	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/pkg/manifestgen/sync"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap toolkit components",
	Long:  "The bootstrap sub-commands bootstrap the toolkit components on the targeted Git provider.",
}

var (
	bootstrapVersion            string
	bootstrapDefaultComponents  []string
	bootstrapExtraComponents    []string
	bootstrapRegistry           string
	bootstrapImagePullSecret    string
	bootstrapBranch             string
	bootstrapWatchAllNamespaces bool
	bootstrapNetworkPolicy      bool
	bootstrapManifestsPath      string
	bootstrapArch               = flags.Arch(defaults.Arch)
	bootstrapLogLevel           = flags.LogLevel(defaults.LogLevel)
	bootstrapRequiredComponents = []string{"source-controller", "kustomize-controller"}
	bootstrapTokenAuth          bool
	bootstrapClusterDomain      string
)

const (
	bootstrapDefaultBranch = "main"
)

func init() {
	bootstrapCmd.PersistentFlags().StringVarP(&bootstrapVersion, "version", "v", defaults.Version,
		"toolkit version")
	bootstrapCmd.PersistentFlags().StringSliceVar(&bootstrapDefaultComponents, "components", defaults.Components,
		"list of components, accepts comma-separated values")
	bootstrapCmd.PersistentFlags().StringSliceVar(&bootstrapExtraComponents, "components-extra", nil,
		"list of components in addition to those supplied or defaulted, accepts comma-separated values")
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapRegistry, "registry", "ghcr.io/fluxcd",
		"container registry where the toolkit images are published")
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapImagePullSecret, "image-pull-secret", "",
		"Kubernetes secret name used for pulling the toolkit images from a private registry")
	bootstrapCmd.PersistentFlags().Var(&bootstrapArch, "arch", bootstrapArch.Description())
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapBranch, "branch", bootstrapDefaultBranch,
		"default branch (for GitHub this must match the default branch setting for the organization)")
	bootstrapCmd.PersistentFlags().BoolVar(&bootstrapWatchAllNamespaces, "watch-all-namespaces", true,
		"watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed")
	bootstrapCmd.PersistentFlags().BoolVar(&bootstrapNetworkPolicy, "network-policy", true,
		"deny ingress access to the toolkit controllers from other namespaces using network policies")
	bootstrapCmd.PersistentFlags().BoolVar(&bootstrapTokenAuth, "token-auth", false,
		"when enabled, the personal access token will be used instead of SSH deploy key")
	bootstrapCmd.PersistentFlags().Var(&bootstrapLogLevel, "log-level", bootstrapLogLevel.Description())
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapManifestsPath, "manifests", "", "path to the manifest directory")
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapClusterDomain, "cluster-domain", "cluster.local", "internal cluster domain")
	bootstrapCmd.PersistentFlags().MarkHidden("manifests")
	rootCmd.AddCommand(bootstrapCmd)
}

func bootstrapComponents() []string {
	return append(bootstrapDefaultComponents, bootstrapExtraComponents...)
}

func bootstrapValidate() error {
	components := bootstrapComponents()
	for _, component := range bootstrapRequiredComponents {
		if !utils.ContainsItemString(components, component) {
			return fmt.Errorf("component %s is required", component)
		}
	}
	return nil
}

func generateInstallManifests(targetPath, namespace, tmpDir string, localManifests string) (string, error) {
	opts := install.Options{
		BaseURL:                localManifests,
		Version:                bootstrapVersion,
		Namespace:              namespace,
		Components:             bootstrapComponents(),
		Registry:               bootstrapRegistry,
		ImagePullSecret:        bootstrapImagePullSecret,
		Arch:                   bootstrapArch.String(),
		WatchAllNamespaces:     bootstrapWatchAllNamespaces,
		NetworkPolicy:          bootstrapNetworkPolicy,
		LogLevel:               bootstrapLogLevel.String(),
		NotificationController: defaults.NotificationController,
		ManifestFile:           defaults.ManifestFile,
		Timeout:                timeout,
		TargetPath:             targetPath,
		ClusterDomain:          bootstrapClusterDomain,
	}

	if localManifests == "" {
		opts.BaseURL = defaults.BaseURL
	}

	output, err := install.Generate(opts)
	if err != nil {
		return "", fmt.Errorf("generating install manifests failed: %w", err)
	}

	if filePath, err := output.WriteFile(tmpDir); err != nil {
		return "", fmt.Errorf("generating install manifests failed: %w", err)
	} else {
		return filePath, nil
	}

}

func applyInstallManifests(ctx context.Context, manifestPath string, components []string) error {
	kubectlArgs := []string{"apply", "-f", manifestPath}
	if _, err := utils.ExecKubectlCommand(ctx, utils.ModeOS, kubeconfig, kubecontext, kubectlArgs...); err != nil {
		return fmt.Errorf("install failed")
	}

	for _, deployment := range components {
		kubectlArgs = []string{"-n", namespace, "rollout", "status", "deployment", deployment, "--timeout", timeout.String()}
		if _, err := utils.ExecKubectlCommand(ctx, utils.ModeOS, kubeconfig, kubecontext, kubectlArgs...); err != nil {
			return fmt.Errorf("install failed")
		}
	}
	return nil
}

func generateSyncManifests(url, branch, name, namespace, targetPath, tmpDir string, interval time.Duration) error {
	opts := sync.Options{
		Name:         name,
		Namespace:    namespace,
		URL:          url,
		Branch:       branch,
		Interval:     interval,
		TargetPath:   targetPath,
		ManifestFile: sync.MakeDefaultOptions().ManifestFile,
	}

	manifest, err := sync.Generate(opts)
	if err != nil {
		return fmt.Errorf("generating install manifests failed: %w", err)
	}

	if _, err := manifest.WriteFile(tmpDir); err != nil {
		return err
	}

	if err := utils.GenerateKustomizationYaml(filepath.Join(tmpDir, targetPath, namespace)); err != nil {
		return err
	}

	return nil
}

func applySyncManifests(ctx context.Context, kubeClient client.Client, name, namespace, targetPath, tmpDir string) error {
	kubectlArgs := []string{"apply", "-k", filepath.Join(tmpDir, targetPath, namespace)}
	if _, err := utils.ExecKubectlCommand(ctx, utils.ModeStderrOS, kubeconfig, kubecontext, kubectlArgs...); err != nil {
		return err
	}

	logger.Waitingf("waiting for cluster sync")

	var gitRepository sourcev1.GitRepository
	if err := wait.PollImmediate(pollInterval, timeout,
		isGitRepositoryReady(ctx, kubeClient, types.NamespacedName{Name: name, Namespace: namespace}, &gitRepository)); err != nil {
		return err
	}

	var kustomization kustomizev1.Kustomization
	if err := wait.PollImmediate(pollInterval, timeout,
		isKustomizationReady(ctx, kubeClient, types.NamespacedName{Name: name, Namespace: namespace}, &kustomization)); err != nil {
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
