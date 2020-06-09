package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v32/github"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
)

var bootstrapGitHubCmd = &cobra.Command{
	Use:   "github",
	Short: "Bootstrap GitHub repository",
	Long: `
The bootstrap command creates the GitHub repository if it doesn't exists and
commits the toolkit components manifests to the master branch.
Then it configure the target cluster to synchronize with the repository.
If the toolkit components are present on the cluster,
the bootstrap command will perform an upgrade if needed.`,
	Example: `  # Create a GitHub personal access token and export it as an env var
  export GITHUB_TOKEN=<my-token>

  # Run bootstrap for a private repo owned by a GitHub organization
  bootstrap github --owner=<organization> --repository=<repo name>

  # Run bootstrap for a public repository on a personal account
  bootstrap github --owner=<user> --repository=<repo name> --private=false --personal=true 

  # Run bootstrap for a private repo hosted on GitHub Enterprise
  bootstrap github --owner=<organization> --repository=<repo name> --hostname=<domain>
`,
	RunE: bootstrapGitHubCmdRun,
}

var (
	ghOwner      string
	ghRepository string
	ghInterval   time.Duration
	ghPersonal   bool
	ghPrivate    bool
	ghHostname   string
)

const (
	ghTokenName             = "GITHUB_TOKEN"
	ghBranch                = "master"
	ghInstallManifest       = "toolkit.yaml"
	ghSourceManifest        = "toolkit-source.yaml"
	ghKustomizationManifest = "toolkit-kustomization.yaml"
	ghDefaultHostname       = "github.com"
)

func init() {
	bootstrapGitHubCmd.Flags().StringVar(&ghOwner, "owner", "", "GitHub user or organization name")
	bootstrapGitHubCmd.Flags().StringVar(&ghRepository, "repository", "", "GitHub repository name")
	bootstrapGitHubCmd.Flags().BoolVar(&ghPersonal, "personal", false, "is personal repository")
	bootstrapGitHubCmd.Flags().BoolVar(&ghPrivate, "private", true, "is private repository")
	bootstrapGitHubCmd.Flags().DurationVar(&ghInterval, "interval", time.Minute, "sync interval")
	bootstrapGitHubCmd.Flags().StringVar(&ghHostname, "hostname", ghDefaultHostname, "GitHub hostname")
	bootstrapCmd.AddCommand(bootstrapGitHubCmd)
}

func bootstrapGitHubCmdRun(cmd *cobra.Command, args []string) error {
	ghToken := os.Getenv(ghTokenName)
	if ghToken == "" {
		return fmt.Errorf("%s environment variable not found", ghTokenName)
	}

	ghURL := fmt.Sprintf("https://%s/%s/%s", ghHostname, ghOwner, ghRepository)
	sshURL := fmt.Sprintf("ssh://git@%s/%s/%s", ghHostname, ghOwner, ghRepository)
	if ghOwner == "" || ghRepository == "" {
		return fmt.Errorf("owner and repository are required")
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

	// create GitHub repository if doesn't exists
	logAction("connecting to %s", ghHostname)
	if err := createGitHubRepository(ctx, ghHostname, ghOwner, ghRepository, ghToken, ghPrivate, ghPersonal); err != nil {
		return err
	}

	// clone repository and checkout the master branch
	repo, err := checkoutGitHubRepository(ctx, ghURL, ghBranch, ghToken, tmpDir)
	if err != nil {
		return err
	}
	logSuccess("repository cloned")

	// generate install manifests
	logGenerate("generating manifests")
	manifest, err := generateGitHubInstall(namespace, tmpDir)
	if err != nil {
		return err
	}

	// stage install manifests
	changed, err := commitGitHubManifests(repo, namespace)
	if err != nil {
		return err
	}

	// push install manifests
	if changed {
		if err := pushGitHubRepository(ctx, repo, ghToken); err != nil {
			return err
		}
		logSuccess("components manifests pushed")
	} else {
		logSuccess("components are up to date")
	}

	// determine if repo synchronization is working
	isInstall := shouldInstallGitHub(ctx, kubeClient, namespace)

	if isInstall {
		// apply install manifests
		logAction("installing components in %s namespace", namespace)
		command := fmt.Sprintf("kubectl apply -f %s", manifest)
		if _, err := utils.execCommand(ctx, ModeOS, command); err != nil {
			return fmt.Errorf("install failed")
		}
		logSuccess("install completed")

		// check installation
		logWaiting("verifying installation")
		for _, deployment := range components {
			command = fmt.Sprintf("kubectl -n %s rollout status deployment %s --timeout=%s",
				namespace, deployment, timeout.String())
			if _, err := utils.execCommand(ctx, ModeOS, command); err != nil {
				return fmt.Errorf("install failed")
			} else {
				logSuccess("%s ready", deployment)
			}
		}
	}

	// setup SSH deploy key
	if shouldCreateGitHubDeployKey(ctx, kubeClient, namespace) {
		logAction("configuring deploy key")
		u, err := url.Parse(sshURL)
		if err != nil {
			return fmt.Errorf("git URL parse failed: %w", err)
		}

		key, err := generateGitHubDeployKey(ctx, kubeClient, u, namespace)
		if err != nil {
			return fmt.Errorf("generating deploy key failed: %w", err)
		}

		if err := createGitHubDeployKey(ctx, key, ghHostname, ghOwner, ghRepository, ghToken, ghPersonal); err != nil {
			return nil
		}
		logSuccess("deploy key configured")
	}

	// configure repo synchronization
	if isInstall {
		// generate source and kustomization manifests
		logAction("generating sync manifests")
		if err := generateGitHubKustomization(sshURL, namespace, namespace, tmpDir, ghInterval); err != nil {
			return err
		}

		// stage manifests
		changed, err = commitGitHubManifests(repo, namespace)
		if err != nil {
			return err
		}

		// push manifests
		if changed {
			if err := pushGitHubRepository(ctx, repo, ghToken); err != nil {
				return err
			}
		}
		logSuccess("sync manifests pushed")

		// apply manifests and waiting for sync
		logAction("applying sync manifests")
		if err := applyGitHubKustomization(ctx, kubeClient, namespace, namespace, tmpDir); err != nil {
			return err
		}
	}

	logSuccess("bootstrap finished")
	return nil
}

func makeGitHubClient(hostname, token string) (*github.Client, error) {
	auth := github.BasicAuthTransport{
		Username: "git",
		Password: token,
	}

	gh := github.NewClient(auth.Client())
	if hostname != ghDefaultHostname {
		baseURL := fmt.Sprintf("https://%s/api/v3/", hostname)
		uploadURL := fmt.Sprintf("https://%s/api/uploads/", hostname)
		if g, err := github.NewEnterpriseClient(baseURL, uploadURL, auth.Client()); err == nil {
			gh = g
		} else {
			return nil, fmt.Errorf("github client error: %w", err)
		}
	}

	return gh, nil
}

func createGitHubRepository(ctx context.Context, hostname, owner, name, token string, isPrivate, isPersonal bool) error {
	gh, err := makeGitHubClient(hostname, token)
	if err != nil {
		return err
	}
	org := ""
	if !isPersonal {
		org = owner
	}

	if _, _, err := gh.Repositories.Get(ctx, org, name); err == nil {
		return nil
	}

	autoInit := true
	_, _, err = gh.Repositories.Create(ctx, org, &github.Repository{
		AutoInit: &autoInit,
		Name:     &name,
		Private:  &isPrivate,
	})
	if err != nil {
		if !strings.Contains(err.Error(), "name already exists on this account") {
			return fmt.Errorf("github create repository error: %w", err)
		}
	} else {
		logSuccess("repository created")
	}
	return nil
}

func checkoutGitHubRepository(ctx context.Context, url, branch, token, path string) (*git.Repository, error) {
	auth := &http.BasicAuth{
		Username: "git",
		Password: token,
	}
	repo, err := git.PlainCloneContext(ctx, path, false, &git.CloneOptions{
		URL:           url,
		Auth:          auth,
		RemoteName:    git.DefaultRemoteName,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		NoCheckout:    false,
		Progress:      nil,
		Tags:          git.NoTags,
	})
	if err != nil {
		return nil, fmt.Errorf("git clone error: %w", err)
	}

	_, err = repo.Head()
	if err != nil {
		return nil, fmt.Errorf("git resolve HEAD error: %w", err)
	}

	return repo, nil
}

func generateGitHubInstall(namespace, tmpDir string) (string, error) {
	tkDir := path.Join(tmpDir, ".tk")
	defer os.RemoveAll(tkDir)

	if err := os.MkdirAll(tkDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("generating manifests failed: %w", err)
	}

	if err := genInstallManifests(bootstrapVersion, namespace, components, tkDir); err != nil {
		return "", fmt.Errorf("generating manifests failed: %w", err)
	}

	manifestsDir := path.Join(tmpDir, namespace)
	if err := os.MkdirAll(manifestsDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("generating manifests failed: %w", err)
	}

	manifest := path.Join(manifestsDir, ghInstallManifest)
	if err := buildKustomization(tkDir, manifest); err != nil {
		return "", fmt.Errorf("build kustomization failed: %w", err)
	}

	return manifest, nil
}

func commitGitHubManifests(repo *git.Repository, namespace string) (bool, error) {
	w, err := repo.Worktree()
	if err != nil {
		return false, err
	}

	_, err = w.Add(namespace)
	if err != nil {
		return false, err
	}

	status, err := w.Status()
	if err != nil {
		return false, err
	}

	if !status.IsClean() {
		if _, err := w.Commit("Add manifests", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "tk",
				Email: "tk@users.noreply.github.com",
				When:  time.Now(),
			},
		}); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func pushGitHubRepository(ctx context.Context, repo *git.Repository, token string) error {
	auth := &http.BasicAuth{
		Username: "git",
		Password: token,
	}
	err := repo.PushContext(ctx, &git.PushOptions{
		Auth:     auth,
		Progress: nil,
	})
	if err != nil {
		return fmt.Errorf("git push error: %w", err)
	}
	return nil
}

func generateGitHubKustomization(url, name, namespace, tmpDir string, interval time.Duration) error {
	gvk := sourcev1.GroupVersion.WithKind("GitRepository")
	gitRepository := sourcev1.GitRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: sourcev1.GitRepositorySpec{
			URL: url,
			Interval: metav1.Duration{
				Duration: interval,
			},
			Reference: &sourcev1.GitRepositoryRef{
				Branch: "master",
			},
			SecretRef: &corev1.LocalObjectReference{
				Name: name,
			},
		},
	}

	gitData, err := yaml.Marshal(gitRepository)
	if err != nil {
		return err
	}

	if err := utils.writeFile(string(gitData), filepath.Join(tmpDir, namespace, ghSourceManifest)); err != nil {
		return err
	}

	gvk = kustomizev1.GroupVersion.WithKind("Kustomization")
	emptyAPIGroup := ""
	kustomization := kustomizev1.Kustomization{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kustomizev1.KustomizationSpec{
			Interval: metav1.Duration{
				Duration: 10 * time.Minute,
			},
			Path:  "./",
			Prune: true,
			SourceRef: corev1.TypedLocalObjectReference{
				APIGroup: &emptyAPIGroup,
				Kind:     "GitRepository",
				Name:     name,
			},
		},
	}

	ksData, err := yaml.Marshal(kustomization)
	if err != nil {
		return err
	}

	if err := utils.writeFile(string(ksData), filepath.Join(tmpDir, namespace, ghKustomizationManifest)); err != nil {
		return err
	}

	return nil
}

func applyGitHubKustomization(ctx context.Context, kubeClient client.Client, name, namespace, tmpDir string) error {
	command := fmt.Sprintf("kubectl apply -f %s", filepath.Join(tmpDir, namespace))
	if _, err := utils.execCommand(ctx, ModeStderrOS, command); err != nil {
		return err
	}

	logWaiting("waiting for cluster sync")
	if err := wait.PollImmediate(pollInterval, timeout,
		isKustomizationReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	return nil
}

func shouldInstallGitHub(ctx context.Context, kubeClient client.Client, namespace string) bool {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      namespace,
	}
	var kustomization kustomizev1.Kustomization
	if err := kubeClient.Get(ctx, namespacedName, &kustomization); err != nil {
		return true
	}

	return kustomization.Status.LastAppliedRevision == ""
}

func shouldCreateGitHubDeployKey(ctx context.Context, kubeClient client.Client, namespace string) bool {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      namespace,
	}

	var existing corev1.Secret
	if err := kubeClient.Get(ctx, namespacedName, &existing); err != nil {
		return true
	}
	return false
}

func generateGitHubDeployKey(ctx context.Context, kubeClient client.Client, url *url.URL, namespace string) (string, error) {
	pair, err := generateKeyPair(ctx)
	if err != nil {
		return "", err
	}

	hostKey, err := scanHostKey(ctx, url)
	if err != nil {
		return "", err
	}

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"identity":     string(pair.PrivateKey),
			"identity.pub": string(pair.PublicKey),
			"known_hosts":  string(hostKey),
		},
	}
	if err := upsertSecret(ctx, kubeClient, secret); err != nil {
		return "", err
	}

	return string(pair.PublicKey), nil
}

func createGitHubDeployKey(ctx context.Context, key, hostname, owner, name, token string, isPersonal bool) error {
	gh, err := makeGitHubClient(hostname, token)
	if err != nil {
		return err
	}
	keyName := fmt.Sprintf("tk-%s", namespace)
	org := ""
	if !isPersonal {
		org = owner
	}
	isReadOnly := true
	_, _, err = gh.Repositories.CreateKey(ctx, org, name, &github.Key{
		Title:    &keyName,
		Key:      &key,
		ReadOnly: &isReadOnly,
	})
	if err != nil {
		return fmt.Errorf("github create deploy key error: %w", err)
	}
	return nil
}
