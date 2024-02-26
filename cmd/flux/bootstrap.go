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
	"crypto/elliptic"
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Deploy Flux on a cluster the GitOps way.",
	Long: `The bootstrap sub-commands push the Flux manifests to a Git repository
and deploy Flux on the cluster.`,
}

type bootstrapFlags struct {
	version  string
	logLevel flags.LogLevel

	branch            string
	recurseSubmodules bool
	manifestsPath     string

	defaultComponents  []string
	extraComponents    []string
	requiredComponents []string

	registry        string
	imagePullSecret string

	secretName     string
	tokenAuth      bool
	keyAlgorithm   flags.PublicKeyAlgorithm
	keyRSABits     flags.RSAKeyBits
	keyECDSACurve  flags.ECDSACurve
	sshHostname    string
	caFile         string
	privateKeyFile string

	watchAllNamespaces bool
	networkPolicy      bool
	clusterDomain      string
	tolerationKeys     []string

	authorName  string
	authorEmail string

	gpgKeyRingPath string
	gpgPassphrase  string
	gpgKeyID       string

	force bool

	commitMessageAppendix string
}

const (
	bootstrapDefaultBranch = "main"
)

var bootstrapArgs = NewBootstrapFlags()

func init() {
	bootstrapCmd.PersistentFlags().StringVarP(&bootstrapArgs.version, "version", "v", "",
		"toolkit version, when specified the manifests are downloaded from https://github.com/fluxcd/flux2/releases")

	bootstrapCmd.PersistentFlags().StringSliceVar(&bootstrapArgs.defaultComponents, "components", rootArgs.defaults.Components,
		"list of components, accepts comma-separated values")
	bootstrapCmd.PersistentFlags().StringSliceVar(&bootstrapArgs.extraComponents, "components-extra", nil,
		"list of components in addition to those supplied or defaulted, accepts values such as 'image-reflector-controller,image-automation-controller'")

	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.registry, "registry", "ghcr.io/fluxcd",
		"container registry where the Flux controller images are published")
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.imagePullSecret, "image-pull-secret", "",
		"Kubernetes secret name used for pulling the controller images from a private registry")

	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.branch, "branch", bootstrapDefaultBranch, "Git branch")
	bootstrapCmd.PersistentFlags().BoolVar(&bootstrapArgs.recurseSubmodules, "recurse-submodules", false,
		"when enabled, configures the GitRepository source to initialize and include Git submodules in the artifact it produces")

	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.manifestsPath, "manifests", "", "path to the manifest directory")

	bootstrapCmd.PersistentFlags().BoolVar(&bootstrapArgs.watchAllNamespaces, "watch-all-namespaces", true,
		"watch for custom resources in all namespaces, if set to false it will only watch the namespace where the Flux controllers are installed")
	bootstrapCmd.PersistentFlags().BoolVar(&bootstrapArgs.networkPolicy, "network-policy", true,
		"setup Kubernetes network policies to deny ingress access to the Flux controllers from other namespaces")
	bootstrapCmd.PersistentFlags().BoolVar(&bootstrapArgs.tokenAuth, "token-auth", false,
		"when enabled, the personal access token will be used instead of the SSH deploy key")
	bootstrapCmd.PersistentFlags().Var(&bootstrapArgs.logLevel, "log-level", bootstrapArgs.logLevel.Description())
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.clusterDomain, "cluster-domain", rootArgs.defaults.ClusterDomain, "internal cluster domain")
	bootstrapCmd.PersistentFlags().StringSliceVar(&bootstrapArgs.tolerationKeys, "toleration-keys", nil,
		"list of toleration keys used to schedule the controller pods onto nodes with matching taints")

	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.secretName, "secret-name", rootArgs.defaults.Namespace, "name of the secret the sync credentials can be found in or stored to")
	bootstrapCmd.PersistentFlags().Var(&bootstrapArgs.keyAlgorithm, "ssh-key-algorithm", bootstrapArgs.keyAlgorithm.Description())
	bootstrapCmd.PersistentFlags().Var(&bootstrapArgs.keyRSABits, "ssh-rsa-bits", bootstrapArgs.keyRSABits.Description())
	bootstrapCmd.PersistentFlags().Var(&bootstrapArgs.keyECDSACurve, "ssh-ecdsa-curve", bootstrapArgs.keyECDSACurve.Description())
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.sshHostname, "ssh-hostname", "", "SSH hostname, to be used when the SSH host differs from the HTTPS one")
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.caFile, "ca-file", "", "path to TLS CA file used for validating self-signed certificates")
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.privateKeyFile, "private-key-file", "", "path to a private key file used for authenticating to the Git SSH server")

	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.authorName, "author-name", "Flux", "author name for Git commits")
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.authorEmail, "author-email", "", "author email for Git commits")

	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.gpgKeyRingPath, "gpg-key-ring", "", "path to GPG key ring for signing commits")
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.gpgPassphrase, "gpg-passphrase", "", "passphrase for decrypting GPG private key")
	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.gpgKeyID, "gpg-key-id", "", "key id for selecting a particular key")

	bootstrapCmd.PersistentFlags().StringVar(&bootstrapArgs.commitMessageAppendix, "commit-message-appendix", "", "string to add to the commit messages, e.g. '[ci skip]'")

	bootstrapCmd.PersistentFlags().BoolVar(&bootstrapArgs.force, "force", false, "override existing Flux installation if it's managed by a different tool such as Helm")
	bootstrapCmd.PersistentFlags().MarkHidden("manifests")

	rootCmd.AddCommand(bootstrapCmd)
}

func NewBootstrapFlags() bootstrapFlags {
	return bootstrapFlags{
		logLevel:           flags.LogLevel(rootArgs.defaults.LogLevel),
		requiredComponents: []string{"source-controller", "kustomize-controller"},
		keyAlgorithm:       flags.PublicKeyAlgorithm(sourcesecret.ECDSAPrivateKeyAlgorithm),
		keyRSABits:         2048,
		keyECDSACurve:      flags.ECDSACurve{Curve: elliptic.P384()},
	}
}

func bootstrapComponents() []string {
	return append(bootstrapArgs.defaultComponents, bootstrapArgs.extraComponents...)
}

func buildEmbeddedManifestBase() (string, error) {
	if !isEmbeddedVersion(bootstrapArgs.version) {
		return "", nil
	}
	tmpBaseDir, err := manifestgen.MkdirTempAbs("", "flux-manifests-")
	if err != nil {
		return "", err
	}
	if err := writeEmbeddedManifests(tmpBaseDir); err != nil {
		return "", err
	}
	return tmpBaseDir, nil
}

func bootstrapValidate() error {
	components := bootstrapComponents()
	for _, component := range bootstrapArgs.requiredComponents {
		if !utils.ContainsItemString(components, component) {
			return fmt.Errorf("component %s is required", component)
		}
	}

	if err := utils.ValidateComponents(components); err != nil {
		return err
	}

	return nil
}

func mapTeamSlice(s []string, defaultPermission string) map[string]string {
	m := make(map[string]string, len(s))
	for _, v := range s {
		m[v] = defaultPermission
		if s := strings.Split(v, ":"); len(s) == 2 {
			m[s[0]] = s[1]
		}
	}

	return m
}

// confirmBootstrap gets a confirmation for running bootstrap over an existing Flux installation.
// It returns a nil error if Flux is not installed or the user confirms overriding an existing installation
func confirmBootstrap(ctx context.Context, kubeClient client.Client) error {
	installed := true
	info, err := getFluxClusterInfo(ctx, kubeClient)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("cluster info unavailable: %w", err)
		}
		installed = false
	}

	if installed {
		err = confirmFluxInstallOverride(info)
		if err != nil {
			if err == promptui.ErrAbort {
				return fmt.Errorf("bootstrap cancelled")
			}
			return err
		}
	}
	return nil
}
