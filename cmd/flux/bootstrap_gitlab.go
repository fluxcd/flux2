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
	"regexp"
	"strings"
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

var bootstrapGitLabCmd = &cobra.Command{
	Use:   "gitlab",
	Short: "Deploy Flux on a cluster connected to a GitLab repository",
	Long: `The bootstrap gitlab command creates the GitLab repository if it doesn't exists and
commits the Flux manifests to the specified branch.
Then it configures the target cluster to synchronize with that repository.
If the Flux components are present on the cluster,
the bootstrap command will perform an upgrade if needed.`,
	Example: `  # Create a GitLab API token and export it as an env var
  export GITLAB_TOKEN=<my-token>

  # Run bootstrap for a private repository using HTTPS token authentication
  flux bootstrap gitlab --owner=<group> --repository=<repository name> --token-auth

  # Run bootstrap for a private repository using SSH authentication
  flux bootstrap gitlab --owner=<group> --repository=<repository name>

  # Run bootstrap for a repository path
  flux bootstrap gitlab --owner=<group> --repository=<repository name> --path=dev-cluster

  # Run bootstrap for a public repository on a personal account
  flux bootstrap gitlab --owner=<user> --repository=<repository name> --private=false --personal --token-auth

  # Run bootstrap for a private repository hosted on a GitLab server
  flux bootstrap gitlab --owner=<group> --repository=<repository name> --hostname=<domain> --token-auth

  # Run bootstrap for an existing repository with a branch named main
  flux bootstrap gitlab --owner=<organization> --repository=<repository name> --branch=main --token-auth

  # Run bootstrap for a private repository using Deploy Token authentication
  flux bootstrap gitlab --owner=<group> --repository=<repository name> --deploy-token-auth
  `,
	RunE: bootstrapGitLabCmdRun,
}

const (
	glDefaultPermission = "maintain"
	glDefaultDomain     = "gitlab.com"
	glTokenEnvVar       = "GITLAB_TOKEN"
	gitlabProjectRegex  = `\A[[:alnum:]\x{00A9}-\x{1f9ff}_][[:alnum:]\p{Pd}\x{00A9}-\x{1f9ff}_\.]*\z`
)

type gitlabFlags struct {
	owner           string
	repository      string
	interval        time.Duration
	personal        bool
	private         bool
	hostname        string
	path            flags.SafeRelativePath
	teams           []string
	readWriteKey    bool
	reconcile       bool
	deployTokenAuth bool
}

var gitlabArgs gitlabFlags

func init() {
	bootstrapGitLabCmd.Flags().StringVar(&gitlabArgs.owner, "owner", "", "GitLab user or group name")
	bootstrapGitLabCmd.Flags().StringVar(&gitlabArgs.repository, "repository", "", "GitLab repository name")
	bootstrapGitLabCmd.Flags().StringSliceVar(&gitlabArgs.teams, "team", []string{}, "GitLab teams to be given maintainer access (also accepts comma-separated values)")
	bootstrapGitLabCmd.Flags().BoolVar(&gitlabArgs.personal, "personal", false, "if true, the owner is assumed to be a GitLab user; otherwise a group")
	bootstrapGitLabCmd.Flags().BoolVar(&gitlabArgs.private, "private", true, "if true, the repository is setup or configured as private")
	bootstrapGitLabCmd.Flags().DurationVar(&gitlabArgs.interval, "interval", time.Minute, "sync interval")
	bootstrapGitLabCmd.Flags().StringVar(&gitlabArgs.hostname, "hostname", glDefaultDomain, "GitLab hostname")
	bootstrapGitLabCmd.Flags().Var(&gitlabArgs.path, "path", "path relative to the repository root, when specified the cluster sync will be scoped to this path")
	bootstrapGitLabCmd.Flags().BoolVar(&gitlabArgs.readWriteKey, "read-write-key", false, "if true, the deploy key is configured with read/write permissions")
	bootstrapGitLabCmd.Flags().BoolVar(&gitlabArgs.reconcile, "reconcile", false, "if true, the configured options are also reconciled if the repository already exists")
	bootstrapGitLabCmd.Flags().BoolVar(&gitlabArgs.deployTokenAuth, "deploy-token-auth", false, "when enabled, a Project Deploy Token is generated and will be used instead of the SSH deploy token")

	bootstrapCmd.AddCommand(bootstrapGitLabCmd)
}

func bootstrapGitLabCmdRun(cmd *cobra.Command, args []string) error {
	glToken := os.Getenv(glTokenEnvVar)
	if glToken == "" {
		var err error
		glToken, err = readPasswordFromStdin("Please enter your GitLab personal access token (PAT): ")
		if err != nil {
			return fmt.Errorf("could not read token: %w", err)
		}
	}

	if projectNameIsValid, err := regexp.MatchString(gitlabProjectRegex, gitlabArgs.repository); err != nil || !projectNameIsValid {
		if err == nil {
			err = fmt.Errorf("%s is an invalid project name for gitlab.\nIt can contain only letters, digits, emojis, '_', '.', dash, space. It must start with letter, digit, emoji or '_'.", gitlabArgs.repository)
		}
		return err
	}

	if bootstrapArgs.tokenAuth && gitlabArgs.deployTokenAuth {
		return fmt.Errorf("--token-auth and --deploy-token-auth cannot be set both.")
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

	var caBundle []byte
	if bootstrapArgs.caFile != "" {
		var err error
		caBundle, err = os.ReadFile(bootstrapArgs.caFile)
		if err != nil {
			return fmt.Errorf("unable to read TLS CA file: %w", err)
		}
	}

	// Build GitLab provider
	providerCfg := provider.Config{
		Provider: provider.GitProviderGitLab,
		Hostname: gitlabArgs.hostname,
		Token:    glToken,
		CaBundle: caBundle,
	}
	// Workaround for: https://github.com/fluxcd/go-git-providers/issues/55
	if hostname := providerCfg.Hostname; hostname != glDefaultDomain &&
		!strings.HasPrefix(hostname, "https://") &&
		!strings.HasPrefix(hostname, "http://") {
		providerCfg.Hostname = "https://" + providerCfg.Hostname
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
		Username:  gitlabArgs.owner,
		Password:  glToken,
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
		TargetPath:             gitlabArgs.path.ToSlash(),
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
		TargetPath:   gitlabArgs.path.String(),
		ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
	}
	if bootstrapArgs.tokenAuth {
		secretOpts.Username = "git"
		secretOpts.Password = glToken
		secretOpts.CAFile = caBundle
	} else if gitlabArgs.deployTokenAuth {
		// the actual deploy token will be reconciled later
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

		secretOpts.SSHHostname = gitlabArgs.hostname
		if bootstrapArgs.sshHostname != "" {
			secretOpts.SSHHostname = bootstrapArgs.sshHostname
		}
	}

	// Sync manifest config
	syncOpts := sync.Options{
		Interval:          gitlabArgs.interval,
		Name:              *kubeconfigArgs.Namespace,
		Namespace:         *kubeconfigArgs.Namespace,
		Branch:            bootstrapArgs.branch,
		Secret:            bootstrapArgs.secretName,
		TargetPath:        gitlabArgs.path.ToSlash(),
		ManifestFile:      sync.MakeDefaultOptions().ManifestFile,
		RecurseSubmodules: bootstrapArgs.recurseSubmodules,
	}

	entityList, err := bootstrap.LoadEntityListFromPath(bootstrapArgs.gpgKeyRingPath)
	if err != nil {
		return err
	}

	// Bootstrap config
	bootstrapOpts := []bootstrap.GitProviderOption{
		bootstrap.WithProviderRepository(gitlabArgs.owner, gitlabArgs.repository, gitlabArgs.personal),
		bootstrap.WithBranch(bootstrapArgs.branch),
		bootstrap.WithBootstrapTransportType("https"),
		bootstrap.WithSignature(bootstrapArgs.authorName, bootstrapArgs.authorEmail),
		bootstrap.WithCommitMessageAppendix(bootstrapArgs.commitMessageAppendix),
		bootstrap.WithProviderTeamPermissions(mapTeamSlice(gitlabArgs.teams, glDefaultPermission)),
		bootstrap.WithReadWriteKeyPermissions(gitlabArgs.readWriteKey),
		bootstrap.WithKubeconfig(kubeconfigArgs, kubeclientOptions),
		bootstrap.WithLogger(logger),
		bootstrap.WithGitCommitSigning(entityList, bootstrapArgs.gpgPassphrase, bootstrapArgs.gpgKeyID),
	}
	if bootstrapArgs.sshHostname != "" {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithSSHHostname(bootstrapArgs.sshHostname))
	}
	if bootstrapArgs.tokenAuth || gitlabArgs.deployTokenAuth {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithSyncTransportType("https"))
	}
	if gitlabArgs.deployTokenAuth {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithDeployTokenAuth())
	}
	if !gitlabArgs.private {
		bootstrapOpts = append(bootstrapOpts, bootstrap.WithProviderRepositoryConfig("", "", "public"))
	}
	if gitlabArgs.reconcile {
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
