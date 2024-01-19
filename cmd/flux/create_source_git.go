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
	"net/url"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/pkg/apis/meta"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
)

type sourceGitFlags struct {
	url               string
	branch            string
	tag               string
	semver            string
	refName           string
	commit            string
	username          string
	password          string
	keyAlgorithm      flags.PublicKeyAlgorithm
	keyRSABits        flags.RSAKeyBits
	keyECDSACurve     flags.ECDSACurve
	secretRef         string
	caFile            string
	privateKeyFile    string
	recurseSubmodules bool
	silent            bool
	ignorePaths       []string
}

var createSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Create or update a GitRepository source",
	Long: `The create source git command generates a GitRepository resource and waits for it to sync.
For Git over SSH, host and SSH keys are automatically generated and stored in a Kubernetes secret.
For private Git repositories, the basic authentication credentials are stored in a Kubernetes secret.`,
	Example: `  # Create a source from a public Git repository master branch
  flux create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --branch=master

  # Create a source for a Git repository pinned to specific git tag
  flux create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --tag="3.2.3"

  # Create a source from a public Git repository tag that matches a semver range
  flux create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --tag-semver=">=3.2.0 <3.3.0"

  # Create a source for a Git repository using SSH authentication
  flux create source git podinfo \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --branch=master

  # Create a source for a Git repository using SSH authentication and an
  # ECDSA P-521 curve public key
  flux create source git podinfo \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --branch=master \
    --ssh-key-algorithm=ecdsa \
    --ssh-ecdsa-curve=p521

  # Create a source for a Git repository using SSH authentication and a
  #	passwordless private key from file
  # The public SSH host key will still be gathered from the host
  flux create source git podinfo \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --branch=master \
    --private-key-file=./private.key

  # Create a source for a Git repository using SSH authentication and a
  # private key with a password from file
  # The public SSH host key will still be gathered from the host
  flux create source git podinfo \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --branch=master \
    --private-key-file=./private.key \
    --password=<password>

  # Create a source for a Git repository using basic authentication
  flux create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --branch=master \
    --username=username \
    --password=password`,
	RunE: createSourceGitCmdRun,
}

var sourceGitArgs = newSourceGitFlags()

func init() {
	createSourceGitCmd.Flags().StringVar(&sourceGitArgs.url, "url", "", "git address, e.g. ssh://git@host/org/repository")
	createSourceGitCmd.Flags().StringVar(&sourceGitArgs.branch, "branch", "", "git branch")
	createSourceGitCmd.Flags().StringVar(&sourceGitArgs.tag, "tag", "", "git tag")
	createSourceGitCmd.Flags().StringVar(&sourceGitArgs.semver, "tag-semver", "", "git tag semver range")
	createSourceGitCmd.Flags().StringVar(&sourceGitArgs.refName, "ref-name", "", " git reference name")
	createSourceGitCmd.Flags().StringVar(&sourceGitArgs.commit, "commit", "", "git commit")
	createSourceGitCmd.Flags().StringVarP(&sourceGitArgs.username, "username", "u", "", "basic authentication username")
	createSourceGitCmd.Flags().StringVarP(&sourceGitArgs.password, "password", "p", "", "basic authentication password")
	createSourceGitCmd.Flags().Var(&sourceGitArgs.keyAlgorithm, "ssh-key-algorithm", sourceGitArgs.keyAlgorithm.Description())
	createSourceGitCmd.Flags().Var(&sourceGitArgs.keyRSABits, "ssh-rsa-bits", sourceGitArgs.keyRSABits.Description())
	createSourceGitCmd.Flags().Var(&sourceGitArgs.keyECDSACurve, "ssh-ecdsa-curve", sourceGitArgs.keyECDSACurve.Description())
	createSourceGitCmd.Flags().StringVar(&sourceGitArgs.secretRef, "secret-ref", "", "the name of an existing secret containing SSH or basic credentials")
	createSourceGitCmd.Flags().StringVar(&sourceGitArgs.caFile, "ca-file", "", "path to TLS CA file used for validating self-signed certificates")
	createSourceGitCmd.Flags().StringVar(&sourceGitArgs.privateKeyFile, "private-key-file", "", "path to a passwordless private key file used for authenticating to the Git SSH server")
	createSourceGitCmd.Flags().BoolVar(&sourceGitArgs.recurseSubmodules, "recurse-submodules", false,
		"when enabled, configures the GitRepository source to initialize and include Git submodules in the artifact it produces")
	createSourceGitCmd.Flags().BoolVarP(&sourceGitArgs.silent, "silent", "s", false, "assumes the deploy key is already setup, skips confirmation")
	createSourceGitCmd.Flags().StringSliceVar(&sourceGitArgs.ignorePaths, "ignore-paths", nil, "set paths to ignore in git resource (can specify multiple paths with commas: path1,path2)")

	createSourceCmd.AddCommand(createSourceGitCmd)
}

func newSourceGitFlags() sourceGitFlags {
	return sourceGitFlags{
		keyAlgorithm:  flags.PublicKeyAlgorithm(sourcesecret.ECDSAPrivateKeyAlgorithm),
		keyRSABits:    2048,
		keyECDSACurve: flags.ECDSACurve{Curve: elliptic.P384()},
	}
}

func createSourceGitCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if sourceGitArgs.url == "" {
		return fmt.Errorf("url is required")
	}

	u, err := url.Parse(sourceGitArgs.url)
	if err != nil {
		return fmt.Errorf("git URL parse failed: %w", err)
	}
	if u.Scheme != "ssh" && u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("git URL scheme '%s' not supported, can be: ssh, http and https", u.Scheme)
	}

	if sourceGitArgs.branch == "" && sourceGitArgs.tag == "" && sourceGitArgs.semver == "" && sourceGitArgs.commit == "" && sourceGitArgs.refName == "" {
		return fmt.Errorf("a Git ref is required, use one of the following: --branch, --tag, --commit, --ref-name or --tag-semver")
	}

	if sourceGitArgs.caFile != "" && u.Scheme == "ssh" {
		return fmt.Errorf("specifying a CA file is not supported for Git over SSH")
	}

	tmpDir, err := os.MkdirTemp("", name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	var ignorePaths *string
	if len(sourceGitArgs.ignorePaths) > 0 {
		ignorePathsStr := strings.Join(sourceGitArgs.ignorePaths, "\n")
		ignorePaths = &ignorePathsStr
	}

	gitRepository := sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *kubeconfigArgs.Namespace,
			Labels:    sourceLabels,
		},
		Spec: sourcev1.GitRepositorySpec{
			URL: sourceGitArgs.url,
			Interval: metav1.Duration{
				Duration: createArgs.interval,
			},
			RecurseSubmodules: sourceGitArgs.recurseSubmodules,
			Reference:         &sourcev1.GitRepositoryRef{},
			Ignore:            ignorePaths,
		},
	}

	if createSourceArgs.fetchTimeout > 0 {
		gitRepository.Spec.Timeout = &metav1.Duration{Duration: createSourceArgs.fetchTimeout}
	}

	if sourceGitArgs.commit != "" {
		gitRepository.Spec.Reference.Commit = sourceGitArgs.commit
		gitRepository.Spec.Reference.Branch = sourceGitArgs.branch
	} else if sourceGitArgs.refName != "" {
		gitRepository.Spec.Reference.Name = sourceGitArgs.refName
	} else if sourceGitArgs.semver != "" {
		gitRepository.Spec.Reference.SemVer = sourceGitArgs.semver
	} else if sourceGitArgs.tag != "" {
		gitRepository.Spec.Reference.Tag = sourceGitArgs.tag
	} else {
		gitRepository.Spec.Reference.Branch = sourceGitArgs.branch
	}

	if sourceGitArgs.secretRef != "" {
		gitRepository.Spec.SecretRef = &meta.LocalObjectReference{
			Name: sourceGitArgs.secretRef,
		}
	}

	if createArgs.export {
		return printExport(exportGit(&gitRepository))
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	logger.Generatef("generating GitRepository source")
	if sourceGitArgs.secretRef == "" {
		secretOpts := sourcesecret.Options{
			Name:         name,
			Namespace:    *kubeconfigArgs.Namespace,
			ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
		}
		switch u.Scheme {
		case "ssh":
			keypair, err := sourcesecret.LoadKeyPairFromPath(sourceGitArgs.privateKeyFile, sourceGitArgs.password)
			if err != nil {
				return err
			}
			secretOpts.Keypair = keypair
			secretOpts.SSHHostname = u.Host
			secretOpts.PrivateKeyAlgorithm = sourcesecret.PrivateKeyAlgorithm(sourceGitArgs.keyAlgorithm)
			secretOpts.RSAKeyBits = int(sourceGitArgs.keyRSABits)
			secretOpts.ECDSACurve = sourceGitArgs.keyECDSACurve.Curve
			secretOpts.Password = sourceGitArgs.password
		case "https":
			if sourceGitArgs.caFile != "" {
				caBundle, err := os.ReadFile(sourceGitArgs.caFile)
				if err != nil {
					return fmt.Errorf("unable to read TLS CA file: %w", err)
				}
				secretOpts.CAFile = caBundle
			}
			secretOpts.Username = sourceGitArgs.username
			secretOpts.Password = sourceGitArgs.password
		case "http":
			logger.Warningf("insecure configuration: credentials configured for an HTTP URL")
			secretOpts.Username = sourceGitArgs.username
			secretOpts.Password = sourceGitArgs.password
		}
		secret, err := sourcesecret.Generate(secretOpts)
		if err != nil {
			return err
		}
		var s corev1.Secret
		if err = yaml.Unmarshal([]byte(secret.Content), &s); err != nil {
			return err
		}
		if len(s.StringData) > 0 {
			if hk, ok := s.StringData[sourcesecret.KnownHostsSecretKey]; ok {
				logger.Successf("collected public key from SSH server:\n%s", hk)
			}
			if ppk, ok := s.StringData[sourcesecret.PublicKeySecretKey]; ok {
				logger.Generatef("deploy key: %s", ppk)
				if !sourceGitArgs.silent {
					prompt := promptui.Prompt{
						Label:     "Have you added the deploy key to your repository",
						IsConfirm: true,
					}
					if _, err := prompt.Run(); err != nil {
						return fmt.Errorf("aborting")
					}
				}
			}
			logger.Actionf("applying secret with repository credentials")
			if err := upsertSecret(ctx, kubeClient, s); err != nil {
				return err
			}
			gitRepository.Spec.SecretRef = &meta.LocalObjectReference{
				Name: s.Name,
			}
			logger.Successf("authentication configured")
		}
	}

	logger.Actionf("applying GitRepository source")
	namespacedName, err := upsertGitRepository(ctx, kubeClient, &gitRepository)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for GitRepository source reconciliation")
	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true,
		isObjectReadyConditionFunc(kubeClient, namespacedName, &gitRepository)); err != nil {
		return err
	}
	logger.Successf("GitRepository source reconciliation completed")

	if gitRepository.Status.Artifact == nil {
		return fmt.Errorf("GitRepository source reconciliation completed but no artifact was found")
	}
	logger.Successf("fetched revision: %s", gitRepository.Status.Artifact.Revision)
	return nil
}

func upsertGitRepository(ctx context.Context, kubeClient client.Client,
	gitRepository *sourcev1.GitRepository) (types.NamespacedName, error) {
	namespacedName := types.NamespacedName{
		Namespace: gitRepository.GetNamespace(),
		Name:      gitRepository.GetName(),
	}

	var existing sourcev1.GitRepository
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, gitRepository); err != nil {
				return namespacedName, err
			} else {
				logger.Successf("GitRepository source created")
				return namespacedName, nil
			}
		}
		return namespacedName, err
	}

	existing.Labels = gitRepository.Labels
	existing.Spec = gitRepository.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return namespacedName, err
	}
	gitRepository = &existing
	logger.Successf("GitRepository source updated")
	return namespacedName, nil
}
