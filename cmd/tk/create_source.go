package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var createSourceCmd = &cobra.Command{
	Use:   "source [name]",
	Short: "Create source resource",
	Long: `
The create source command generates a source.fluxcd.io resource and waits for it to sync.
For Git over SSH, host and SSH keys are automatically generated.`,
	Example: `  # Create a source from a public Git repository master branch
  create source podinfo --git-url https://github.com/stefanprodan/podinfo-deploy --git-branch master

  # Create a source from a public Git repository tag that matches a semver range
  create source podinfo --git-url https://github.com/stefanprodan/podinfo-deploy  --git-semver=">=0.0.1-rc.1 <0.1.0"

  #  Create a source from a Git repository using SSH authentication
  create source podinfo --git-url ssh://git@github.com/stefanprodan/podinfo-deploy

  # Create a source from a Git repository using basic authentication
  create source podinfo --git-url https://github.com/stefanprodan/podinfo-deploy -u username -p password
`,
	RunE: createSourceCmdRun,
}

var (
	sourceGitURL    string
	sourceGitBranch string
	sourceGitSemver string
	sourceUsername  string
	sourcePassword  string
)

func init() {
	createSourceCmd.Flags().StringVar(&sourceGitURL, "git-url", "", "git address, e.g. ssh://git@host/org/repository")
	createSourceCmd.Flags().StringVar(&sourceGitBranch, "git-branch", "master", "git branch")
	createSourceCmd.Flags().StringVar(&sourceGitSemver, "git-semver", "", "git tag semver range")
	createSourceCmd.Flags().StringVarP(&sourceUsername, "username", "u", "", "basic authentication username")
	createSourceCmd.Flags().StringVarP(&sourcePassword, "password", "p", "", "basic authentication password")

	createCmd.AddCommand(createSourceCmd)
}

func createSourceCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("source name is required")
	}
	name := args[0]

	if sourceGitURL == "" {
		return fmt.Errorf("git-url is required")
	}

	tmpDir, err := ioutil.TempDir("", name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	u, err := url.Parse(sourceGitURL)
	if err != nil {
		return fmt.Errorf("git URL parse failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	withAuth := false
	if strings.HasPrefix(sourceGitURL, "ssh") {
		if err := generateSSH(ctx, name, u.Host, tmpDir); err != nil {
			return err
		}
		withAuth = true
	} else if sourceUsername != "" && sourcePassword != "" {
		if err := generateBasicAuth(ctx, name); err != nil {
			return err
		}
		withAuth = true
	}

	logAction("generating source %s in %s namespace", name, namespace)

	gitRepository := sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: sourcev1.GitRepositorySpec{
			URL: sourceGitURL,
			Interval: metav1.Duration{
				Duration: interval,
			},
		},
	}

	if withAuth {
		gitRepository.Spec.SecretRef = &corev1.LocalObjectReference{
			Name: name,
		}
	}
	if sourceGitSemver != "" {
		gitRepository.Spec.Reference = &sourcev1.GitRepositoryRef{
			SemVer: sourceGitSemver,
		}
	} else {
		gitRepository.Spec.Reference = &sourcev1.GitRepositoryRef{
			Branch: sourceGitBranch,
		}
	}

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	err = kubeClient.Get(ctx, namespacedName, &gitRepository)
	if errors.IsNotFound(err) {
		if err := kubeClient.Create(ctx, &gitRepository); err != nil {
			return err
		}
	} else {
		if err := kubeClient.Update(ctx, &gitRepository); err != nil {
			return err
		}
	}

	logAction("waiting for source sync")
	if err := wait.PollImmediate(2*time.Second, timeout,
		isGitRepositoryReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logSuccess("source %s is ready", name)

	err = kubeClient.Get(ctx, namespacedName, &gitRepository)
	if err != nil {
		return fmt.Errorf("source sync failed: %w", err)
	}

	if gitRepository.Status.Artifact != nil {
		logSuccess("revision %s", gitRepository.Status.Artifact.Revision)
	} else {
		return fmt.Errorf("source sync failed, artifact not found")
	}

	return nil
}

func generateBasicAuth(ctx context.Context, name string) error {
	logAction("saving credentials")
	credentials := fmt.Sprintf("--from-literal=username='%s' --from-literal=password='%s'",
		sourceUsername, sourcePassword)
	secret := fmt.Sprintf("kubectl -n %s create secret generic %s %s --dry-run=client -oyaml | kubectl apply -f-",
		namespace, name, credentials)
	if _, err := utils.execCommand(ctx, ModeOS, secret); err != nil {
		return fmt.Errorf("kubectl create secret failed")
	}
	return nil
}

func generateSSH(ctx context.Context, name, host, tmpDir string) error {
	logAction("generating host key for %s", host)

	command := fmt.Sprintf("ssh-keyscan %s > %s/known_hosts", host, tmpDir)
	if _, err := utils.execCommand(ctx, ModeStderrOS, command); err != nil {
		return fmt.Errorf("ssh-keyscan failed")
	}

	logAction("generating deploy key")

	command = fmt.Sprintf("ssh-keygen -b 2048 -t rsa -f %s/identity -q -N \"\"", tmpDir)
	if _, err := utils.execCommand(ctx, ModeStderrOS, command); err != nil {
		return fmt.Errorf("ssh-keygen failed")
	}

	command = fmt.Sprintf("cat %s/identity.pub", tmpDir)
	if deployKey, err := utils.execCommand(ctx, ModeCapture, command); err != nil {
		return fmt.Errorf("unable to read identity.pub: %w", err)
	} else {
		fmt.Print(deployKey)
	}

	prompt := promptui.Prompt{
		Label:     "Have you added the deploy key to your repository",
		IsConfirm: true,
	}
	if _, err := prompt.Run(); err != nil {
		return fmt.Errorf("aborting")
	}

	logAction("saving deploy key")
	files := fmt.Sprintf("--from-file=%s/identity --from-file=%s/identity.pub --from-file=%s/known_hosts",
		tmpDir, tmpDir, tmpDir)
	secret := fmt.Sprintf("kubectl -n %s create secret generic %s %s --dry-run=client -oyaml | kubectl apply -f-",
		namespace, name, files)
	if _, err := utils.execCommand(ctx, ModeOS, secret); err != nil {
		return fmt.Errorf("create secret failed")
	}
	return nil
}

func isGitRepositoryReady(ctx context.Context, kubeClient client.Client, name, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		var gitRepository sourcev1.GitRepository
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := kubeClient.Get(ctx, namespacedName, &gitRepository)
		if err != nil {
			return false, err
		}

		for _, condition := range gitRepository.Status.Conditions {
			if condition.Type == sourcev1.ReadyCondition {
				if condition.Status == corev1.ConditionTrue {
					return true, nil
				} else if condition.Status == corev1.ConditionFalse {
					return false, fmt.Errorf(condition.Message)
				}
			}
		}
		return false, nil
	}
}
