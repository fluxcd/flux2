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
	"os"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/internal/bootstrap"
	"github.com/fluxcd/flux2/internal/bootstrap/git/gogit"
	"github.com/fluxcd/flux2/internal/bootstrap/provider"
	"github.com/fluxcd/flux2/internal/flags"
	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/pkg/manifestgen/sync"
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

  # Run bootstrap for a private repository owned by a GitHub organization
  flux bootstrap github --owner=<organization> --repository=<repository name>

  # Run bootstrap for a private repository and assign organization teams to it
  flux bootstrap github --owner=<organization> --repository=<repository name> --team=<team1 slug> --team=<team2 slug>

  # Run bootstrap for a private repository and assign organization teams with their access level(e.g maintain, admin) to it
  flux bootstrap github --owner=<organization> --repository=<repository name> --team=<team1 slug>:<access-level>

  # Run bootstrap for a repository path
  flux bootstrap github --owner=<organization> --repository=<repository name> --path=dev-cluster

  # Run bootstrap for a public repository on a personal account
  flux bootstrap github --owner=<user> --repository=<repository name> --private=false --personal=true

  # Run bootstrap for a private repository hosted on GitHub Enterprise using SSH auth
  flux bootstrap github --owner=<organization> --repository=<repository name> --hostname=<domain> --ssh-hostname=<domain>

  # Run bootstrap for a private repository hosted on GitHub Enterprise using HTTPS auth
  flux bootstrap github --owner=<organization> --repository=<repository name> --hostname=<domain> --token-auth

  # Run bootstrap for an existing repository with a branch named main
  flux bootstrap github --owner=<organization> --repository=<repository name> --branch=main`,
	RunE: bootstrapGitHubCmdRun,
}

type githubFlags struct {
	owner        string
	repository   string
	interval     time.Duration
	personal     bool
	private      bool
	hostname     string
	path         flags.SafeRelativePath
	teams        []string
	readWriteKey bool
	reconcile    bool
}

const (
	ghDefaultPermission = "maintain"
	ghDefaultDomain     = "github.com"
	ghTokenEnvVar       = "GITHUB_TOKEN"
)

var githubArgs githubFlags

func init() {
	bootstrapGitHubCmd.Flags().StringVar(&githubArgs.owner, "owner", "", "GitHub user or organization name")
	bootstrapGitHubCmd.Flags().StringVar(&githubArgs.repository, "repository", "", "GitHub repository name")
	bootstrapGitHubCmd.Flags().StringSliceVar(&githubArgs.teams, "team", []string{}, "GitHub team and the access to be given to it(team:maintain). Defaults to maintainer access if no access level is specified (also accepts comma-separated values)")
	bootstrapGitHubCmd.Flags().BoolVar(&githubArgs.personal, "personal", false, "if true, the owner is assumed to be a GitHub user; otherwise an org")
	bootstrapGitHubCmd.Flags().BoolVar(&githubArgs.private, "private", true, "if true, the repository is setup or configured as private")
	bootstrapGitHubCmd.Flags().DurationVar(&githubArgs.interval, "interval", time.Minute, "sync interval")
	bootstrapGitHubCmd.Flags().StringVar(&githubArgs.hostname, "hostname", ghDefaultDomain, "GitHub hostname")
	bootstrapGitHubCmd.Flags().Var(&githubArgs.path, "path", "path relative to the repository root, when specified the cluster sync will be scoped to this path")
	bootstrapGitHubCmd.Flags().BoolVar(&githubArgs.readWriteKey, "read-write-key", false, "if true, the deploy key is configured with read/write permissions")
	bootstrapGitHubCmd.Flags().BoolVar(&githubArgs.reconcile, "reconcile", false, "if true, the configured options are also reconciled if the repository already exists")

	bootstrapCmd.AddCommand(bootstrapGitHubCmd)
}

func bootstrapGitHubCmdRun(cmd *cobra.Command, args []string) error {
	ghToken := os.Getenv(ghTokenEnvVar)
	if ghToken == "" {
		var err error
		ghToken, err = readPasswordFromStdin("Please enter your GitHub personal access token (PAT): ")
		if err != nil {
			return fmt.Errorf("could not read token: %w", err)
		}
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

	// Manifest base
	if ver, err := getVersion(bootstrapArgs.version); err == nil {
		bootstrapArgs.version = ver
	}
	manifestsBase, err := buildEmbeddedManifestBase()
	if err != nil {
		return err
	}
	defer os.RemoveAll(manifestsBase)

	var caBundle []byte
	if bootstrapArgs.caFile != "" {
		var err error
		caBundle, err = os.ReadFile(bootstrapArgs.caFile)
		if err != nil {
			return fmt.Errorf("unable to read TLS CA file: %w", err)
		}
	}
	// Build GitHub provider
	providerCfg := provider.Config{
		Provider: provider.GitProviderGitHub,
		Hostname: githubArgs.hostname,
		Token:    ghToken,
		CaBundle: caBundle,
	}
	providerClient, err := provider.BuildGitProvider(providerCfg)
	if err != nil {
		return err
	}

	// Lazy go-git repository
	tmpDir, err := os.MkdirTemp("", "flux-bootstrap-")
	if err != nil {
		return fmt.Errorf("failed to create temporary working dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	gitClient := gogit.New(tmpDir, &http.BasicAuth{
		Username: githubArgs.owner,
		Password: ghToken,
	})

	// Install manifest config
	installOptions := install.Options{
		BaseURL:                rootArgs.defaults.BaseURL,
		Version:                bootstrapArgs.version,
		Namespace:              rootArgs.namespace,
		Components:             bootstrapComponents(),
		Registry:               bootstrapArgs.registry,
		ImagePullSecret:        bootstrapArgs.imagePullSecret,
		WatchAllNamespaces:     bootstrapArgs.watchAllNamespaces,
		NetworkPolicy:          bootstrapArgs.networkPolicy,
		LogLevel:               bootstrapArgs.logLevel.String(),
		NotificationController: rootArgs.defaults.NotificationController,
		ManifestFile:           rootArgs.defaults.ManifestFile,
		Timeout:                rootArgs.timeout,
		TargetPath:             githubArgs.path.ToSlash(),
		ClusterDomain:          bootstrapArgs.clusterDomain,
		TolerationKeys:         bootstrapArgs.tolerationKeys,
	}
	if customBaseURL := bootstrapArgs.manifestsPath; customBaseURL != "" {
		installOptions.BaseURL = customBaseURL
	}

	// Source generation and secret config
	secretOpts := sourcesecret.Options{
		Name:         bootstrapArgs.secretName,
		Namespace:    rootArgs.namespace,
		TargetPath:   githubArgs.path.ToSlash(),
		ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
	}
	if bootstrapArgs.tokenAuth {
		secretOpts.Username = "git"
		secretOpts.Password = ghToken

		if bootstrapArgs.caFile != "" {
			secretOpts.CAFilePath = bootstrapArgs.caFile
		}
	} else {
		secretOpts.PrivateKeyAlgorithm = sourcesecret.PrivateKeyAlgorithm(bootstrapArgs.keyAlgorithm)
		secretOpts.RSAKeyBits = int(bootstrapArgs.keyRSABits)
		secretOpts.ECDSACurve = bootstrapArgs.keyECDSACurve.Curve
		secretOpts.SSHHostname = githubArgs.hostname

		if bootstrapArgs.sshHostname != "" {
			secretOpts.SSHHostname = bootstrapArgs.sshHostname
		}
	}

	// Sync manifest config
	syncOpts := sync.Options{
		Interval:          githubArgs.interval,
		Name:              rootArgs.namespace,
		Namespace:         rootArgs.namespace,
		Branch:            bootstrapArgs.branch,
		Secret:            bootstrapArgs.secretName,
		TargetPath:        githubArgs.path.ToSlash(),
		ManifestFile:      sync.MakeDefaultOptions().ManifestFile,
		GitImplementation: sourceGitArgs.gitImplementation.String(),
		RecurseSubmodules: bootstrapArgs.recurseSubmodules,
	}

	// Bootstrap config
	bootstrapOpts := []bootstrap.GitProviderOption{
		bootstrap.WithProviderRepository(githubArgs.owner, githubArgs.repository, githubArgs.personal),
		bootstrap.WithBranch(bootstrapArgs.branch),
		bootstrap.WithBootstrapTransportType("https"),
		bootstrap.WithAuthor(bootstrapArgs.authorName, bootstrapArgs.authorEmail),
		bootstrap.WithCommitMessageAppendix(bootstrapArgs.commitMessageAppendix),
		bootstrap.WithProviderTeamPermissions(mapTeamSlice(githubArgs.teams, ghDefaultPermission)),
		bootstrap.WithReadWriteKeyPermissions(githubArgs.readWriteKey),
		bootstrap.WithKubeconfig(rootArgs.kubeconfig, rootArgs.kubecontext),
		bootstrap.WithLogger(logger),
		bootstrap.WithCABundle(caBundle),
	}
	if bootstrapArgs.sshHostname != "" {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithSSHHostname(bootstrapArgs.sshHostname))
	}
	if bootstrapArgs.tokenAuth {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithSyncTransportType("https"))
	}
	if !githubArgs.private {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithProviderRepositoryConfig("", "", "public"))
	}
	if githubArgs.reconcile {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithReconcile())
	}

	// Setup bootstrapper with constructed configs
	b, err := bootstrap.NewGitProviderBootstrapper(gitClient, providerClient, kubeClient, bootstrapOpts...)
	if err != nil {
		return err
	}

	// Run
	return bootstrap.Run(ctx, b, manifestsBase, installOptions, secretOpts, syncOpts, rootArgs.pollInterval, rootArgs.timeout)
}
