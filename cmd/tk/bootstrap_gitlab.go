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
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxcd/pkg/git"
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

  # Run bootstrap for a private repo owned by a GitLab group
  bootstrap gitlab --owner=<group> --repository=<repo name>

  # Run bootstrap for a repository path
  bootstrap gitlab --owner=<group> --repository=<repo name> --path=dev-cluster

  # Run bootstrap for a public repository on a personal account
  bootstrap gitlab --owner=<user> --repository=<repo name> --private=false --personal=true 

  # Run bootstrap for a private repo hosted on a GitLab server 
  bootstrap gitlab --owner=<group> --repository=<repo name> --hostname=<domain>
`,
	RunE: bootstrapGitLabCmdRun,
}

var (
	glOwner      string
	glRepository string
	glInterval   time.Duration
	glPersonal   bool
	glPrivate    bool
	glHostname   string
	glPath       string
)

func init() {
	bootstrapGitLabCmd.Flags().StringVar(&glOwner, "owner", "", "GitLab user or group name")
	bootstrapGitLabCmd.Flags().StringVar(&glRepository, "repository", "", "GitLab repository name")
	bootstrapGitLabCmd.Flags().BoolVar(&glPersonal, "personal", false, "is personal repository")
	bootstrapGitLabCmd.Flags().BoolVar(&glPrivate, "private", true, "is private repository")
	bootstrapGitLabCmd.Flags().DurationVar(&glInterval, "interval", time.Minute, "sync interval")
	bootstrapGitLabCmd.Flags().StringVar(&glHostname, "hostname", git.GitLabDefaultHostname, "GitLab hostname")
	bootstrapGitLabCmd.Flags().StringVar(&glPath, "path", "", "repository path, when specified the cluster sync will be scoped to this path")

	bootstrapCmd.AddCommand(bootstrapGitLabCmd)
}

func bootstrapGitLabCmdRun(cmd *cobra.Command, args []string) error {
	glToken := os.Getenv(git.GitLabTokenName)
	if glToken == "" {
		return fmt.Errorf("%s environment variable not found", git.GitLabTokenName)
	}

	repository, err := git.NewRepository(glRepository, glOwner, glHostname, glToken, "tk", glOwner+"@users.noreply.gitlab.com")
	if err != nil {
		return err
	}

	provider := &git.GitLabProvider{
		IsPrivate:  glPrivate,
		IsPersonal: glPersonal,
	}

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	tmpDir, err := ioutil.TempDir("", namespace)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// create GitLab project if doesn't exists
	logger.Actionf("connecting to %s", glHostname)
	changed, err := provider.CreateRepository(ctx, repository)
	if err != nil {
		return err
	}
	if changed {
		logger.Successf("repository created")
	}

	// clone repository and checkout the master branch
	if err := repository.Checkout(ctx, bootstrapBranch, tmpDir); err != nil {
		return err
	}
	logger.Successf("repository cloned")

	// generate install manifests
	logger.Generatef("generating manifests")
	manifest, err := generateInstallManifests(glPath, namespace, tmpDir)
	if err != nil {
		return err
	}

	// stage install manifests
	changed, err = repository.Commit(ctx, path.Join(glPath, namespace), "Add manifests")
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
	isInstall := shouldInstallManifests(ctx, kubeClient, namespace)

	if isInstall {
		// apply install manifests
		logger.Actionf("installing components in %s namespace", namespace)
		if err := applyInstallManifests(ctx, manifest, components); err != nil {
			return err
		}
		logger.Successf("install completed")
	}

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

		keyName := "tk"
		if glPath != "" {
			keyName = fmt.Sprintf("tk-%s", glPath)
		}

		if changed, err := provider.AddDeployKey(ctx, repository, key, keyName); err != nil {
			return err
		} else if changed {
			logger.Successf("deploy key configured")
		}
	}

	// configure repo synchronization
	if isInstall {
		// generate source and kustomization manifests
		logger.Actionf("generating sync manifests")
		if err := generateSyncManifests(repository.GetSSH(), namespace, namespace, glPath, tmpDir, glInterval); err != nil {
			return err
		}

		// commit and push manifests
		if changed, err = repository.Commit(ctx, path.Join(glPath, namespace), "Add manifests"); err != nil {
			return err
		} else if changed {
			if err := repository.Push(ctx); err != nil {
				return err
			}
			logger.Successf("sync manifests pushed")
		}

		// apply manifests and waiting for sync
		logger.Actionf("applying sync manifests")
		if err := applySyncManifests(ctx, kubeClient, namespace, namespace, glPath, tmpDir); err != nil {
			return err
		}
	}

	logger.Successf("bootstrap finished")
	return nil
}
