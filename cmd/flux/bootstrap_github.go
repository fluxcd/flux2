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
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/git"
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

var (
	ghOwner       string
	ghRepository  string
	ghInterval    time.Duration
	ghPersonal    bool
	ghPrivate     bool
	ghHostname    string
	ghPath        string
	ghTeams       []string
	ghDelete      bool
	ghSSHHostname string
)

const (
	ghDefaultPermission = "maintain"
)

func init() {
	bootstrapGitHubCmd.Flags().StringVar(&ghOwner, "owner", "", "GitHub user or organization name")
	bootstrapGitHubCmd.Flags().StringVar(&ghRepository, "repository", "", "GitHub repository name")
	bootstrapGitHubCmd.Flags().StringArrayVar(&ghTeams, "team", []string{}, "GitHub team to be given maintainer access")
	bootstrapGitHubCmd.Flags().BoolVar(&ghPersonal, "personal", false, "is personal repository")
	bootstrapGitHubCmd.Flags().BoolVar(&ghPrivate, "private", true, "is private repository")
	bootstrapGitHubCmd.Flags().DurationVar(&ghInterval, "interval", time.Minute, "sync interval")
	bootstrapGitHubCmd.Flags().StringVar(&ghHostname, "hostname", git.GitHubDefaultHostname, "GitHub hostname")
	bootstrapGitHubCmd.Flags().StringVar(&ghSSHHostname, "ssh-hostname", "", "GitHub SSH hostname, to be used when the SSH host differs from the HTTPS one")
	bootstrapGitHubCmd.Flags().StringVar(&ghPath, "path", "", "repository path, when specified the cluster sync will be scoped to this path")

	bootstrapGitHubCmd.Flags().BoolVar(&ghDelete, "delete", false, "delete repository (used for testing only)")
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

	repository, err := git.NewRepository(ghRepository, ghOwner, ghHostname, ghToken, "flux", ghOwner+"@users.noreply.github.com")
	if err != nil {
		return err
	}

	if ghSSHHostname != "" {
		repository.SSHHost = ghSSHHostname
	}

	provider := &git.GithubProvider{
		IsPrivate:  ghPrivate,
		IsPersonal: ghPersonal,
	}

	tmpDir, err := ioutil.TempDir("", namespace)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if ghDelete {
		if err := provider.DeleteRepository(ctx, repository); err != nil {
			return err
		}
		logger.Successf("repository deleted")
		return nil
	}

	// create GitHub repository if doesn't exists
	logger.Actionf("connecting to %s", ghHostname)
	changed, err := provider.CreateRepository(ctx, repository)
	if err != nil {
		return err
	}
	if changed {
		logger.Successf("repository created")
	}

	withErrors := false
	// add teams to org repository
	if !ghPersonal {
		for _, team := range ghTeams {
			if changed, err := provider.AddTeam(ctx, repository, team, ghDefaultPermission); err != nil {
				logger.Failuref(err.Error())
				withErrors = true
			} else if changed {
				logger.Successf("%s team access granted", team)
			}
		}
	}

	// clone repository and checkout the main branch
	if err := repository.Checkout(ctx, bootstrapBranch, tmpDir); err != nil {
		return err
	}
	logger.Successf("repository cloned")

	// generate install manifests
	logger.Generatef("generating manifests")
	manifest, err := generateInstallManifests(ghPath, namespace, tmpDir, bootstrapManifestsPath)
	if err != nil {
		return err
	}

	// stage install manifests
	changed, err = repository.Commit(ctx, path.Join(ghPath, namespace), "Add manifests")
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

	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return err
	}

	// determine if repo synchronization is working
	isInstall := shouldInstallManifests(ctx, kubeClient, namespace)

	if isInstall {
		// apply install manifests
		logger.Actionf("installing components in %s namespace", namespace)
		if err := applyInstallManifests(ctx, manifest, bootstrapComponents); err != nil {
			return err
		}
		logger.Successf("install completed")
	}

	if bootstrapTokenAuth {
		// setup HTTPS token auth
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespace,
				Namespace: namespace,
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
		if shouldCreateDeployKey(ctx, kubeClient, namespace) {
			logger.Actionf("configuring deploy key")
			u, err := url.Parse(repository.GetSSH())
			if err != nil {
				return fmt.Errorf("git URL parse failed: %w", err)
			}

			key, err := generateDeployKey(ctx, kubeClient, u, namespace)
			if err != nil {
				return fmt.Errorf("generating deploy key failed: %w", err)
			}

			keyName := "flux"
			if ghPath != "" {
				keyName = fmt.Sprintf("flux-%s", ghPath)
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
	if err := generateSyncManifests(repository.GetSSH(), bootstrapBranch, namespace, namespace, ghPath, tmpDir, ghInterval); err != nil {
		return err
	}

	// commit and push manifests
	if changed, err = repository.Commit(ctx, path.Join(ghPath, namespace), "Add manifests"); err != nil {
		return err
	} else if changed {
		if err := repository.Push(ctx); err != nil {
			return err
		}
		logger.Successf("sync manifests pushed")
	}

	// apply manifests and waiting for sync
	logger.Actionf("applying sync manifests")
	if err := applySyncManifests(ctx, kubeClient, namespace, namespace, ghPath, tmpDir); err != nil {
		return err
	}

	if withErrors {
		return fmt.Errorf("bootstrap completed with errors")
	}

	logger.Successf("bootstrap finished")
	return nil
}
