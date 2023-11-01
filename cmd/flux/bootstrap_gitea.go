/*
Copyright 2023 The Flux authors

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

	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/bootstrap"
	"github.com/fluxcd/flux2/v2/pkg/bootstrap/provider"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sync"
)

var bootstrapGiteaCmd = &cobra.Command{
	Use:   "gitea",
	Short: "Deploy Flux on a cluster connected to a Gitea repository",
	Long: `The bootstrap gitea command creates the Gitea repository if it doesn't exists and
commits the Flux manifests to the specified branch.
Then it configures the target cluster to synchronize with that repository.
If the Flux components are present on the cluster,
the bootstrap command will perform an upgrade if needed.`,
	Example: `  # Create a Gitea personal access token and export it as an env var
  export GITEA_TOKEN=<my-token>

  # Run bootstrap for a private repository owned by a Gitea organization
  flux bootstrap gitea --owner=<organization> --repository=<repository name> --path=clusters/my-cluster

  # Run bootstrap for a private repository and assign organization teams to it
  flux bootstrap gitea --owner=<organization> --repository=<repository name> --team=<team1 slug> --team=<team2 slug> --path=clusters/my-cluster

  # Run bootstrap for a private repository and assign organization teams with their access level(e.g maintain, admin) to it
  flux bootstrap gitea --owner=<organization> --repository=<repository name> --team=<team1 slug>:<access-level> --path=clusters/my-cluster

  # Run bootstrap for a public repository on a personal account
  flux bootstrap gitea --owner=<user> --repository=<repository name> --private=false --personal=true --path=clusters/my-cluster

  # Run bootstrap for a private repository hosted on Gitea Enterprise using SSH auth
  flux bootstrap gitea --owner=<organization> --repository=<repository name> --hostname=<domain> --ssh-hostname=<domain> --path=clusters/my-cluster

  # Run bootstrap for a private repository hosted on Gitea Enterprise using HTTPS auth
  flux bootstrap gitea --owner=<organization> --repository=<repository name> --hostname=<domain> --token-auth --path=clusters/my-cluster

  # Run bootstrap for an existing repository with a branch named main
  flux bootstrap gitea --owner=<organization> --repository=<repository name> --branch=main --path=clusters/my-cluster`,
	RunE: bootstrapGiteaCmdRun,
}

type giteaFlags struct {
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
	gtDefaultPermission = "maintain"
	gtDefaultDomain     = "gitea.com"
	gtTokenEnvVar       = "GITEA_TOKEN"
)

var giteaArgs giteaFlags

func init() {
	bootstrapGiteaCmd.Flags().StringVar(&giteaArgs.owner, "owner", "", "Gitea user or organization name")
	bootstrapGiteaCmd.Flags().StringVar(&giteaArgs.repository, "repository", "", "Gitea repository name")
	bootstrapGiteaCmd.Flags().StringSliceVar(&giteaArgs.teams, "team", []string{}, "Gitea team and the access to be given to it(team:maintain). Defaults to maintainer access if no access level is specified (also accepts comma-separated values)")
	bootstrapGiteaCmd.Flags().BoolVar(&giteaArgs.personal, "personal", false, "if true, the owner is assumed to be a Gitea user; otherwise an org")
	bootstrapGiteaCmd.Flags().BoolVar(&giteaArgs.private, "private", true, "if true, the repository is setup or configured as private")
	bootstrapGiteaCmd.Flags().DurationVar(&giteaArgs.interval, "interval", time.Minute, "sync interval")
	bootstrapGiteaCmd.Flags().StringVar(&giteaArgs.hostname, "hostname", gtDefaultDomain, "Gitea hostname")
	bootstrapGiteaCmd.Flags().Var(&giteaArgs.path, "path", "path relative to the repository root, when specified the cluster sync will be scoped to this path")
	bootstrapGiteaCmd.Flags().BoolVar(&giteaArgs.readWriteKey, "read-write-key", false, "if true, the deploy key is configured with read/write permissions")
	bootstrapGiteaCmd.Flags().BoolVar(&giteaArgs.reconcile, "reconcile", false, "if true, the configured options are also reconciled if the repository already exists")

	bootstrapCmd.AddCommand(bootstrapGiteaCmd)
}

func bootstrapGiteaCmdRun(cmd *cobra.Command, args []string) error {
	gtToken := os.Getenv(gtTokenEnvVar)
	if gtToken == "" {
		var err error
		gtToken, err = readPasswordFromStdin("Please enter your Gitea personal access token (PAT): ")
		if err != nil {
			return fmt.Errorf("could not read token: %w", err)
		}
	}

	if err := bootstrapValidate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	// Manifest base
	if ver, err := getVersion(bootstrapArgs.version); err != nil {
		return err
	} else {
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
	// Build Gitea provider
	providerCfg := provider.Config{
		Provider: provider.GitProviderGitea,
		Hostname: giteaArgs.hostname,
		Token:    gtToken,
		CaBundle: caBundle,
	}
	providerClient, err := provider.BuildGitProvider(providerCfg)
	if err != nil {
		return err
	}

	tmpDir, err := manifestgen.MkdirTempAbs("", "flux-bootstrap-")
	if err != nil {
		return fmt.Errorf("failed to create temporary working dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	clientOpts := []gogit.ClientOption{gogit.WithDiskStorage(), gogit.WithFallbackToDefaultKnownHosts()}
	gitClient, err := gogit.NewClient(tmpDir, &git.AuthOptions{
		Transport: git.HTTPS,
		Username:  giteaArgs.owner,
		Password:  gtToken,
		CAFile:    caBundle,
	}, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create a Git client: %w", err)
	}

	// Install manifest config
	installOptions := install.Options{
		BaseURL:                rootArgs.defaults.BaseURL,
		Version:                bootstrapArgs.version,
		Namespace:              *kubeconfigArgs.Namespace,
		Components:             bootstrapComponents(),
		Registry:               bootstrapArgs.registry,
		ImagePullSecret:        bootstrapArgs.imagePullSecret,
		WatchAllNamespaces:     bootstrapArgs.watchAllNamespaces,
		NetworkPolicy:          bootstrapArgs.networkPolicy,
		LogLevel:               bootstrapArgs.logLevel.String(),
		NotificationController: rootArgs.defaults.NotificationController,
		ManifestFile:           rootArgs.defaults.ManifestFile,
		Timeout:                rootArgs.timeout,
		TargetPath:             giteaArgs.path.ToSlash(),
		ClusterDomain:          bootstrapArgs.clusterDomain,
		TolerationKeys:         bootstrapArgs.tolerationKeys,
	}
	if customBaseURL := bootstrapArgs.manifestsPath; customBaseURL != "" {
		installOptions.BaseURL = customBaseURL
	}

	// Source generation and secret config
	secretOpts := sourcesecret.Options{
		Name:         bootstrapArgs.secretName,
		Namespace:    *kubeconfigArgs.Namespace,
		TargetPath:   giteaArgs.path.ToSlash(),
		ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
	}
	if bootstrapArgs.tokenAuth {
		secretOpts.Username = "git"
		secretOpts.Password = gtToken
		secretOpts.CAFile = caBundle
	} else {
		secretOpts.PrivateKeyAlgorithm = sourcesecret.PrivateKeyAlgorithm(bootstrapArgs.keyAlgorithm)
		secretOpts.RSAKeyBits = int(bootstrapArgs.keyRSABits)
		secretOpts.ECDSACurve = bootstrapArgs.keyECDSACurve.Curve

		secretOpts.SSHHostname = giteaArgs.hostname
		if bootstrapArgs.sshHostname != "" {
			secretOpts.SSHHostname = bootstrapArgs.sshHostname
		}
	}

	// Sync manifest config
	syncOpts := sync.Options{
		Interval:          giteaArgs.interval,
		Name:              *kubeconfigArgs.Namespace,
		Namespace:         *kubeconfigArgs.Namespace,
		Branch:            bootstrapArgs.branch,
		Secret:            bootstrapArgs.secretName,
		TargetPath:        giteaArgs.path.ToSlash(),
		ManifestFile:      sync.MakeDefaultOptions().ManifestFile,
		RecurseSubmodules: bootstrapArgs.recurseSubmodules,
	}

	entityList, err := bootstrap.LoadEntityListFromPath(bootstrapArgs.gpgKeyRingPath)
	if err != nil {
		return err
	}

	// Bootstrap config
	bootstrapOpts := []bootstrap.GitProviderOption{
		bootstrap.WithProviderRepository(giteaArgs.owner, giteaArgs.repository, giteaArgs.personal),
		bootstrap.WithBranch(bootstrapArgs.branch),
		bootstrap.WithBootstrapTransportType("https"),
		bootstrap.WithSignature(bootstrapArgs.authorName, bootstrapArgs.authorEmail),
		bootstrap.WithCommitMessageAppendix(bootstrapArgs.commitMessageAppendix),
		bootstrap.WithProviderTeamPermissions(mapTeamSlice(giteaArgs.teams, gtDefaultPermission)),
		bootstrap.WithReadWriteKeyPermissions(giteaArgs.readWriteKey),
		bootstrap.WithKubeconfig(kubeconfigArgs, kubeclientOptions),
		bootstrap.WithLogger(logger),
		bootstrap.WithGitCommitSigning(entityList, bootstrapArgs.gpgPassphrase, bootstrapArgs.gpgKeyID),
	}
	if bootstrapArgs.sshHostname != "" {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithSSHHostname(bootstrapArgs.sshHostname))
	}
	if bootstrapArgs.tokenAuth {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithSyncTransportType("https"))
	}
	if !giteaArgs.private {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithProviderRepositoryConfig("", "", "public"))
	}
	if giteaArgs.reconcile {
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
