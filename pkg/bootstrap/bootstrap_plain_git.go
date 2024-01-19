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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	gogit "github.com/go-git/go-git/v5"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/cli-utils/pkg/object"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/repository"
	"github.com/fluxcd/pkg/kustomize/filesys"
	runclient "github.com/fluxcd/pkg/runtime/client"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/log"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/kustomization"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sync"
	"github.com/fluxcd/flux2/v2/pkg/status"
)

type PlainGitBootstrapper struct {
	url    string
	branch string

	signature             git.Signature
	commitMessageAppendix string

	gpgKeyRing    openpgp.EntityList
	gpgPassphrase string
	gpgKeyID      string

	restClientGetter  genericclioptions.RESTClientGetter
	restClientOptions *runclient.Options

	postGenerateSecret []PostGenerateSecretFunc

	gitClient repository.Client
	kube      client.Client
	logger    log.Logger
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

func NewPlainGitProvider(git repository.Client, kube client.Client, opts ...GitOption) (*PlainGitBootstrapper, error) {
	b := &PlainGitBootstrapper{
		gitClient: git,
		kube:      kube,
	}
	for _, opt := range opts {
		opt.applyGit(b)
	}
	return b, nil
}

func (b *PlainGitBootstrapper) ReconcileComponents(ctx context.Context, manifestsBase string, options install.Options, _ sourcesecret.Options) error {
	// Clone if not already
	if _, err := b.gitClient.Head(); err != nil {
		if err != git.ErrNoGitRepository {
			return err
		}

		b.logger.Actionf("cloning branch %q from Git repository %q", b.branch, b.url)
		var cloned bool
		if err = retry(1, 2*time.Second, func() (err error) {
			if err = b.cleanGitRepoDir(); err != nil {
				b.logger.Warningf(" failed to clean directory for git repo: %w", err)
				return
			}
			_, err = b.gitClient.Clone(ctx, b.url, repository.CloneConfig{
				CheckoutStrategy: repository.CheckoutStrategy{
					Branch: b.branch,
				},
			})
			if err != nil {
				b.logger.Warningf(" clone failure: %s", err)
			}
			if err == nil {
				cloned = true
			}
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

	// Write generated files and make a commit
	var signer *openpgp.Entity
	if b.gpgKeyRing != nil {
		signer, err = getOpenPgpEntity(b.gpgKeyRing, b.gpgPassphrase, b.gpgKeyID)
		if err != nil {
			return fmt.Errorf("failed to generate OpenPGP entity: %w", err)
		}
	}
	commitMsg := fmt.Sprintf("Add Flux %s component manifests", options.Version)
	if b.commitMessageAppendix != "" {
		commitMsg = commitMsg + "\n\n" + b.commitMessageAppendix
	}

	commit, err := b.gitClient.Commit(git.Commit{
		Author:  b.signature,
		Message: commitMsg,
	}, repository.WithFiles(map[string]io.Reader{
		manifests.Path: strings.NewReader(manifests.Content),
	}), repository.WithSigner(signer))
	if err != nil && err != git.ErrNoStagedFiles {
		return fmt.Errorf("failed to commit component manifests: %w", err)
	}

	if err == nil {
		b.logger.Successf("committed component manifests to %q (%q)", b.branch, commit)
		b.logger.Actionf("pushing component manifests to %q", b.url)
		if err = b.gitClient.Push(ctx, repository.PushConfig{}); err != nil {
			return fmt.Errorf("failed to push manifests: %w", err)
		}
	} else {
		b.logger.Successf("component manifests are up to date")
	}

	// Conditionally install manifests
	if mustInstallManifests(ctx, b.kube, options.Namespace) {
		b.logger.Actionf("installing components in %q namespace", options.Namespace)

		componentsYAML := filepath.Join(b.gitClient.Path(), manifests.Path)
		kfile := filepath.Join(filepath.Dir(componentsYAML), konfig.DefaultKustomizationFileName())
		if _, err := os.Stat(kfile); err == nil {
			// Apply the components and their patches
			if _, err := utils.Apply(ctx, b.restClientGetter, b.restClientOptions, b.gitClient.Path(), kfile); err != nil {
				return err
			}
		} else {
			// Apply the CRDs and controllers
			if _, err := utils.Apply(ctx, b.restClientGetter, b.restClientOptions, b.gitClient.Path(), componentsYAML); err != nil {
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
	if ok && options.Keypair == nil && len(options.CAFile) == 0 && len(options.Username+options.Password) == 0 {
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
	if _, err := b.gitClient.Head(); err != nil {
		if err == git.ErrNoGitRepository {
			b.logger.Actionf("cloning branch %q from Git repository %q", b.branch, b.url)
			var cloned bool
			if err = retry(1, 2*time.Second, func() (err error) {
				if err = b.cleanGitRepoDir(); err != nil {
					b.logger.Warningf(" failed to clean directory for git repo: %w", err)
					return
				}
				_, err = b.gitClient.Clone(ctx, b.url, repository.CloneConfig{
					CheckoutStrategy: repository.CheckoutStrategy{
						Branch: b.branch,
					},
				})
				if err != nil {
					b.logger.Warningf(" clone failure: %s", err)
				}
				if err == nil {
					cloned = true
				}
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

	// Create secure Kustomize FS
	fs, err := filesys.MakeFsOnDiskSecureBuild(b.gitClient.Path())
	if err != nil {
		return fmt.Errorf("failed to initialize Kustomize file system: %w", err)
	}

	if err = fs.WriteFile(filepath.Join(b.gitClient.Path(), manifests.Path), []byte(manifests.Content)); err != nil {
		return err
	}

	// Generate Kustomization
	kusManifests, err := kustomization.Generate(kustomization.Options{
		FileSystem: fs,
		BaseDir:    b.gitClient.Path(),
		TargetPath: filepath.Dir(manifests.Path),
	})
	if err != nil {
		return fmt.Errorf("%s generation failed: %w", konfig.DefaultKustomizationFileName(), err)
	}
	b.logger.Successf("generated sync manifests")

	// Write generated files and make a commit
	var signer *openpgp.Entity
	if b.gpgKeyRing != nil {
		signer, err = getOpenPgpEntity(b.gpgKeyRing, b.gpgPassphrase, b.gpgKeyID)
		if err != nil {
			return fmt.Errorf("failed to generate OpenPGP entity: %w", err)
		}
	}
	commitMsg := "Add Flux sync manifests"
	if b.commitMessageAppendix != "" {
		commitMsg = commitMsg + "\n\n" + b.commitMessageAppendix
	}

	commit, err := b.gitClient.Commit(git.Commit{
		Author:  b.signature,
		Message: commitMsg,
	}, repository.WithFiles(map[string]io.Reader{
		kusManifests.Path: strings.NewReader(kusManifests.Content),
	}), repository.WithSigner(signer))
	if err != nil && err != git.ErrNoStagedFiles {
		return fmt.Errorf("failed to commit sync manifests: %w", err)
	}

	if err == nil {
		b.logger.Successf("committed sync manifests to %q (%q)", b.branch, commit)
		b.logger.Actionf("pushing sync manifests to %q", b.url)
		err = b.gitClient.Push(ctx, repository.PushConfig{})
		if err != nil {
			if strings.HasPrefix(err.Error(), gogit.ErrNonFastForwardUpdate.Error()) {
				b.logger.Waitingf("git conflict detected, retrying with a fresh clone")
				if err := os.RemoveAll(b.gitClient.Path()); err != nil {
					return fmt.Errorf("failed to remove tmp dir: %w", err)
				}
				if err := os.Mkdir(b.gitClient.Path(), 0o700); err != nil {
					return fmt.Errorf("failed to recreate tmp dir: %w", err)
				}
				if err = retry(1, 2*time.Second, func() (err error) {
					if err = b.cleanGitRepoDir(); err != nil {
						b.logger.Warningf(" failed to clean directory for git repo: %w", err)
						return
					}
					_, err = b.gitClient.Clone(ctx, b.url, repository.CloneConfig{
						CheckoutStrategy: repository.CheckoutStrategy{
							Branch: b.branch,
						},
					})
					if err != nil {
						b.logger.Warningf(" clone failure: %s", err)
					}
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
	if _, err := utils.Apply(ctx, b.restClientGetter, b.restClientOptions, b.gitClient.Path(), filepath.Join(b.gitClient.Path(), kusManifests.Path)); err != nil {
		return err
	}

	b.logger.Successf("reconciled sync configuration")

	return nil
}

func (b *PlainGitBootstrapper) ReportKustomizationHealth(ctx context.Context, options sync.Options, pollInterval, timeout time.Duration) error {
	head, err := b.gitClient.Head()
	if err != nil {
		return err
	}

	objKey := client.ObjectKey{Name: options.Name, Namespace: options.Namespace}

	expectRevision := fmt.Sprintf("%s@%s", options.Branch, git.Hash(head).Digest())
	b.logger.Waitingf("waiting for Kustomization %q to be reconciled", objKey.String())
	k := &kustomizev1.Kustomization{
		TypeMeta: metav1.TypeMeta{
			Kind: kustomizev1.KustomizationKind,
		},
	}
	if err := wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true,
		objectReconciled(b.kube, objKey, k, expectRevision)); err != nil {
		// If the poll timed out, we want to log the ready condition message as
		// that likely contains the reason
		if errors.Is(err, context.DeadlineExceeded) {
			readyCondition := apimeta.FindStatusCondition(k.Status.Conditions, meta.ReadyCondition)
			if readyCondition != nil && readyCondition.Status != metav1.ConditionTrue {
				err = fmt.Errorf("kustomization '%s' not ready: '%s'", objKey, readyCondition.Message)
			}
		}
		b.logger.Failuref(err.Error())
		return fmt.Errorf("error while waiting for Kustomization to be ready: '%s'", err)
	}
	b.logger.Successf("Kustomization reconciled successfully")
	return nil
}

func (b *PlainGitBootstrapper) ReportGitRepoHealth(ctx context.Context, options sync.Options, pollInterval, timeout time.Duration) error {
	head, err := b.gitClient.Head()
	if err != nil {
		return err
	}

	objKey := client.ObjectKey{Name: options.Name, Namespace: options.Namespace}

	b.logger.Waitingf("waiting for GitRepository %q to be reconciled", objKey.String())
	expectRevision := fmt.Sprintf("%s@%s", options.Branch, git.Hash(head).Digest())
	g := &sourcev1.GitRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       sourcev1.GitRepositoryKind,
			APIVersion: sourcev1.GroupVersion.String(),
		},
	}
	if err := wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true,
		objectReconciled(b.kube, objKey, g, expectRevision)); err != nil {
		// If the poll timed out, we want to log the ready condition message as
		// that likely contains the reason
		if errors.Is(err, context.DeadlineExceeded) {
			readyCondition := apimeta.FindStatusCondition(g.Status.Conditions, meta.ReadyCondition)
			if readyCondition != nil && readyCondition.Status != metav1.ConditionTrue {
				err = fmt.Errorf("gitrepository '%s' not ready: '%s'", objKey, readyCondition.Message)
			}
		}
		b.logger.Failuref(err.Error())
		return fmt.Errorf("error while waiting for GitRepository to be ready: '%s'", err)
	}
	b.logger.Successf("GitRepository reconciled successfully")
	return nil
}
func (b *PlainGitBootstrapper) ReportComponentsHealth(ctx context.Context, install install.Options, timeout time.Duration) error {
	cfg, err := utils.KubeConfig(b.restClientGetter, b.restClientOptions)
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

// cleanGitRepoDir cleans the directory meant for the Git repo.
func (b *PlainGitBootstrapper) cleanGitRepoDir() error {
	dirs, err := os.ReadDir(b.gitClient.Path())
	if err != nil {
		return err
	}
	var errs []error
	for _, dir := range dirs {
		if err := os.RemoveAll(filepath.Join(b.gitClient.Path(), dir.Name())); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func getOpenPgpEntity(keyRing openpgp.EntityList, passphrase, keyID string) (*openpgp.Entity, error) {
	if len(keyRing) == 0 {
		return nil, fmt.Errorf("empty GPG key ring")
	}

	var entity *openpgp.Entity
	if keyID != "" {
		keyID = strings.TrimPrefix(keyID, "0x")
		if len(keyID) != 16 {
			return nil, fmt.Errorf("invalid GPG key id length; expected %d, got %d", 16, len(keyID))
		}
		keyID = strings.ToUpper(keyID)

		for _, ent := range keyRing {
			if ent.PrimaryKey.KeyIdString() == keyID {
				entity = ent
			}
		}

		if entity == nil {
			return nil, fmt.Errorf("no GPG keyring matching key id '%s' found", keyID)
		}
		if entity.PrivateKey == nil {
			return nil, fmt.Errorf("keyring does not contain private key for key id '%s'", keyID)
		}
	} else {
		entity = keyRing[0]
	}

	err := entity.PrivateKey.Decrypt([]byte(passphrase))
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt GPG private key: %w", err)
	}

	return entity, nil
}
