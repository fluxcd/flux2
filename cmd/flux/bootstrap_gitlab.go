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
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/pkg/git"

	"github.com/fluxcd/flux2/internal/flags"
	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
)

var bootstrapGitLabCmd = &cobra.Command{
	Use:   "gitlab",
	Short: "Bootstrap toolkit components in a GitLab repository",
	Long: `The bootstrap gitlab command creates the GitLab repository if it doesn't exists and
commits the toolkit components manifests to the master branch.
Then it configures the target cluster to synchronize with the repository.
If the toolkit components are present on the cluster,
the bootstrap command will perform an upgrade if needed.`,
	Example: `  # Create a GitLab API token and export it as an env var
  export GITLAB_TOKEN=<my-token>

  # Run bootstrap for a private repo using HTTPS token authentication
  flux bootstrap gitlab --owner=<group> --repository=<repo name> --token-auth

  # Run bootstrap for a private repo using SSH authentication
  flux bootstrap gitlab --owner=<group> --repository=<repo name>

  # Run bootstrap for a repository path
  flux bootstrap gitlab --owner=<group> --repository=<repo name> --path=dev-cluster

  # Run bootstrap for a public repository on a personal account
  flux bootstrap gitlab --owner=<user> --repository=<repo name> --private=false --personal --token-auth

  # Run bootstrap for a private repo hosted on a GitLab server
  flux bootstrap gitlab --owner=<group> --repository=<repo name> --hostname=<domain> --token-auth

  # Run bootstrap for a an existing repository with a branch named main
  flux bootstrap gitlab --owner=<organization> --repository=<repo name> --branch=main --token-auth
`,
	RunE: bootstrapGitLabCmdRun,
}

const (
	gitlabProjectRegex = `\A[[:alnum:]\x{00A9}-\x{1f9ff}_][[:alnum:]\p{Pd}\x{00A9}-\x{1f9ff}_\.]*\z`
)

type gitlabFlags struct {
	owner       string
	repository  string
	interval    time.Duration
	personal    bool
	private     bool
	hostname    string
	sshHostname string
	path        flags.SafeRelativePath
}

var gitlabArgs gitlabFlags

func init() {
	bootstrapGitLabCmd.Flags().StringVar(&gitlabArgs.owner, "owner", "", "GitLab user or group name")
	bootstrapGitLabCmd.Flags().StringVar(&gitlabArgs.repository, "repository", "", "GitLab repository name")
	bootstrapGitLabCmd.Flags().BoolVar(&gitlabArgs.personal, "personal", false, "if true, the owner is assumed to be a GitLab user; otherwise a group")
	bootstrapGitLabCmd.Flags().BoolVar(&gitlabArgs.private, "private", true, "if true, the repository is assumed to be private")
	bootstrapGitLabCmd.Flags().DurationVar(&gitlabArgs.interval, "interval", time.Minute, "sync interval")
	bootstrapGitLabCmd.Flags().StringVar(&gitlabArgs.hostname, "hostname", git.GitLabDefaultHostname, "GitLab hostname")
	bootstrapGitLabCmd.Flags().StringVar(&gitlabArgs.sshHostname, "ssh-hostname", "", "GitLab SSH hostname, to be used when the SSH host differs from the HTTPS one")
	bootstrapGitLabCmd.Flags().Var(&gitlabArgs.path, "path", "path relative to the repository root, when specified the cluster sync will be scoped to this path")

	bootstrapCmd.AddCommand(bootstrapGitLabCmd)
}

func bootstrapGitLabCmdRun(cmd *cobra.Command, args []string) error {
	glToken := os.Getenv(git.GitLabTokenName)
	if glToken == "" {
		return fmt.Errorf("%s environment variable not found", git.GitLabTokenName)
	}

	projectNameIsValid, err := regexp.MatchString(gitlabProjectRegex, gitlabArgs.repository)
	if err != nil {
		return err
	}
	if !projectNameIsValid {
		return fmt.Errorf("%s is an invalid project name for gitlab.\nIt can contain only letters, digits, emojis, '_', '.', dash, space. It must start with letter, digit, emoji or '_'.", gitlabArgs.repository)
	}

	if err := bootstrapValidate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	usedPath, bootstrapPathDiffers := checkIfBootstrapPathDiffers(ctx, kubeClient, rootArgs.namespace, filepath.ToSlash(gitlabArgs.path.String()))

	if bootstrapPathDiffers {
		return fmt.Errorf("cluster already bootstrapped to %v path", usedPath)
	}

	repository, err := git.NewRepository(
		gitlabArgs.repository,
		gitlabArgs.owner,
		gitlabArgs.hostname,
		glToken,
		"flux",
		gitlabArgs.owner+"@users.noreply.gitlab.com",
	)
	if err != nil {
		return err
	}

	if gitlabArgs.sshHostname != "" {
		repository.SSHHost = gitlabArgs.sshHostname
	}

	tmpDir, err := ioutil.TempDir("", rootArgs.namespace)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	provider := &git.GitLabProvider{
		IsPrivate:  gitlabArgs.private,
		IsPersonal: gitlabArgs.personal,
	}

	// create GitLab project if doesn't exists
	logger.Actionf("connecting to %s", gitlabArgs.hostname)
	changed, err := provider.CreateRepository(ctx, repository)
	if err != nil {
		return err
	}
	if changed {
		logger.Successf("repository created")
	}

	// clone repository and checkout the master branch
	if err := repository.Checkout(ctx, bootstrapArgs.branch, tmpDir); err != nil {
		return err
	}
	logger.Successf("repository cloned")

	// generate install manifests
	logger.Generatef("generating manifests")
	installManifest, err := generateInstallManifests(
		gitlabArgs.path.String(),
		rootArgs.namespace,
		tmpDir,
		bootstrapArgs.manifestsPath,
	)
	if err != nil {
		return err
	}

	// stage install manifests
	changed, err = repository.Commit(
		ctx,
		path.Join(gitlabArgs.path.String(), rootArgs.namespace),
		fmt.Sprintf("Add flux %s components manifests", bootstrapArgs.version),
	)
	if err != nil {
		return err
	}

	// push install manifests
	if changed {
		if err := repository.Push(ctx); err != nil {
			return err
		}
		logger.Successf("components manifests pushed")
	} else {
		logger.Successf("components are up to date")
	}

	// determine if repo synchronization is working
	isInstall := shouldInstallManifests(ctx, kubeClient, rootArgs.namespace)

	if isInstall {
		// apply install manifests
		logger.Actionf("installing components in %s namespace", rootArgs.namespace)
		if err := applyInstallManifests(ctx, installManifest, bootstrapComponents()); err != nil {
			return err
		}
		logger.Successf("install completed")
	}

	repoURL := repository.GetSSH()
	secretOpts := sourcesecret.Options{
		Name:      rootArgs.namespace,
		Namespace: rootArgs.namespace,
	}
	if bootstrapArgs.tokenAuth {
		// Setup HTTPS token auth
		repoURL = repository.GetURL()
		secretOpts.Username = "git"
		secretOpts.Password = glToken
	} else if shouldCreateDeployKey(ctx, kubeClient, rootArgs.namespace) {
		// Setup SSH auth
		u, err := url.Parse(repoURL)
		if err != nil {
			return fmt.Errorf("git URL parse failed: %w", err)
		}
		secretOpts.SSHHostname = u.Hostname()
		secretOpts.PrivateKeyAlgorithm = sourcesecret.RSAPrivateKeyAlgorithm
		secretOpts.RSAKeyBits = 2048
	}

	secret, err := sourcesecret.Generate(secretOpts)
	if err != nil {
		return err
	}
	var s corev1.Secret
	if err := yaml.Unmarshal([]byte(secret.Content), &s); err != nil {
		return err
	}
	if len(s.StringData) > 0 {
		logger.Actionf("configuring deploy key")
		if err := upsertSecret(ctx, kubeClient, s); err != nil {
			return err
		}

		if ppk, ok := s.StringData[sourcesecret.PublicKeySecretKey]; ok {
			keyName := "flux"
			if gitlabArgs.path != "" {
				keyName = fmt.Sprintf("flux-%s", gitlabArgs.path)
			}

			if changed, err := provider.AddDeployKey(ctx, repository, ppk, keyName); err != nil {
				return err
			} else if changed {
				logger.Successf("deploy key configured")
			}
		}
	}

	// configure repo synchronization
	logger.Actionf("generating sync manifests")
	syncManifests, err := generateSyncManifests(
		repoURL,
		bootstrapArgs.branch,
		rootArgs.namespace,
		rootArgs.namespace,
		filepath.ToSlash(gitlabArgs.path.String()),
		tmpDir,
		gitlabArgs.interval,
	)
	if err != nil {
		return err
	}

	// commit and push manifests
	if changed, err = repository.Commit(
		ctx,
		path.Join(gitlabArgs.path.String(), rootArgs.namespace),
		fmt.Sprintf("Add flux %s sync manifests", bootstrapArgs.version),
	); err != nil {
		return err
	} else if changed {
		if err := repository.Push(ctx); err != nil {
			return err
		}
		logger.Successf("sync manifests pushed")
	}

	// apply manifests and waiting for sync
	logger.Actionf("applying sync manifests")
	if err := applySyncManifests(ctx, kubeClient, rootArgs.namespace, rootArgs.namespace, syncManifests); err != nil {
		return err
	}

	logger.Successf("bootstrap finished")
	return nil
}
