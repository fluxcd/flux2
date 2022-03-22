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

package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/yaml"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"

	"github.com/fluxcd/flux2/internal/bootstrap/git"
	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/flux2/pkg/log"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/pkg/manifestgen/kustomization"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/pkg/manifestgen/sync"
	"github.com/fluxcd/flux2/pkg/status"
)

type PlainGitBootstrapper struct {
	url      string
	branch   string
	caBundle []byte

	author                git.Author
	commitMessageAppendix string

	gpgKeyRingPath string
	gpgPassphrase  string
	gpgKeyID       string

	restClientGetter genericclioptions.RESTClientGetter

	postGenerateSecret []PostGenerateSecretFunc

	git    git.Git
	kube   client.Client
	logger log.Logger
}

type GitOption interface {
	applyGit(b *PlainGitBootstrapper)
}

func WithRepositoryURL(url string) GitOption {
	return repositoryURLOption(url)
}

type repositoryURLOption string

func (o repositoryURLOption) applyGit(b *PlainGitBootstrapper) {
	b.url = string(o)
}

func WithPostGenerateSecretFunc(callback PostGenerateSecretFunc) GitOption {
	return postGenerateSecret(callback)
}

type postGenerateSecret PostGenerateSecretFunc

func (o postGenerateSecret) applyGit(b *PlainGitBootstrapper) {
	b.postGenerateSecret = append(b.postGenerateSecret, PostGenerateSecretFunc(o))
}

func NewPlainGitProvider(git git.Git, kube client.Client, opts ...GitOption) (*PlainGitBootstrapper, error) {
	b := &PlainGitBootstrapper{
		git:  git,
		kube: kube,
	}
	for _, opt := range opts {
		opt.applyGit(b)
	}
	return b, nil
}

func (b *PlainGitBootstrapper) ReconcileComponents(ctx context.Context, manifestsBase string, options install.Options, secretOpts sourcesecret.Options) error {
	// Clone if not already
	if _, err := b.git.Status(); err != nil {
		if err != git.ErrNoGitRepository {
			return err
		}

		b.logger.Actionf("cloning branch %q from Git repository %q", b.branch, b.url)
		var cloned bool
		if err = retry(1, 2*time.Second, func() (err error) {
			cloned, err = b.git.Clone(ctx, b.url, b.branch, b.caBundle)
			return
		}); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		if cloned {
			b.logger.Successf("cloned repository")
		}
	}

	// Generate component manifests
	b.logger.Actionf("generating component manifests")
	manifests, err := install.Generate(options, manifestsBase)
	if err != nil {
		return fmt.Errorf("component manifest generation failed: %w", err)
	}
	b.logger.Successf("generated component manifests")

	// Write manifest to Git repository
	if err = b.git.Write(manifests.Path, strings.NewReader(manifests.Content)); err != nil {
		return fmt.Errorf("failed to write manifest %q: %w", manifests.Path, err)
	}

	// Git commit generated
	gpgOpts := git.WithGpgSigningOption(b.gpgKeyRingPath, b.gpgPassphrase, b.gpgKeyID)
	commitMsg := fmt.Sprintf("Add Flux %s component manifests", options.Version)
	if b.commitMessageAppendix != "" {
		commitMsg = commitMsg + "\n\n" + b.commitMessageAppendix
	}
	commit, err := b.git.Commit(git.Commit{
		Author:  b.author,
		Message: commitMsg,
	}, gpgOpts)
	if err != nil && err != git.ErrNoStagedFiles {
		return fmt.Errorf("failed to commit sync manifests: %w", err)
	}
	if err == nil {
		b.logger.Successf("committed sync manifests to %q (%q)", b.branch, commit)
		b.logger.Actionf("pushing component manifests to %q", b.url)
		if err = b.git.Push(ctx, b.caBundle); err != nil {
			return fmt.Errorf("failed to push manifests: %w", err)
		}
	} else {
		b.logger.Successf("component manifests are up to date")
	}

	// Conditionally install manifests
	if mustInstallManifests(ctx, b.kube, options.Namespace) {
		componentsYAML := filepath.Join(b.git.Path(), manifests.Path)

		// Apply components using any existing customisations
		kfile := filepath.Join(filepath.Dir(componentsYAML), konfig.DefaultKustomizationFileName())
		if _, err := os.Stat(kfile); err == nil {
			// Apply the components and their patches
			b.logger.Actionf("installing components in %q namespace", options.Namespace)
			if _, err := utils.Apply(ctx, b.restClientGetter, kfile); err != nil {
				return err
			}
		} else {
			// Apply the CRDs and controllers
			if _, err := utils.Apply(ctx, b.restClientGetter, componentsYAML); err != nil {
				return err
			}
		}
		b.logger.Successf("installed components")
	}

	b.logger.Successf("reconciled components")
	return nil
}

func (b *PlainGitBootstrapper) ReconcileSourceSecret(ctx context.Context, options sourcesecret.Options) error {
	// Determine if there is an existing secret
	secretKey := client.ObjectKey{Name: options.Name, Namespace: options.Namespace}
	b.logger.Actionf("determining if source secret %q exists", secretKey)
	ok, err := secretExists(ctx, b.kube, secretKey)
	if err != nil {
		return fmt.Errorf("failed to determine if deploy key secret exists: %w", err)
	}

	// Return early if exists and no custom config is passed
	if ok && len(options.CAFilePath+options.PrivateKeyPath+options.Username+options.Password) == 0 {
		b.logger.Successf("source secret up to date")
		return nil
	}

	// Generate source secret
	b.logger.Actionf("generating source secret")
	manifest, err := sourcesecret.Generate(options)
	if err != nil {
		return err
	}
	var secret corev1.Secret
	if err := yaml.Unmarshal([]byte(manifest.Content), &secret); err != nil {
		return fmt.Errorf("failed to unmarshal generated source secret manifest: %w", err)
	}

	for _, callback := range b.postGenerateSecret {
		if err = callback(ctx, secret, options); err != nil {
			return err
		}
	}

	// Apply source secret
	b.logger.Actionf("applying source secret %q", secretKey)
	if err = reconcileSecret(ctx, b.kube, secret); err != nil {
		return err
	}
	b.logger.Successf("reconciled source secret")

	return nil
}

func (b *PlainGitBootstrapper) ReconcileSyncConfig(ctx context.Context, options sync.Options) error {
	// Confirm that sync configuration does not overwrite existing config
	if curPath, err := kustomizationPathDiffers(ctx, b.kube, client.ObjectKey{Name: options.Name, Namespace: options.Namespace}, options.TargetPath); err != nil {
		return fmt.Errorf("failed to determine if sync configuration would overwrite existing Kustomization: %w", err)
	} else if curPath != "" {
		return fmt.Errorf("sync path configuration (%q) would overwrite path (%q) of existing Kustomization", options.TargetPath, curPath)
	}

	// Clone if not already
	if _, err := b.git.Status(); err != nil {
		if err == git.ErrNoGitRepository {
			b.logger.Actionf("cloning branch %q from Git repository %q", b.branch, b.url)
			var cloned bool
			if err = retry(1, 2*time.Second, func() (err error) {
				cloned, err = b.git.Clone(ctx, b.url, b.branch, b.caBundle)
				return
			}); err != nil {
				return fmt.Errorf("failed to clone repository: %w", err)
			}
			if cloned {
				b.logger.Successf("cloned repository")
			}
		}
		return err
	}

	// Generate sync manifests and write to Git repository
	b.logger.Actionf("generating sync manifests")
	manifests, err := sync.Generate(options)
	if err != nil {
		return fmt.Errorf("sync manifests generation failed: %w", err)
	}
	if err = b.git.Write(manifests.Path, strings.NewReader(manifests.Content)); err != nil {
		return fmt.Errorf("failed to write manifest %q: %w", manifests.Path, err)
	}
	kusManifests, err := kustomization.Generate(kustomization.Options{
		FileSystem: filesys.MakeFsOnDisk(),
		BaseDir:    b.git.Path(),
		TargetPath: filepath.Dir(manifests.Path),
	})
	if err != nil {
		return fmt.Errorf("kustomization.yaml generation failed: %w", err)
	}
	if err = b.git.Write(kusManifests.Path, strings.NewReader(kusManifests.Content)); err != nil {
		return fmt.Errorf("failed to write manifest %q: %w", kusManifests.Path, err)
	}
	b.logger.Successf("generated sync manifests")

	// Git commit generated
	gpgOpts := git.WithGpgSigningOption(b.gpgKeyRingPath, b.gpgPassphrase, b.gpgKeyID)
	commitMsg := fmt.Sprintf("Add Flux sync manifests")
	if b.commitMessageAppendix != "" {
		commitMsg = commitMsg + "\n\n" + b.commitMessageAppendix
	}
	commit, err := b.git.Commit(git.Commit{
		Author:  b.author,
		Message: commitMsg,
	}, gpgOpts)

	if err != nil && err != git.ErrNoStagedFiles {
		return fmt.Errorf("failed to commit sync manifests: %w", err)
	}

	if err == nil {
		b.logger.Successf("committed sync manifests to %q (%q)", b.branch, commit)
		b.logger.Actionf("pushing sync manifests to %q", b.url)
		err = b.git.Push(ctx, b.caBundle)
		if err != nil {
			if strings.HasPrefix(err.Error(), gogit.ErrNonFastForwardUpdate.Error()) {
				b.logger.Waitingf("git conflict detected, retrying with a fresh clone")
				if err := os.RemoveAll(b.git.Path()); err != nil {
					return fmt.Errorf("failed to remove tmp dir: %w", err)
				}
				if err := os.Mkdir(b.git.Path(), 0o700); err != nil {
					return fmt.Errorf("failed to recreate tmp dir: %w", err)
				}
				if err = retry(1, 2*time.Second, func() (err error) {
					_, err = b.git.Clone(ctx, b.url, b.branch, b.caBundle)
					return
				}); err != nil {
					return fmt.Errorf("failed to clone repository: %w", err)
				}
				return b.ReconcileSyncConfig(ctx, options)
			}
			return fmt.Errorf("failed to push sync manifests: %w", err)
		}
	} else {
		b.logger.Successf("sync manifests are up to date")
	}

	// Apply to cluster
	b.logger.Actionf("applying sync manifests")
	if _, err := utils.Apply(ctx, b.restClientGetter, filepath.Join(b.git.Path(), kusManifests.Path)); err != nil {
		return err
	}

	b.logger.Successf("reconciled sync configuration")

	return nil
}

func (b *PlainGitBootstrapper) ReportKustomizationHealth(ctx context.Context, options sync.Options, pollInterval, timeout time.Duration) error {
	head, err := b.git.Head()
	if err != nil {
		return err
	}

	objKey := client.ObjectKey{Name: options.Name, Namespace: options.Namespace}

	b.logger.Waitingf("waiting for Kustomization %q to be reconciled", objKey.String())

	expectRevision := fmt.Sprintf("%s/%s", options.Branch, head)
	var k kustomizev1.Kustomization
	if err := wait.PollImmediate(pollInterval, timeout, kustomizationReconciled(
		ctx, b.kube, objKey, &k, expectRevision),
	); err != nil {
		b.logger.Failuref(err.Error())
		return err
	}

	b.logger.Successf("Kustomization reconciled successfully")
	return nil
}

func (b *PlainGitBootstrapper) ReportComponentsHealth(ctx context.Context, install install.Options, timeout time.Duration) error {
	cfg, err := utils.KubeConfig(b.restClientGetter)
	if err != nil {
		return err
	}

	checker, err := status.NewStatusChecker(cfg, 5*time.Second, timeout, b.logger)
	if err != nil {
		return err
	}

	var components = install.Components
	components = append(components, install.ComponentsExtra...)

	var identifiers []object.ObjMetadata
	for _, component := range components {
		identifiers = append(identifiers, object.ObjMetadata{
			Namespace: install.Namespace,
			Name:      component,
			GroupKind: schema.GroupKind{Group: "apps", Kind: "Deployment"},
		})
	}

	b.logger.Actionf("confirming components are healthy")
	if err := checker.Assess(identifiers...); err != nil {
		return err
	}
	b.logger.Successf("all components are healthy")
	return nil
}
