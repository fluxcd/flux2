/*
Copyright 2021 The Flux authors

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

var bootstrapBServerCmd = &cobra.Command{
	Use:   "bitbucket-server",
	Short: "Deploy Flux on a cluster connected to a Bitbucket Server repository",
	Long: `The bootstrap bitbucket-server command creates the Bitbucket Server repository if it doesn't exists and
commits the Flux manifests to the master branch.
Then it configures the target cluster to synchronize with the repository.
If the Flux components are present on the cluster,
the bootstrap command will perform an upgrade if needed.`,
	Example: `  # Create a Bitbucket Server API token and export it as an env var
  export BITBUCKET_TOKEN=<my-token>

  # Run bootstrap for a private repository using HTTPS token authentication
  flux bootstrap bitbucket-server --owner=<project> --username=<user> --repository=<repository name> --hostname=<domain> --token-auth --path=clusters/my-cluster

  # Run bootstrap for a private repository using SSH authentication
  flux bootstrap bitbucket-server --owner=<project> --username=<user> --repository=<repository name> --hostname=<domain> --path=clusters/my-cluster

  # Run bootstrap for a public repository on a personal account
  flux bootstrap bitbucket-server --owner=<user> --repository=<repository name> --private=false --personal --hostname=<domain> --token-auth --path=clusters/my-cluster

  # Run bootstrap for an existing repository with a branch named main
  flux bootstrap bitbucket-server --owner=<project> --username=<user> --repository=<repository name> --branch=main --hostname=<domain> --token-auth --path=clusters/my-cluster`,
	RunE: bootstrapBServerCmdRun,
}

const (
	bServerDefaultPermission = "push"
	bServerTokenEnvVar       = "BITBUCKET_TOKEN"
)

type bServerFlags struct {
	owner        string
	repository   string
	interval     time.Duration
	personal     bool
	username     string
	private      bool
	hostname     string
	path         flags.SafeRelativePath
	teams        []string
	readWriteKey bool
	reconcile    bool
}

var bServerArgs bServerFlags

func init() {
	bootstrapBServerCmd.Flags().StringVar(&bServerArgs.owner, "owner", "", "Bitbucket Server user or project name")
	bootstrapBServerCmd.Flags().StringVar(&bServerArgs.repository, "repository", "", "Bitbucket Server repository name")
	bootstrapBServerCmd.Flags().StringSliceVar(&bServerArgs.teams, "group", []string{}, "Bitbucket Server groups to be given write access (also accepts comma-separated values)")
	bootstrapBServerCmd.Flags().BoolVar(&bServerArgs.personal, "personal", false, "if true, the owner is assumed to be a Bitbucket Server user; otherwise a group")
	bootstrapBServerCmd.Flags().StringVarP(&bServerArgs.username, "username", "u", "git", "authentication username")
	bootstrapBServerCmd.Flags().BoolVar(&bServerArgs.private, "private", true, "if true, the repository is setup or configured as private")
	bootstrapBServerCmd.Flags().DurationVar(&bServerArgs.interval, "interval", time.Minute, "sync interval")
	bootstrapBServerCmd.Flags().StringVar(&bServerArgs.hostname, "hostname", "", "Bitbucket Server hostname")
	bootstrapBServerCmd.Flags().Var(&bServerArgs.path, "path", "path relative to the repository root, when specified the cluster sync will be scoped to this path")
	bootstrapBServerCmd.Flags().BoolVar(&bServerArgs.readWriteKey, "read-write-key", false, "if true, the deploy key is configured with read/write permissions")
	bootstrapBServerCmd.Flags().BoolVar(&bServerArgs.reconcile, "reconcile", false, "if true, the configured options are also reconciled if the repository already exists")

	bootstrapCmd.AddCommand(bootstrapBServerCmd)
}

func bootstrapBServerCmdRun(cmd *cobra.Command, args []string) error {
	bitbucketToken := os.Getenv(bServerTokenEnvVar)
	if bitbucketToken == "" {
		var err error
		bitbucketToken, err = readPasswordFromStdin("Please enter your Bitbucket personal access token (PAT): ")
		if err != nil {
			return fmt.Errorf("could not read token: %w", err)
		}
	}

	if bServerArgs.hostname == "" {
		return fmt.Errorf("invalid hostname %q", bServerArgs.hostname)
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

	if !bootstrapArgs.force {
		err = confirmBootstrap(ctx, kubeClient)
		if err != nil {
			return err
		}
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

	user := bServerArgs.username
	if bServerArgs.personal {
		user = bServerArgs.owner
	}

	var caBundle []byte
	if bootstrapArgs.caFile != "" {
		var err error
		caBundle, err = os.ReadFile(bootstrapArgs.caFile)
		if err != nil {
			return fmt.Errorf("unable to read TLS CA file: %w", err)
		}
	}

	// Build Bitbucket Server provider
	providerCfg := provider.Config{
		Provider: provider.GitProviderStash,
		Hostname: bServerArgs.hostname,
		Username: user,
		Token:    bitbucketToken,
		CaBundle: caBundle,
	}

	providerClient, err := provider.BuildGitProvider(providerCfg)
	if err != nil {
		return err
	}

	// Lazy go-git repository
	tmpDir, err := manifestgen.MkdirTempAbs("", "flux-bootstrap-")
	if err != nil {
		return fmt.Errorf("failed to create temporary working dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	clientOpts := []gogit.ClientOption{gogit.WithDiskStorage(), gogit.WithFallbackToDefaultKnownHosts()}
	gitClient, err := gogit.NewClient(tmpDir, &git.AuthOptions{
		Transport: git.HTTPS,
		Username:  user,
		Password:  bitbucketToken,
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
		TargetPath:             bServerArgs.path.ToSlash(),
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
		TargetPath:   bServerArgs.path.String(),
		ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
	}
	if bootstrapArgs.tokenAuth {
		if bServerArgs.personal {
			secretOpts.Username = bServerArgs.owner
		} else {
			secretOpts.Username = bServerArgs.username
		}
		secretOpts.Password = bitbucketToken
		secretOpts.CAFile = caBundle
	} else {
		keypair, err := sourcesecret.LoadKeyPairFromPath(bootstrapArgs.privateKeyFile, gitArgs.password)
		if err != nil {
			return err
		}
		secretOpts.Keypair = keypair
		secretOpts.PrivateKeyAlgorithm = sourcesecret.PrivateKeyAlgorithm(bootstrapArgs.keyAlgorithm)
		secretOpts.RSAKeyBits = int(bootstrapArgs.keyRSABits)
		secretOpts.ECDSACurve = bootstrapArgs.keyECDSACurve.Curve

		secretOpts.SSHHostname = bServerArgs.hostname
		if bootstrapArgs.sshHostname != "" {
			secretOpts.SSHHostname = bootstrapArgs.sshHostname
		}
	}

	// Sync manifest config
	syncOpts := sync.Options{
		Interval:          bServerArgs.interval,
		Name:              *kubeconfigArgs.Namespace,
		Namespace:         *kubeconfigArgs.Namespace,
		Branch:            bootstrapArgs.branch,
		Secret:            bootstrapArgs.secretName,
		TargetPath:        bServerArgs.path.ToSlash(),
		ManifestFile:      sync.MakeDefaultOptions().ManifestFile,
		RecurseSubmodules: bootstrapArgs.recurseSubmodules,
	}

	entityList, err := bootstrap.LoadEntityListFromPath(bootstrapArgs.gpgKeyRingPath)
	if err != nil {
		return err
	}

	// Bootstrap config

	bootstrapOpts := []bootstrap.GitProviderOption{
		bootstrap.WithProviderRepository(bServerArgs.owner, bServerArgs.repository, bServerArgs.personal),
		bootstrap.WithBranch(bootstrapArgs.branch),
		bootstrap.WithBootstrapTransportType("https"),
		bootstrap.WithSignature(bootstrapArgs.authorName, bootstrapArgs.authorEmail),
		bootstrap.WithCommitMessageAppendix(bootstrapArgs.commitMessageAppendix),
		bootstrap.WithProviderTeamPermissions(mapTeamSlice(bServerArgs.teams, bServerDefaultPermission)),
		bootstrap.WithReadWriteKeyPermissions(bServerArgs.readWriteKey),
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
	if !bServerArgs.private {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithProviderRepositoryConfig("", "", "public"))
	}
	if bServerArgs.reconcile {
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
