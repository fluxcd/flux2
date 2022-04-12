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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

	"github.com/fluxcd/flux2/internal/bootstrap"
	"github.com/fluxcd/flux2/internal/bootstrap/git/gogit"
	"github.com/fluxcd/flux2/internal/flags"
	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/pkg/manifestgen/sync"
)

var bootstrapGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Bootstrap toolkit components in a Git repository",
	Long: `The bootstrap git command commits the toolkit components manifests to the
branch of a Git repository. It then configures the target cluster to synchronize with
the repository. If the toolkit components are present on the cluster, the bootstrap
command will perform an upgrade if needed.`,
	Example: `  # Run bootstrap for a Git repository and authenticate with your SSH agent
  flux bootstrap git --url=ssh://git@example.com/repository.git

  # Run bootstrap for a Git repository and authenticate using a password
  flux bootstrap git --url=https://example.com/repository.git --password=<password>

  # Run bootstrap for a Git repository and authenticate using a password from environment variable
  GIT_PASSWORD=<password> && flux bootstrap git --url=https://example.com/repository.git

  # Run bootstrap for a Git repository with a passwordless private key
  flux bootstrap git --url=ssh://git@example.com/repository.git --private-key-file=<path/to/private.key>

  # Run bootstrap for a Git repository with a private key and password
  flux bootstrap git --url=ssh://git@example.com/repository.git --private-key-file=<path/to/private.key> --password=<password>
`,
	RunE: bootstrapGitCmdRun,
}

type gitFlags struct {
	url      string
	interval time.Duration
	path     flags.SafeRelativePath
	username string
	password string
	silent   bool
}

const (
	gitPasswordEnvVar = "GIT_PASSWORD"
)

var gitArgs gitFlags

func init() {
	bootstrapGitCmd.Flags().StringVar(&gitArgs.url, "url", "", "Git repository URL")
	bootstrapGitCmd.Flags().DurationVar(&gitArgs.interval, "interval", time.Minute, "sync interval")
	bootstrapGitCmd.Flags().Var(&gitArgs.path, "path", "path relative to the repository root, when specified the cluster sync will be scoped to this path")
	bootstrapGitCmd.Flags().StringVarP(&gitArgs.username, "username", "u", "git", "basic authentication username")
	bootstrapGitCmd.Flags().StringVarP(&gitArgs.password, "password", "p", "", "basic authentication password")
	bootstrapGitCmd.Flags().BoolVarP(&gitArgs.silent, "silent", "s", false, "assumes the deploy key is already setup, skips confirmation")

	bootstrapCmd.AddCommand(bootstrapGitCmd)
}

func bootstrapGitCmdRun(cmd *cobra.Command, args []string) error {
	gitPassword := os.Getenv(gitPasswordEnvVar)
	if gitPassword != "" && gitArgs.password == "" {
		gitArgs.password = gitPassword
	}
	if bootstrapArgs.tokenAuth && gitArgs.password == "" {
		var err error
		gitPassword, err = readPasswordFromStdin("Please enter your Git repository password: ")
		if err != nil {
			return fmt.Errorf("could not read token: %w", err)
		}
		gitArgs.password = gitPassword
	}

	if err := bootstrapValidate(); err != nil {
		return err
	}

	repositoryURL, err := url.Parse(gitArgs.url)
	if err != nil {
		return err
	}
	gitAuth, err := transportForURL(repositoryURL)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
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

	// Lazy go-git repository
	tmpDir, err := os.MkdirTemp("", "flux-bootstrap-")
	if err != nil {
		return fmt.Errorf("failed to create temporary working dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	gitClient := gogit.New(tmpDir, gitAuth)

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
		TargetPath:             gitArgs.path.ToSlash(),
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
		TargetPath:   gitArgs.path.String(),
		ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
	}
	if bootstrapArgs.tokenAuth {
		secretOpts.Username = gitArgs.username
		secretOpts.Password = gitArgs.password

		if bootstrapArgs.caFile != "" {
			secretOpts.CAFilePath = bootstrapArgs.caFile
		}

		// Remove port of the given host when not syncing over HTTP/S to not assume port for protocol
		// This _might_ be overwritten later on by e.g. --ssh-hostname
		if repositoryURL.Scheme != "https" && repositoryURL.Scheme != "http" {
			repositoryURL.Host = repositoryURL.Hostname()
		}

		// Configure repository URL to match auth config for sync.
		repositoryURL.User = nil
		repositoryURL.Scheme = "https"
	} else {
		secretOpts.PrivateKeyAlgorithm = sourcesecret.PrivateKeyAlgorithm(bootstrapArgs.keyAlgorithm)
		secretOpts.Password = gitArgs.password
		secretOpts.RSAKeyBits = int(bootstrapArgs.keyRSABits)
		secretOpts.ECDSACurve = bootstrapArgs.keyECDSACurve.Curve

		// Configure repository URL to match auth config for sync

		// Override existing user when user is not already set
		// or when a username was passed in
		if repositoryURL.User == nil || gitArgs.username != "git" {
			repositoryURL.User = url.User(gitArgs.username)
		}

		repositoryURL.Scheme = "ssh"
		if bootstrapArgs.sshHostname != "" {
			repositoryURL.Host = bootstrapArgs.sshHostname
		}
		if bootstrapArgs.privateKeyFile != "" {
			secretOpts.PrivateKeyPath = bootstrapArgs.privateKeyFile
		}

		// Configure last as it depends on the config above.
		secretOpts.SSHHostname = repositoryURL.Host
	}

	// Sync manifest config
	syncOpts := sync.Options{
		Interval:          gitArgs.interval,
		Name:              *kubeconfigArgs.Namespace,
		Namespace:         *kubeconfigArgs.Namespace,
		URL:               repositoryURL.String(),
		Branch:            bootstrapArgs.branch,
		Secret:            bootstrapArgs.secretName,
		TargetPath:        gitArgs.path.ToSlash(),
		ManifestFile:      sync.MakeDefaultOptions().ManifestFile,
		GitImplementation: sourceGitArgs.gitImplementation.String(),
		RecurseSubmodules: bootstrapArgs.recurseSubmodules,
	}

	var caBundle []byte
	if bootstrapArgs.caFile != "" {
		var err error
		caBundle, err = os.ReadFile(bootstrapArgs.caFile)
		if err != nil {
			return fmt.Errorf("unable to read TLS CA file: %w", err)
		}
	}

	// Bootstrap config
	bootstrapOpts := []bootstrap.GitOption{
		bootstrap.WithRepositoryURL(gitArgs.url),
		bootstrap.WithBranch(bootstrapArgs.branch),
		bootstrap.WithAuthor(bootstrapArgs.authorName, bootstrapArgs.authorEmail),
		bootstrap.WithCommitMessageAppendix(bootstrapArgs.commitMessageAppendix),
		bootstrap.WithKubeconfig(kubeconfigArgs, kubeclientOptions),
		bootstrap.WithPostGenerateSecretFunc(promptPublicKey),
		bootstrap.WithLogger(logger),
		bootstrap.WithCABundle(caBundle),
		bootstrap.WithGitCommitSigning(bootstrapArgs.gpgKeyRingPath, bootstrapArgs.gpgPassphrase, bootstrapArgs.gpgKeyID),
	}

	// Setup bootstrapper with constructed configs
	b, err := bootstrap.NewPlainGitProvider(gitClient, kubeClient, bootstrapOpts...)
	if err != nil {
		return err
	}

	// Run
	return bootstrap.Run(ctx, b, manifestsBase, installOptions, secretOpts, syncOpts, rootArgs.pollInterval, rootArgs.timeout)
}

// transportForURL constructs a transport.AuthMethod based on the scheme
// of the given URL and the configured flags. If the protocol equals
// "ssh" but no private key is configured, authentication using the local
// SSH-agent is attempted.
func transportForURL(u *url.URL) (transport.AuthMethod, error) {
	switch u.Scheme {
	case "https":
		return &http.BasicAuth{
			Username: gitArgs.username,
			Password: gitArgs.password,
		}, nil
	case "ssh":
		if bootstrapArgs.privateKeyFile != "" {
			return ssh.NewPublicKeysFromFile(u.User.Username(), bootstrapArgs.privateKeyFile, gitArgs.password)
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("scheme %q is not supported", u.Scheme)
	}
}

func promptPublicKey(ctx context.Context, secret corev1.Secret, _ sourcesecret.Options) error {
	ppk, ok := secret.StringData[sourcesecret.PublicKeySecretKey]
	if !ok {
		return nil
	}

	logger.Successf("public key: %s", strings.TrimSpace(ppk))

	if !gitArgs.silent {
		prompt := promptui.Prompt{
			Label:     "Please give the key access to your repository",
			IsConfirm: true,
		}
		_, err := prompt.Run()
		if err != nil {
			return fmt.Errorf("aborting")
		}
	}
	return nil
}
