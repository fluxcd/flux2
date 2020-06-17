package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

var bootstrapGitLabCmd = &cobra.Command{
	Use:   "gitlab",
	Short: "Bootstrap GitLab repository",
	Long: `
The bootstrap command creates the GitHub repository if it doesn't exists and
commits the toolkit components manifests to the master branch.
Then it configure the target cluster to synchronize with the repository.
If the toolkit components are present on the cluster,
the bootstrap command will perform an upgrade if needed.`,
	Example: `  # Create a GitLab personal access token and export it as an env var
  export GITLAB_TOKEN=<my-token>

  # Run bootstrap for a private repo owned by a GitLab organization
  bootstrap gitlab --owner=<organization> --repository=<repo name>

  # Run bootstrap for a private repo hosted on GitLab server 
  bootstrap gitlab --owner=<organization> --repository=<repo name> --hostname=<domain>
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

const (
	glTokenName       = "GITLAB_TOKEN"
	glDefaultHostname = "gitlab.com"
)

func init() {
	bootstrapGitLabCmd.Flags().StringVar(&glOwner, "owner", "", "GitLab user or organization name")
	bootstrapGitLabCmd.Flags().StringVar(&glRepository, "repository", "", "GitLab repository name")
	bootstrapGitLabCmd.Flags().BoolVar(&glPersonal, "personal", false, "is personal repository")
	bootstrapGitLabCmd.Flags().BoolVar(&glPrivate, "private", true, "is private repository")
	bootstrapGitLabCmd.Flags().DurationVar(&glInterval, "interval", time.Minute, "sync interval")
	bootstrapGitLabCmd.Flags().StringVar(&glHostname, "hostname", glDefaultHostname, "GitLab hostname")
	bootstrapGitLabCmd.Flags().StringVar(&glPath, "path", "", "repository path, when specified the cluster sync will be scoped to this path")

	bootstrapCmd.AddCommand(bootstrapGitLabCmd)
}

func bootstrapGitLabCmdRun(cmd *cobra.Command, args []string) error {
	glToken := os.Getenv(glTokenName)
	if glToken == "" {
		return fmt.Errorf("%s environment variable not found", glTokenName)
	}

	gitURL := fmt.Sprintf("https://%s/%s/%s", glHostname, glOwner, glRepository)
	sshURL := fmt.Sprintf("ssh://git@%s/%s/%s", glHostname, glOwner, glRepository)
	if glOwner == "" || glRepository == "" {
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

	// create GitLab project if doesn't exists
	logAction("connecting to %s", glHostname)
	if err := createGitLabRepository(ctx, glHostname, glOwner, glRepository, glToken, glPrivate, glPersonal); err != nil {
		return err
	}

	// clone repository and checkout the master branch
	repo, err := checkoutGitHubRepository(ctx, gitURL, ghBranch, glToken, tmpDir)
	if err != nil {
		return err
	}
	logSuccess("repository cloned")

	// generate install manifests
	logGenerate("generating manifests")
	manifest, err := generateGitHubInstall(glPath, namespace, tmpDir)
	if err != nil {
		return err
	}

	// stage install manifests
	changed, err := commitGitHubManifests(repo, glPath, namespace)
	if err != nil {
		return err
	}

	if changed {
		if err := pushGitHubRepository(ctx, repo, glToken); err != nil {
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

		if err := createGitLabDeployKey(ctx, key, glHostname, glOwner, glRepository, glPath, glToken); err != nil {
			return err
		}
		logSuccess("deploy key configured")
	}

	// configure repo synchronization
	if isInstall {
		// generate source and kustomization manifests
		logAction("generating sync manifests")
		if err := generateGitHubKustomization(sshURL, namespace, namespace, glPath, tmpDir, glInterval); err != nil {
			return err
		}

		// stage manifests
		changed, err = commitGitHubManifests(repo, glPath, namespace)
		if err != nil {
			return err
		}

		// push manifests
		if changed {
			if err := pushGitHubRepository(ctx, repo, glToken); err != nil {
				return err
			}
		}
		logSuccess("sync manifests pushed")

		// apply manifests and waiting for sync
		logAction("applying sync manifests")
		if err := applyGitHubKustomization(ctx, kubeClient, namespace, namespace, glPath, tmpDir); err != nil {
			return err
		}
	}

	logSuccess("bootstrap finished")
	return nil
}

func makeGitLabClient(hostname, token string) (*gitlab.Client, error) {
	gl, err := gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}

	if glHostname != glDefaultHostname {
		gl, err = gitlab.NewClient(token, gitlab.WithBaseURL(fmt.Sprintf("https://%s/api/v4", hostname)))
		if err != nil {
			return nil, err
		}
	}
	return gl, nil
}

func createGitLabRepository(ctx context.Context, hostname, owner, repository, token string, isPrivate, isPersonal bool) error {
	gl, err := makeGitLabClient(hostname, token)
	if err != nil {
		return fmt.Errorf("client error: %w", err)
	}

	var id *int
	if !isPersonal {
		groups, _, err := gl.Groups.ListGroups(&gitlab.ListGroupsOptions{Search: gitlab.String(owner)}, gitlab.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("list groups error: %w", err)
		}

		if len(groups) > 0 {
			id = &groups[0].ID
		}
	}

	visibility := gitlab.PublicVisibility
	if isPrivate {
		visibility = gitlab.PrivateVisibility
	}

	projects, _, err := gl.Projects.ListProjects(&gitlab.ListProjectsOptions{Search: gitlab.String(repository)}, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("list projects error: %w", err)
	}

	if len(projects) == 0 {
		p := &gitlab.CreateProjectOptions{
			Name:                 gitlab.String(repository),
			NamespaceID:          id,
			Visibility:           &visibility,
			InitializeWithReadme: gitlab.Bool(true),
		}

		project, _, err := gl.Projects.CreateProject(p)
		if err != nil {
			return fmt.Errorf("create project error: %w", err)
		}
		logSuccess("project created id: %v", project.ID)
	}

	return nil
}

func createGitLabDeployKey(ctx context.Context, key, hostname, owner, repository, targetPath, token string) error {
	gl, err := makeGitLabClient(hostname, token)
	if err != nil {
		return fmt.Errorf("client error: %w", err)
	}

	var projId int
	projects, _, err := gl.Projects.ListProjects(&gitlab.ListProjectsOptions{Search: gitlab.String(repository)}, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("list projects error: %w", err)
	}
	if len(projects) > 0 {
		projId = projects[0].ID
	} else {
		return fmt.Errorf("no project found")
	}

	keyName := "tk"
	if targetPath != "" {
		keyName = fmt.Sprintf("tk-%s", targetPath)
	}

	// check if the key exists
	keys, _, err := gl.DeployKeys.ListProjectDeployKeys(projId, &gitlab.ListProjectDeployKeysOptions{})
	if err != nil {
		return fmt.Errorf("list keys error: %w", err)
	}

	shouldCreateKey := true
	var existingKey *gitlab.DeployKey
	for _, k := range keys {
		if k.Title == keyName {
			if k.Key != key {
				existingKey = k
			} else {
				shouldCreateKey = false
			}
			break
		}
	}

	// delete existing key if the value differs
	if existingKey != nil {
		_, err := gl.DeployKeys.DeleteDeployKey(projId, existingKey.ID, gitlab.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("delete key error: %w", err)
		}
	}

	// create key
	if shouldCreateKey {
		_, _, err := gl.DeployKeys.AddDeployKey(projId, &gitlab.AddDeployKeyOptions{
			Title:   gitlab.String(keyName),
			Key:     gitlab.String(key),
			CanPush: gitlab.Bool(false),
		}, gitlab.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("add key error: %w", err)
		}
	}

	return nil
}
