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

package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-containerregistry/pkg/crane"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
	"github.com/fluxcd/pkg/git/repository"
	"github.com/fluxcd/pkg/runtime/conditions"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/fluxcd/test-infra/tftestenv"
)

// installFlux adds the core Flux components to the cluster specified in the kubeconfig file.
func installFlux(ctx context.Context, tmpDir string, kubeconfigPath string) error {
	// Create flux-system namespace
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "flux-system",
		},
	}
	err := testEnv.Create(ctx, &namespace)
	if err != nil {
		return err
	}

	repoURL := getTransportURL(cfg.fleetInfraRepository)
	if cfg.kustomizationYaml != "" {
		files := make(map[string]io.Reader)
		files["clusters/e2e/flux-system/kustomization.yaml"] = strings.NewReader(cfg.kustomizationYaml)
		files["clusters/e2e/flux-system/gotk-components.yaml"] = strings.NewReader("")
		files["clusters/e2e/flux-system/gotk-sync.yaml"] = strings.NewReader("")
		c, err := getRepository(ctx, tmpDir, repoURL, defaultBranch, cfg.defaultAuthOpts)
		if err != nil {
			return err
		}

		err = commitAndPushAll(ctx, c, files, defaultBranch)
		if err != nil {
			return err
		}
	}

	var bootstrapArgs string
	if cfg.defaultGitTransport == git.SSH {
		f, err := os.CreateTemp("", "flux-e2e-ssh-key-*")
		if err != nil {
			return err
		}
		err = os.WriteFile(f.Name(), []byte(cfg.gitPrivateKey), 0o600)
		if err != nil {
			return err
		}
		bootstrapArgs = fmt.Sprintf("--private-key-file=%s -s", f.Name())
	} else {
		bootstrapArgs = fmt.Sprintf("--token-auth --password=%s", cfg.gitPat)
	}

	bootstrapCmd := fmt.Sprintf("./build/flux bootstrap git  --url=%s %s --kubeconfig=%s --path=clusters/e2e "+
		" --components-extra image-reflector-controller,image-automation-controller",
		repoURL, bootstrapArgs, kubeconfigPath)

	return tftestenv.RunCommand(ctx, "./", bootstrapCmd, tftestenv.RunCommandOptions{
		Timeout: 15 * time.Minute,
	})
}

func uninstallFlux(ctx context.Context) error {
	uninstallCmd := fmt.Sprintf("./build/flux uninstall --kubeconfig %s -s", kubeconfigPath)
	if err := tftestenv.RunCommand(ctx, "./", uninstallCmd, tftestenv.RunCommandOptions{
		Timeout: 15 * time.Minute,
	}); err != nil {
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
	if err := kubeClient.Get(ctx, nn, source); err != nil {
		return err
	}
	if err := checkReadyCondition(source); err != nil {
		return err
	}

	kustomization := &kustomizev1.Kustomization{}
	if err := kubeClient.Get(ctx, nn, kustomization); err != nil {
		return err
	}
	if err := checkReadyCondition(kustomization); err != nil {
		return err
	}

	return nil
}

type nsConfig struct {
	repoURL      string
	ref          *sourcev1.GitRepositoryRef
	protocol     git.TransportType
	objectName   string
	path         string
	modifyKsSpec func(spec *kustomizev1.KustomizationSpec)
}

// setUpFluxConfigs creates the namespace, then creates the git secret,
// git repository and kustomization in that namespace
func setUpFluxConfig(ctx context.Context, name string, opts nsConfig) error {
	transport := cfg.defaultGitTransport
	if opts.protocol != "" {
		transport = opts.protocol
	}

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := testEnv.Create(ctx, &namespace); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-credentials",
			Namespace: name,
		},
	}

	secret.StringData = map[string]string{
		"username": cfg.gitUsername,
		"password": cfg.gitPat,
	}

	if transport == git.SSH {
		secret.StringData = map[string]string{
			"identity":     cfg.gitPrivateKey,
			"identity.pub": cfg.gitPublicKey,
			"known_hosts":  cfg.knownHosts,
		}
	}
	if err := testEnv.Create(ctx, &secret); err != nil {
		return err
	}

	ref := &sourcev1.GitRepositoryRef{
		Branch: name,
	}

	if opts.ref != nil {
		ref = opts.ref
	}

	gitSpec := &sourcev1.GitRepositorySpec{
		Interval: metav1.Duration{
			Duration: 1 * time.Minute,
		},
		Reference: ref,
		SecretRef: &meta.LocalObjectReference{
			Name: secret.Name,
		},
		URL: opts.repoURL,
	}

	source := &sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace.Name},
		Spec:       *gitSpec,
	}
	if err := testEnv.Create(ctx, source); err != nil {
		return err
	}

	ksSpec := &kustomizev1.KustomizationSpec{
		Path:            opts.path,
		TargetNamespace: name,
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
	kustomization := &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace.Name},
		Spec:       *ksSpec,
	}

	return testEnv.Create(ctx, kustomization)
}

func tearDownFluxConfig(ctx context.Context, name string) error {
	var allErr []error

	source := &sourcev1.GitRepository{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: name}}
	if err := testEnv.Delete(ctx, source); err != nil {
		allErr = append(allErr, err)
	}

	kustomization := &kustomizev1.Kustomization{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: name}}
	if err := testEnv.Delete(ctx, kustomization); err != nil {
		allErr = append(allErr, err)
	}

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := testEnv.Delete(ctx, &namespace); err != nil {
		allErr = append(allErr, err)
	}

	return kerrors.NewAggregate(allErr)
}

// getRepository and clones the git repository to the directory.
func getRepository(ctx context.Context, dir, repoURL, branchName string, authOpts *git.AuthOptions) (*gogit.Client, error) {
	c, err := gogit.NewClient(dir, authOpts, gogit.WithSingleBranch(false), gogit.WithDiskStorage())
	if err != nil {
		return nil, err
	}

	_, err = c.Clone(ctx, repoURL, repository.CloneConfig{
		CheckoutStrategy: repository.CheckoutStrategy{
			Branch: branchName,
		},
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

// commitAndPushAll checks out to the specified branch, creates the files, commits and then pushes them to
// the remote git repository.
func commitAndPushAll(ctx context.Context, client *gogit.Client, files map[string]io.Reader, branchName string) error {
	err := client.SwitchBranch(ctx, branchName)
	if err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return err
	}

	_, err = client.Commit(git.Commit{
		Author: git.Signature{
			Name:  git.DefaultPublicKeyAuthUser,
			Email: "test@example.com",
			When:  time.Now(),
		},
	}, repository.WithFiles(files))
	if err != nil {
		if errors.Is(err, git.ErrNoStagedFiles) {
			return nil
		}

		return err
	}

	err = client.Push(ctx, repository.PushConfig{})
	if err != nil {
		return fmt.Errorf("unable to push: %s", err)
	}

	return nil
}

func createTagAndPush(ctx context.Context, client *gogit.Client, branchName, newTag string) error {
	repo, err := extgogit.PlainOpen(client.Path())
	if err != nil {
		return err
	}

	ref, err := repo.Reference(plumbing.NewBranchReferenceName(branchName), false)
	if err != nil {
		return err
	}

	tags, err := repo.TagObjects()
	if err != nil {
		return err
	}

	err = tags.ForEach(func(tag *object.Tag) error {
		if tag.Name == newTag {
			err = repo.DeleteTag(tag.Name)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error deleting local tag: %w", err)
	}

	// Delete remote tag
	if err := client.Push(ctx, repository.PushConfig{
		Refspecs: []string{fmt.Sprintf(":refs/tags/%s", newTag)},
		Force:    true,
	}); err != nil && !errors.Is(err, extgogit.NoErrAlreadyUpToDate) {
		return fmt.Errorf("unable to delete existing tag: %w", err)
	}

	sig := &object.Signature{
		Name:  git.DefaultPublicKeyAuthUser,
		Email: "test@example.com",
		When:  time.Now(),
	}
	if _, err = repo.CreateTag(newTag, ref.Hash(), &extgogit.CreateTagOptions{
		Tagger:  sig,
		Message: "create tag",
	}); err != nil {
		return fmt.Errorf("unable to create tag: %w", err)
	}

	return client.Push(ctx, repository.PushConfig{
		Refspecs: []string{"refs/tags/*:refs/tags/*"},
	})
}

func pushImagesFromURL(repoURL, imgURL string, tags []string) error {
	img, err := crane.Pull(imgURL)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		if err := crane.Push(img, fmt.Sprintf("%s:%s", repoURL, tag)); err != nil {
			return err
		}
	}

	return nil
}

func getTransportURL(urls gitUrl) string {
	if cfg.defaultGitTransport == git.SSH {
		return urls.ssh
	}

	return urls.http
}

func authOpts(repoURL string, authData map[string][]byte) (*git.AuthOptions, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return nil, err
	}

	return git.NewAuthOptions(*u, authData)
}

// checkReadyCondition checks for a Ready condition, it returns nil if the condition is true
// or an error (with the message if the Ready condition is present).
func checkReadyCondition(from conditions.Getter) error {
	if conditions.IsReady(from) {
		return nil
	}
	errMsg := fmt.Sprintf("object not ready")
	readyMsg := conditions.GetMessage(from, meta.ReadyCondition)
	if readyMsg != "" {
		errMsg += ": " + readyMsg
	}
	return errors.New(errMsg)
}
