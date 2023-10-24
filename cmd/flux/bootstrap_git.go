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

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/bootstrap"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sync"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
)

var bootstrapGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Deploy Flux on a cluster connected to a Git repository",
	Long: `The bootstrap git command commits the Flux manifests to the
branch of a Git repository. And then it configures the target cluster to synchronize with
that repository. If the Flux components are present on the cluster, the bootstrap
command will perform an upgrade if needed.`,
	Example: `  # Run bootstrap for a Git repository and authenticate with your SSH agent
  flux bootstrap git --url=ssh://git@example.com/repository.git --path=clusters/my-cluster

  # Run bootstrap for a Git repository and authenticate using a password
  flux bootstrap git --url=https://example.com/repository.git --password=<password> --path=clusters/my-cluster

  # Run bootstrap for a Git repository and authenticate using a password from environment variable
  GIT_PASSWORD=<password> && flux bootstrap git --url=https://example.com/repository.git --path=clusters/my-cluster

  # Run bootstrap for a Git repository with a passwordless private key
  flux bootstrap git --url=ssh://git@example.com/repository.git --private-key-file=<path/to/private.key> --path=clusters/my-cluster

  # Run bootstrap for a Git repository with a private key and password
  flux bootstrap git --url=ssh://git@example.com/repository.git --private-key-file=<path/to/private.key> --password=<password> --path=clusters/my-cluster

  # Run bootstrap for a Git repository on AWS CodeCommit
  flux bootstrap git --url=ssh://<SSH-Key-ID>@git-codecommit.<region>.amazonaws.com/v1/repos/<repository> --private-key-file=<path/to/private.key> --password=<SSH-passphrase> --path=clusters/my-cluster

  # Run bootstrap for a Git repository on Azure Devops
  flux bootstrap git --url=ssh://git@ssh.dev.azure.com/v3/<org>/<project>/<repository> --ssh-key-algorithm=rsa --ssh-rsa-bits=4096 --path=clusters/my-cluster
`,
	RunE: bootstrapGitCmdRun,
}

type gitFlags struct {
	url                 string
	interval            time.Duration
	path                flags.SafeRelativePath
	username            string
	password            string
	silent              bool
	insecureHttpAllowed bool
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
	bootstrapGitCmd.Flags().BoolVar(&gitArgs.insecureHttpAllowed, "allow-insecure-http", false, "allows insecure HTTP connections")

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

	if strings.Contains(repositoryURL.Hostname(), "git-codecommit") && strings.Contains(repositoryURL.Hostname(), "amazonaws.com") {
		if repositoryURL.Scheme == string(git.SSH) {
			if repositoryURL.User == nil {
				return fmt.Errorf("invalid AWS CodeCommit url: ssh username should be specified in the url")
			}
			if repositoryURL.User.Username() == git.DefaultPublicKeyAuthUser {
				return fmt.Errorf("invalid AWS CodeCommit url: ssh username should be the SSH key ID for the provided private key")
			}
			if bootstrapArgs.privateKeyFile == "" {
				return fmt.Errorf("private key file is required for bootstrapping against AWS CodeCommit using ssh")
			}
		}
		if repositoryURL.Scheme == string(git.HTTPS) && !bootstrapArgs.tokenAuth {
			return fmt.Errorf("--token-auth=true must be specified for using an HTTPS AWS CodeCommit url")
		}
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

	// Lazy go-git repository
	tmpDir, err := manifestgen.MkdirTempAbs("", "flux-bootstrap-")
	if err != nil {
		return fmt.Errorf("failed to create temporary working dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var caBundle []byte
	if bootstrapArgs.caFile != "" {
		var err error
		caBundle, err = os.ReadFile(bootstrapArgs.caFile)
		if err != nil {
			return fmt.Errorf("unable to read TLS CA file: %w", err)
		}
	}
	authOpts, err := getAuthOpts(repositoryURL, caBundle)
	if err != nil {
		return fmt.Errorf("failed to create authentication options for %s: %w", repositoryURL.String(), err)
	}

	clientOpts := []gogit.ClientOption{gogit.WithDiskStorage(), gogit.WithFallbackToDefaultKnownHosts()}
	if gitArgs.insecureHttpAllowed {
		clientOpts = append(clientOpts, gogit.WithInsecureCredentialsOverHTTP())
	}
	gitClient, err := gogit.NewClient(tmpDir, authOpts, clientOpts...)
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
		secretOpts.CAFile = caBundle

		// Remove port of the given host when not syncing over HTTP/S to not assume port for protocol
		// This _might_ be overwritten later on by e.g. --ssh-hostname
		if repositoryURL.Scheme != "https" && repositoryURL.Scheme != "http" {
			repositoryURL.Host = repositoryURL.Hostname()
		}

		// Configure repository URL to match auth config for sync.
		repositoryURL.User = nil
		if !gitArgs.insecureHttpAllowed {
			repositoryURL.Scheme = "https"
		}
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

		keypair, err := sourcesecret.LoadKeyPairFromPath(bootstrapArgs.privateKeyFile, gitArgs.password)
		if err != nil {
			return err
		}
		secretOpts.Keypair = keypair

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
		RecurseSubmodules: bootstrapArgs.recurseSubmodules,
	}

	entityList, err := bootstrap.LoadEntityListFromPath(bootstrapArgs.gpgKeyRingPath)
	if err != nil {
		return err
	}

	// Bootstrap config
	bootstrapOpts := []bootstrap.GitOption{
		bootstrap.WithRepositoryURL(gitArgs.url),
		bootstrap.WithBranch(bootstrapArgs.branch),
		bootstrap.WithSignature(bootstrapArgs.authorName, bootstrapArgs.authorEmail),
		bootstrap.WithCommitMessageAppendix(bootstrapArgs.commitMessageAppendix),
		bootstrap.WithKubeconfig(kubeconfigArgs, kubeclientOptions),
		bootstrap.WithPostGenerateSecretFunc(promptPublicKey),
		bootstrap.WithLogger(logger),
		bootstrap.WithGitCommitSigning(entityList, bootstrapArgs.gpgPassphrase, bootstrapArgs.gpgKeyID),
	}

	// Setup bootstrapper with constructed configs
	b, err := bootstrap.NewPlainGitProvider(gitClient, kubeClient, bootstrapOpts...)
	if err != nil {
		return err
	}

	// Run
	return bootstrap.Run(ctx, b, manifestsBase, installOptions, secretOpts, syncOpts, rootArgs.pollInterval, rootArgs.timeout)
}

// getAuthOpts retruns a AuthOptions based on the scheme
// of the given URL and the configured flags. If the protocol equals
// "ssh" but no private key is configured, authentication using the local
// SSH-agent is attempted.
func getAuthOpts(u *url.URL, caBundle []byte) (*git.AuthOptions, error) {
	switch u.Scheme {
	case "http":
		if !gitArgs.insecureHttpAllowed {
			return nil, fmt.Errorf("scheme http is insecure, pass --allow-insecure-http=true to allow it")
		}
		return &git.AuthOptions{
			Transport: git.HTTP,
			Username:  gitArgs.username,
			Password:  gitArgs.password,
		}, nil
	case "https":
		return &git.AuthOptions{
			Transport: git.HTTPS,
			Username:  gitArgs.username,
			Password:  gitArgs.password,
			CAFile:    caBundle,
		}, nil
	case "ssh":
		authOpts := &git.AuthOptions{
			Transport: git.SSH,
			Username:  u.User.Username(),
			Password:  gitArgs.password,
		}
		if bootstrapArgs.privateKeyFile != "" {
			pk, err := os.ReadFile(bootstrapArgs.privateKeyFile)
			if err != nil {
				return nil, err
			}
			kh, err := sourcesecret.ScanHostKey(u.Host)
			if err != nil {
				return nil, err
			}
			authOpts.Identity = pk
			authOpts.KnownHosts = kh
		}
		return authOpts, nil
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
