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
	"crypto/elliptic"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/ssh"
)

var createSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Create or update a GitRepository source",
	Long: `
The create source git command generates a GitRepository resource and waits for it to sync.
For Git over SSH, host and SSH keys are automatically generated and stored in a Kubernetes secret.
For private Git repositories, the basic authentication credentials are stored in a Kubernetes secret.`,
	Example: `  # Create a source from a public Git repository master branch
  create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --branch=master

  # Create a source from a Git repository pinned to specific git tag
  create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --tag="3.2.3"

  # Create a source from a public Git repository tag that matches a semver range
  create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --tag-semver=">=3.2.0 <3.3.0"

  # Create a source from a Git repository using SSH authentication
  create source git podinfo \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --branch=master

  # Create a source from a Git repository using SSH authentication and an
  # ECDSA P-521 curve public key
  create source git podinfo \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --branch=master \
    --ssh-key-algorithm=ecdsa \
    --ssh-ecdsa-curve=p521

  # Create a source from a Git repository using basic authentication
  create source git podinfo \
    --url=https://github.com/stefanprodan/podinfo \
    --username=username \
    --password=password
`,
	RunE: createSourceGitCmdRun,
}

var (
	sourceGitURL          string
	sourceGitBranch       string
	sourceGitTag          string
	sourceGitSemver       string
	sourceGitUsername     string
	sourceGitPassword     string
	sourceGitKeyAlgorithm PublicKeyAlgorithm = "rsa"
	sourceGitRSABits      RSAKeyBits         = 2048
	sourceGitECDSACurve                      = ECDSACurve{elliptic.P384()}
)

func init() {
	createSourceGitCmd.Flags().StringVar(&sourceGitURL, "url", "", "git address, e.g. ssh://git@host/org/repository")
	createSourceGitCmd.Flags().StringVar(&sourceGitBranch, "branch", "master", "git branch")
	createSourceGitCmd.Flags().StringVar(&sourceGitTag, "tag", "", "git tag")
	createSourceGitCmd.Flags().StringVar(&sourceGitSemver, "tag-semver", "", "git tag semver range")
	createSourceGitCmd.Flags().StringVarP(&sourceGitUsername, "username", "u", "", "basic authentication username")
	createSourceGitCmd.Flags().StringVarP(&sourceGitPassword, "password", "p", "", "basic authentication password")
	createSourceGitCmd.Flags().Var(&sourceGitKeyAlgorithm, "ssh-key-algorithm", sourceGitKeyAlgorithm.Description())
	createSourceGitCmd.Flags().Var(&sourceGitRSABits, "ssh-rsa-bits", sourceGitRSABits.Description())
	createSourceGitCmd.Flags().Var(&sourceGitECDSACurve, "ssh-ecdsa-curve", sourceGitECDSACurve.Description())

	createSourceCmd.AddCommand(createSourceGitCmd)
}

func createSourceGitCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("source name is required")
	}
	name := args[0]

	if sourceGitURL == "" {
		return fmt.Errorf("git-url is required")
	}

	tmpDir, err := ioutil.TempDir("", name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	u, err := url.Parse(sourceGitURL)
	if err != nil {
		return fmt.Errorf("git URL parse failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	gitRepository := sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: sourcev1.GitRepositorySpec{
			URL: sourceGitURL,
			Interval: metav1.Duration{
				Duration: interval,
			},
			Reference: &sourcev1.GitRepositoryRef{},
		},
	}

	if sourceGitSemver != "" {
		gitRepository.Spec.Reference.SemVer = sourceGitSemver
	} else if sourceGitTag != "" {
		gitRepository.Spec.Reference.Tag = sourceGitTag
	} else {
		gitRepository.Spec.Reference.Branch = sourceGitBranch
	}

	if export {
		return exportGit(gitRepository)
	}

	withAuth := false
	// TODO(hidde): move all auth prep to separate func?
	if u.Scheme == "ssh" {
		logger.Actionf("generating deploy key pair")
		pair, err := generateKeyPair(ctx)
		if err != nil {
			return err
		}

		fmt.Printf("%s", pair.PublicKey)
		prompt := promptui.Prompt{
			Label:     "Have you added the deploy key to your repository",
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}

		logger.Actionf("collecting preferred public key from SSH server")
		hostKey, err := scanHostKey(ctx, u)
		if err != nil {
			return err
		}
		logger.Successf("collected public key from SSH server:")
		fmt.Printf("%s", hostKey)

		logger.Actionf("applying secret with keys")
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			StringData: map[string]string{
				"identity":     string(pair.PrivateKey),
				"identity.pub": string(pair.PublicKey),
				"known_hosts":  string(hostKey),
			},
		}
		if err := upsertSecret(ctx, kubeClient, secret); err != nil {
			return err
		}
		withAuth = true
	} else if sourceGitUsername != "" && sourceGitPassword != "" {
		logger.Actionf("applying secret with basic auth credentials")
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			StringData: map[string]string{
				"username": sourceGitUsername,
				"password": sourceGitPassword,
			},
		}
		if err := upsertSecret(ctx, kubeClient, secret); err != nil {
			return err
		}
		withAuth = true
	}

	if withAuth {
		logger.Successf("authentication configured")
	}

	logger.Generatef("generating source")

	if withAuth {
		gitRepository.Spec.SecretRef = &corev1.LocalObjectReference{
			Name: name,
		}
	}

	logger.Actionf("applying source")
	if err := upsertGitRepository(ctx, kubeClient, gitRepository); err != nil {
		return err
	}

	logger.Waitingf("waiting for git sync")
	if err := wait.PollImmediate(pollInterval, timeout,
		isGitRepositoryReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("git sync completed")

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err = kubeClient.Get(ctx, namespacedName, &gitRepository)
	if err != nil {
		return fmt.Errorf("git sync failed: %w", err)
	}

	if gitRepository.Status.Artifact != nil {
		logger.Successf("fetched revision: %s", gitRepository.Status.Artifact.Revision)
	} else {
		return fmt.Errorf("git sync failed, artifact not found")
	}

	return nil
}

func generateKeyPair(ctx context.Context) (*ssh.KeyPair, error) {
	var keyGen ssh.KeyPairGenerator
	switch algorithm := sourceGitKeyAlgorithm.String(); algorithm {
	case "rsa":
		keyGen = ssh.NewRSAGenerator(int(sourceGitRSABits))
	case "ecdsa":
		keyGen = ssh.NewECDSAGenerator(sourceGitECDSACurve.Curve)
	case "ed25519":
		keyGen = ssh.NewEd25519Generator()
	default:
		return nil, fmt.Errorf("unsupported public key algorithm: %s", algorithm)
	}
	pair, err := keyGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("key pair generation failed, error: %w", err)
	}
	return pair, nil
}

func scanHostKey(ctx context.Context, url *url.URL) ([]byte, error) {
	host := url.Host
	if url.Port() == "" {
		host = host + ":22"
	}
	hostKey, err := ssh.ScanHostKey(host, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("SSH key scan for host %s failed, error: %w", host, err)
	}
	return hostKey, nil
}

func upsertSecret(ctx context.Context, kubeClient client.Client, secret corev1.Secret) error {
	namespacedName := types.NamespacedName{
		Namespace: secret.GetNamespace(),
		Name:      secret.GetName(),
	}

	var existing corev1.Secret
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, &secret); err != nil {
				return err
			} else {
				return nil
			}
		}
		return err
	}

	existing.StringData = secret.StringData
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return err
	}
	return nil
}

func upsertGitRepository(ctx context.Context, kubeClient client.Client, gitRepository sourcev1.GitRepository) error {
	namespacedName := types.NamespacedName{
		Namespace: gitRepository.GetNamespace(),
		Name:      gitRepository.GetName(),
	}

	var existing sourcev1.GitRepository
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, &gitRepository); err != nil {
				return err
			} else {
				logger.Successf("source created")
				return nil
			}
		}
		return err
	}

	existing.Spec = gitRepository.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return err
	}

	logger.Successf("source updated")
	return nil
}

func isGitRepositoryReady(ctx context.Context, kubeClient client.Client, name, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		var gitRepository sourcev1.GitRepository
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := kubeClient.Get(ctx, namespacedName, &gitRepository)
		if err != nil {
			return false, err
		}

		for _, condition := range gitRepository.Status.Conditions {
			if condition.Type == sourcev1.ReadyCondition {
				if condition.Status == corev1.ConditionTrue {
					return true, nil
				} else if condition.Status == corev1.ConditionFalse {
					return false, fmt.Errorf(condition.Message)
				}
			}
		}
		return false, nil
	}
}
