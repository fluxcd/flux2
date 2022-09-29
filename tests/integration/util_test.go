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

package test

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	git2go "github.com/libgit2/git2go/v33"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	automationv1beta1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	reflectorv1beta1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	notiv1beta1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/libgit2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/fluxcd/test-infra/tftestenv"
)

const defaultBranch = "main"

// fluxConfig contains configuration for installing FLux in a cluster
type fluxConfig struct {
	kubeconfigPath string
	repoURL        string
	password       string
	objects        map[client.Object]controllerutil.MutateFn
	kustomizeYaml  string
}

func setupScheme() error {
	err := sourcev1.AddToScheme(scheme.Scheme)
	if err != nil {
		return err
	}
	err = kustomizev1.AddToScheme(scheme.Scheme)
	if err != nil {
		return err
	}
	err = helmv2beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return err
	}
	err = reflectorv1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return err
	}
	err = automationv1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return err
	}
	err = notiv1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return err
	}

	return nil
}

// installFlux adds the core Flux components to the cluster specified in the kubeconfig file.
func installFlux(ctx context.Context, kubeClient client.Client, conf fluxConfig) error {
	// Create flux-system namespace
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "flux-system",
		},
	}
	err := testEnv.Client.Create(ctx, &namespace)
	// create secret containing credentials for bootstrap repository
	httpsCredentials := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "https-credentials", Namespace: "flux-system"}}
	_, err = controllerutil.CreateOrUpdate(ctx, kubeClient, httpsCredentials, func() error {
		httpsCredentials.StringData = map[string]string{
			"username": "git",
			"password": conf.password,
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Create additional objects that are needed for flux to run correctly
	for obj, fn := range conf.objects {
		_, err = controllerutil.CreateOrUpdate(ctx, kubeClient, obj, fn)
		if err != nil {
			return err
		}
	}

	// Install Flux and push files to git repository
	gitClient, err := cloneRepository(conf.repoURL, "", conf.password)
	if err != nil {
		return err
	}
	repoDir := gitClient.Path()
	err = tftestenv.RunCommand(ctx, repoDir, "mkdir -p ./clusters/e2e/flux-system", tftestenv.RunCommandOptions{})
	if err != nil {
		return err
	}
	err = tftestenv.RunCommand(ctx, repoDir,
		"flux install --components-extra=\"image-reflector-controller,image-automation-controller\" --export > ./clusters/e2e/flux-system/gotk-components.yaml",
		tftestenv.RunCommandOptions{})
	if err != nil {
		return err
	}
	err = tftestenv.RunCommand(ctx, repoDir, fmt.Sprintf("flux create source git flux-system --git-implementation=libgit2 --url=%s --branch=%s --secret-ref=https-credentials --interval=1m  --export > ./clusters/e2e/flux-system/gotk-sync.yaml", conf.repoURL, defaultBranch),
		tftestenv.RunCommandOptions{})
	if err != nil {
		return err
	}
	err = tftestenv.RunCommand(ctx, repoDir, "flux create kustomization flux-system --source=flux-system --path='./clusters/e2e' --prune=true --interval=1m --export >> ./clusters/e2e/flux-system/gotk-sync.yaml", tftestenv.RunCommandOptions{})
	if err != nil {
		return err
	}
	kustomizeYaml := `
resources:
- gotk-components.yaml
- gotk-sync.yaml
`
	if conf.kustomizeYaml != "" {
		kustomizeYaml = conf.kustomizeYaml
	}

	err = tftestenv.RunCommand(ctx, gitClient.Path(), fmt.Sprintf("echo \"%s\" > ./clusters/e2e/flux-system/kustomization.yaml", kustomizeYaml), tftestenv.RunCommandOptions{})
	if err != nil {
		return err
	}

	// commit and push manifests
	err = commitAndPushAll(ctx, gitClient, defaultBranch, "Add sync manifests")
	if err != nil {
		return err
	}

	// Need to apply CRDs first to make sure that the sync resources will apply properly
	err = tftestenv.RunCommand(ctx, repoDir, fmt.Sprintf("kubectl --kubeconfig=%s apply -f ./clusters/e2e/flux-system/gotk-components.yaml", conf.kubeconfigPath), tftestenv.RunCommandOptions{})
	if err != nil {
		return err
	}
	err = tftestenv.RunCommand(ctx, repoDir, fmt.Sprintf("kubectl --kubeconfig=%s apply -k ./clusters/e2e/flux-system/", conf.kubeconfigPath), tftestenv.RunCommandOptions{})
	if err != nil {
		return err
	}
	return nil
}

// verifyGitAndKustomization checks that the gitrespository and kustomization combination are working properly.
func verifyGitAndKustomization(ctx context.Context, kubeClient client.Client, namespace, name string) error {
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	source := &sourcev1.GitRepository{}
	err := kubeClient.Get(ctx, nn, source)
	if err != nil {
		return err
	}
	if apimeta.IsStatusConditionPresentAndEqual(source.Status.Conditions, meta.ReadyCondition, metav1.ConditionTrue) == false {
		return fmt.Errorf("source condition not ready")
	}
	kustomization := &kustomizev1.Kustomization{}
	err = kubeClient.Get(ctx, nn, kustomization)
	if err != nil {
		return err
	}
	if apimeta.IsStatusConditionPresentAndEqual(kustomization.Status.Conditions, meta.ReadyCondition, metav1.ConditionTrue) == false {
		return fmt.Errorf("kustomization condition not ready")
	}
	return nil
}

type nsConfig struct {
	repoURL       string
	protocol      string
	objectName    string
	path          string
	modifyGitSpec func(spec *sourcev1.GitRepositorySpec)
	modifyKsSpec  func(spec *kustomizev1.KustomizationSpec)
}

// setupNamespaces creates the namespace, then creates the git secret,
// git repository and kustomization in that namespace
func setupNamespace(ctx context.Context, name string, opts nsConfig) error {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, testEnv.Client, &namespace, func() error {
		return nil
	})

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-credentials",
			Namespace: name,
		},
	}

	secretData := map[string]string{
		"username": "git",
		"password": cfg.pat,
	}
	if opts.protocol == "ssh" {
		secretData = map[string]string{
			"identity":     cfg.idRsa,
			"identity.pub": cfg.idRsaPub,
			"known_hosts":  cfg.knownHosts,
		}
	}

	_, err = controllerutil.CreateOrUpdate(ctx, testEnv.Client, &secret, func() error {
		secret.StringData = secretData
		return nil
	})

	gitSpec := &sourcev1.GitRepositorySpec{
		Interval: metav1.Duration{
			Duration: 1 * time.Minute,
		},
		Reference: &sourcev1.GitRepositoryRef{
			Branch: name,
		},
		SecretRef: &meta.LocalObjectReference{
			Name: secret.Name,
		},
		URL: opts.repoURL,
	}
	if infraOpts.Provider == "azure" {
		gitSpec.GitImplementation = sourcev1.LibGit2Implementation
	}
	if opts.modifyGitSpec != nil {
		opts.modifyGitSpec(gitSpec)
	}
	source := &sourcev1.GitRepository{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace.Name}}
	_, err = controllerutil.CreateOrUpdate(ctx, testEnv.Client, source, func() error {
		source.Spec = *gitSpec
		return nil
	})
	if err != nil {
		return err
	}

	ksSpec := &kustomizev1.KustomizationSpec{
		Path: opts.path,
		SourceRef: kustomizev1.CrossNamespaceSourceReference{
			Kind:      sourcev1.GitRepositoryKind,
			Name:      source.Name,
			Namespace: source.Namespace,
		},
		Interval: metav1.Duration{
			Duration: 1 * time.Minute,
		},
		Prune: true,
	}
	if opts.modifyKsSpec != nil {
		opts.modifyKsSpec(ksSpec)
	}
	kustomization := &kustomizev1.Kustomization{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace.Name}}
	_, err = controllerutil.CreateOrUpdate(ctx, testEnv.Client, kustomization, func() error {
		kustomization.Spec = *ksSpec
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func cloneRepository(url, branchName string, password string) (*libgit2.Client, error) {
	checkoutBranch := defaultBranch

	tmpDir, err := os.MkdirTemp("", "*-repository")
	if err != nil {
		return nil, err
	}

	client, err := libgit2.NewClient(tmpDir, &git.AuthOptions{
		Transport: git.HTTP,
		Username:  "git",
		Password:  password,
	})
	if err != nil {
		return nil, err
	}

	_, err = client.Clone(context.Background(), url, git.CheckoutOptions{
		Branch: checkoutBranch,
	})
	if err != nil {
		return nil, err
	}

	if branchName != "" && branchName != defaultBranch {
		if err := client.SwitchBranch(context.Background(), branchName); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func addFile(dir, path, content string) error {
	err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0777)
	if err != nil {
		return err
	}
	return nil
}

func commitAndPushAll(ctx context.Context, client *libgit2.Client, branch, message string) error {
	_, err := client.Commit(git.Commit{
		Author: git.Signature{
			Name:  "git",
			Email: "test@example.com",
		},
		Message: message,
	}, nil)
	// no staged files error occurs when we are reusing infrastructure
	// since the remote repository exists and is up to-date
	if err != nil && !strings.Contains(err.Error(), "no staged files") {
		return err
	}

	err = client.Push(ctx)
	if err != nil {
		return err
	}

	return nil
}

func createTagAndPush(client *libgit2.Client, branchName, tag, password string) error {
	repo, err := git2go.OpenRepository(client.Path())
	if err != nil {
		return nil
	}

	branch, err := repo.LookupBranch(branchName, git2go.BranchAll)
	if err != nil {
		return err
	}
	commit, err := repo.LookupCommit(branch.Target())
	if err != nil {
		return err
	}

	tags, err := repo.Tags.List()
	if err != nil {
		return err
	}
	for _, existingTag := range tags {
		if existingTag == tag {
			err = repo.Tags.Remove(tag)
			if err != nil {
				return err
			}
		}
	}

	sig := &git2go.Signature{
		Name:  "git",
		Email: "test@example.com",
		When:  time.Now(),
	}
	_, err = repo.Tags.Create(tag, commit, sig, "create tag")
	if err != nil {
		return err
	}
	origin, err := repo.Remotes.Lookup("origin")
	if err != nil {
		return err
	}
	err = origin.Push([]string{fmt.Sprintf("+refs/tags/%s", tag)}, &git2go.PushOptions{
		RemoteCallbacks: git2go.RemoteCallbacks{
			CredentialsCallback: credentialCallback("git", password),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func credentialCallback(username, password string) git2go.CredentialsCallback {
	return func(url string, usernameFromURL string, allowedTypes git2go.CredType) (*git2go.Cred, error) {
		cred, err := git2go.NewCredentialUserpassPlaintext(username, password)
		if err != nil {
			return nil, err
		}
		return cred, nil
	}
}

func executeTemplate(path string, templateValues map[string]string) (string, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	tmpl := template.Must(template.New("golden").Parse(string(buf)))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, templateValues); err != nil {
		return "", err
	}
	return out.String(), nil
}

func pushImagesFromURL(repoURL, imgURL string, tags []string) error {
	img, err := crane.Pull(imgURL)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		log.Printf("Pushing '%s' to '%s:%s'\n", imgURL, repoURL, tag)
		if err := crane.Push(img, fmt.Sprintf("%s:%s", repoURL, tag)); err != nil {
			return err
		}
	}

	return nil
}
