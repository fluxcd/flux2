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
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fluxcd/pkg/git"

	"github.com/fluxcd/flux2/internal/flags"
	"github.com/fluxcd/flux2/internal/utils"
)

var bootstrapGitHubCmd = &cobra.Command{
	Use:   "github",
	Short: "Bootstrap toolkit components in a GitHub repository",
	Long: `The bootstrap github command creates the GitHub repository if it doesn't exists and
commits the toolkit components manifests to the main branch.
Then it configures the target cluster to synchronize with the repository.
If the toolkit components are present on the cluster,
the bootstrap command will perform an upgrade if needed.`,
	Example: `  # Create a GitHub personal access token and export it as an env var
  export GITHUB_TOKEN=<my-token>

  # Run bootstrap for a private repo owned by a GitHub organization
  flux bootstrap github --owner=<organization> --repository=<repo name>

  # Run bootstrap for a private repo and assign organization teams to it
  flux bootstrap github --owner=<organization> --repository=<repo name> --team=<team1 slug> --team=<team2 slug>

  # Run bootstrap for a repository path
  flux bootstrap github --owner=<organization> --repository=<repo name> --path=dev-cluster

  # Run bootstrap for a public repository on a personal account
  flux bootstrap github --owner=<user> --repository=<repo name> --private=false --personal=true

  # Run bootstrap for a private repo hosted on GitHub Enterprise using SSH auth
  flux bootstrap github --owner=<organization> --repository=<repo name> --hostname=<domain> --ssh-hostname=<domain>

  # Run bootstrap for a private repo hosted on GitHub Enterprise using HTTPS auth
  flux bootstrap github --owner=<organization> --repository=<repo name> --hostname=<domain> --token-auth

  # Run bootstrap for a an existing repository with a branch named main
  flux bootstrap github --owner=<organization> --repository=<repo name> --branch=main
`,
	RunE: bootstrapGitHubCmdRun,
}

type githubFlags struct {
	owner       string
	repository  string
	interval    time.Duration
	personal    bool
	private     bool
	hostname    string
	path        flags.SafeRelativePath
	teams       []string
	delete      bool
	sshHostname string
}

const (
	ghDefaultPermission = "maintain"
)

var githubArgs githubFlags

func init() {
	bootstrapGitHubCmd.Flags().StringVar(&githubArgs.owner, "owner", "", "GitHub user or organization name")
	bootstrapGitHubCmd.Flags().StringVar(&githubArgs.repository, "repository", "", "GitHub repository name")
	bootstrapGitHubCmd.Flags().StringArrayVar(&githubArgs.teams, "team", []string{}, "GitHub team to be given maintainer access")
	bootstrapGitHubCmd.Flags().BoolVar(&githubArgs.personal, "personal", false, "is personal repository")
	bootstrapGitHubCmd.Flags().BoolVar(&githubArgs.private, "private", true, "is private repository")
	bootstrapGitHubCmd.Flags().DurationVar(&githubArgs.interval, "interval", time.Minute, "sync interval")
	bootstrapGitHubCmd.Flags().StringVar(&githubArgs.hostname, "hostname", git.GitHubDefaultHostname, "GitHub hostname")
	bootstrapGitHubCmd.Flags().StringVar(&githubArgs.sshHostname, "ssh-hostname", "", "GitHub SSH hostname, to be used when the SSH host differs from the HTTPS one")
	bootstrapGitHubCmd.Flags().Var(&githubArgs.path, "path", "path relative to the repository root, when specified the cluster sync will be scoped to this path")

	bootstrapGitHubCmd.Flags().BoolVar(&githubArgs.delete, "delete", false, "delete repository (used for testing only)")
	bootstrapGitHubCmd.Flags().MarkHidden("delete")

	bootstrapCmd.AddCommand(bootstrapGitHubCmd)
}

func bootstrapGitHubCmdRun(cmd *cobra.Command, args []string) error {
	ghToken := os.Getenv(git.GitHubTokenName)
	if ghToken == "" {
		return fmt.Errorf("%s environment variable not found", git.GitHubTokenName)
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

	usedPath, bootstrapPathDiffers := checkIfBootstrapPathDiffers(
		ctx,
		kubeClient,
		rootArgs.namespace,
		filepath.ToSlash(githubArgs.path.String()),
	)

	if bootstrapPathDiffers {
		return fmt.Errorf("cluster already bootstrapped to %v path", usedPath)
	}

	repository, err := git.NewRepository(
		githubArgs.repository,
		githubArgs.owner,
		githubArgs.hostname,
		ghToken,
		"flux",
		githubArgs.owner+"@users.noreply.github.com",
	)
	if err != nil {
		return err
	}

	if githubArgs.sshHostname != "" {
		repository.SSHHost = githubArgs.sshHostname
	}

	provider := &git.GithubProvider{
		IsPrivate:  githubArgs.private,
		IsPersonal: githubArgs.personal,
	}

	tmpDir, err := ioutil.TempDir("", rootArgs.namespace)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if githubArgs.delete {
		if err := provider.DeleteRepository(ctx, repository); err != nil {
			return err
		}
		logger.Successf("repository deleted")
		return nil
	}

	// create GitHub repository if doesn't exists
	logger.Actionf("connecting to %s", githubArgs.hostname)
	changed, err := provider.CreateRepository(ctx, repository)
	if err != nil {
		return err
	}
	if changed {
		logger.Successf("repository created")
	}

	withErrors := false
	// add teams to org repository
	if !githubArgs.personal {
		for _, team := range githubArgs.teams {
			if changed, err := provider.AddTeam(ctx, repository, team, ghDefaultPermission); err != nil {
				logger.Failuref(err.Error())
				withErrors = true
			} else if changed {
				logger.Successf("%s team access granted", team)
			}
		}
	}

	// clone repository and checkout the main branch
	if err := repository.Checkout(ctx, bootstrapArgs.branch, tmpDir); err != nil {
		return err
	}
	logger.Successf("repository cloned")

	// generate install manifests
	logger.Generatef("generating manifests")
	installManifest, err := generateInstallManifests(
		githubArgs.path.String(),
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
		path.Join(githubArgs.path.String(), rootArgs.namespace),
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

	repoURL := repository.GetURL()

	if bootstrapArgs.tokenAuth {
		// setup HTTPS token auth
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rootArgs.namespace,
				Namespace: rootArgs.namespace,
			},
			StringData: map[string]string{
				"username": "git",
				"password": ghToken,
			},
		}
		if err := upsertSecret(ctx, kubeClient, secret); err != nil {
			return err
		}
	} else {
		// setup SSH deploy key
		repoURL = repository.GetSSH()
		if shouldCreateDeployKey(ctx, kubeClient, rootArgs.namespace) {
			logger.Actionf("configuring deploy key")
			u, err := url.Parse(repository.GetSSH())
			if err != nil {
				return fmt.Errorf("git URL parse failed: %w", err)
			}

			key, err := generateDeployKey(ctx, kubeClient, u, rootArgs.namespace)
			if err != nil {
				return fmt.Errorf("generating deploy key failed: %w", err)
			}

			keyName := "flux"
			if githubArgs.path != "" {
				keyName = fmt.Sprintf("flux-%s", githubArgs.path)
			}

			if changed, err := provider.AddDeployKey(ctx, repository, key, keyName); err != nil {
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
		filepath.ToSlash(githubArgs.path.String()),
		tmpDir,
		githubArgs.interval,
	)
	if err != nil {
		return err
	}

	// commit and push manifests
	if changed, err = repository.Commit(
		ctx,
		path.Join(githubArgs.path.String(), rootArgs.namespace),
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

	if withErrors {
		return fmt.Errorf("bootstrap completed with errors")
	}

	logger.Successf("bootstrap finished")
	return nil
}
