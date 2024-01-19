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
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta2"
	automationv1beta1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	reflectorv1beta2 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notiv1 "github.com/fluxcd/notification-controller/api/v1"
	notiv1beta3 "github.com/fluxcd/notification-controller/api/v1beta3"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
	"github.com/fluxcd/pkg/git/repository"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"
	extgogit "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

const defaultBranch = "main"

// getKubernetesCredentials returns a path to a kubeconfig file and a kube client instance.
func getKubernetesCredentials(kubeconfig, aksHost, aksCert, aksKey, aksCa string) (string, client.Client, error) {
	tmpDir, err := os.MkdirTemp("", "*-azure-e2e")
	if err != nil {
		return "", nil, err
	}
	kubeconfigPath := fmt.Sprintf("%s/kubeconfig", tmpDir)
	os.WriteFile(kubeconfigPath, []byte(kubeconfig), 0750)
	kubeCfg := &rest.Config{
		Host: aksHost,
		TLSClientConfig: rest.TLSClientConfig{
			CertData: []byte(aksCert),
			KeyData:  []byte(aksKey),
			CAData:   []byte(aksCa),
		},
	}
	err = sourcev1b2.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", nil, err
	}
	err = sourcev1.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", nil, err
	}
	err = kustomizev1.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", nil, err
	}
	err = helmv2beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", nil, err
	}
	err = reflectorv1beta2.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", nil, err
	}
	err = automationv1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", nil, err
	}
	err = notiv1beta3.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", nil, err
	}
	err = notiv1.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", nil, err
	}
	kubeClient, err := client.New(kubeCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return "", nil, err
	}
	return kubeconfigPath, kubeClient, nil
}

// installFlux adds the core Flux components to the cluster specified in the kubeconfig file.
func installFlux(ctx context.Context, kubeClient client.Client, kubeconfigPath, repoUrl, azdoPat string, sp spConfig) error {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "flux-system",
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, cfg.kubeClient, &namespace, func() error {
		return nil
	})

	azureSp := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "azure-sp", Namespace: "flux-system"}}
	_, err = controllerutil.CreateOrUpdate(ctx, kubeClient, azureSp, func() error {
		azureSp.StringData = map[string]string{
			"AZURE_TENANT_ID":     sp.tenantId,
			"AZURE_CLIENT_ID":     sp.clientId,
			"AZURE_CLIENT_SECRET": sp.clientSecret,
		}
		return nil
	})
	if err != nil {
		return err
	}

	//// Install Flux and push files to git repository
	repo, _, err := getRepository(repoUrl, defaultBranch, true, azdoPat)
	if err != nil {
		return fmt.Errorf("error cloning repositoriy: %w", err)
	}

	kustomizeYaml := `
resources:
 - gotk-components.yaml
 - gotk-sync.yaml
patchesStrategicMerge:
 - |-
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: kustomize-controller
     namespace: flux-system
   spec:
    template:
      spec:
        containers:
        - name: manager
          envFrom:
          - secretRef:
              name: azure-sp
 - |-
   apiVersion: apps/v1
   kind: Deployment
   metadata:
    name: source-controller
    namespace: flux-system
   spec:
    template:
      spec:
        containers:
        - name: manager
          envFrom:
          - secretRef:
              name: azure-sp
`

	files := make(map[string]io.Reader)
	files["clusters/e2e/flux-system/kustomization.yaml"] = strings.NewReader(kustomizeYaml)
	files["clusters/e2e/flux-system/gotk-components.yaml"] = strings.NewReader("")
	files["clusters/e2e/flux-system/gotk-sync.yaml"] = strings.NewReader("")
	err = commitAndPushAll(repo, files, defaultBranch)
	if err != nil {
		return fmt.Errorf("error committing and pushing manifests: %w", err)
	}

	bootstrapCmd := fmt.Sprintf("flux bootstrap git  --url=%s --password=%s --kubeconfig=%s"+
		" --token-auth --path=clusters/e2e  --components-extra image-reflector-controller,image-automation-controller",
		repoUrl, azdoPat, kubeconfigPath)
	if err := runCommand(context.Background(), 10*time.Minute, "./", bootstrapCmd); err != nil {
		return fmt.Errorf("error running bootstrap: %w", err)
	}

	return nil
}

func runCommand(ctx context.Context, timeout time.Duration, dir, command string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", command)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failure to run command %s: %v", string(output), err)
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

func setupNamespace(ctx context.Context, kubeClient client.Client, repoUrl, password, name string) error {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, kubeClient, &namespace, func() error {
		return nil
	})
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "https-credentials",
			Namespace: name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, kubeClient, &secret, func() error {
		secret.StringData = map[string]string{
			"username": "git",
			"password": password,
		}
		return nil
	})
	source := &sourcev1.GitRepository{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace.Name}}
	_, err = controllerutil.CreateOrUpdate(ctx, kubeClient, source, func() error {
		source.Spec = sourcev1.GitRepositorySpec{
			Interval: metav1.Duration{
				Duration: 1 * time.Minute,
			},
			Reference: &sourcev1.GitRepositoryRef{
				Branch: name,
			},
			SecretRef: &meta.LocalObjectReference{
				Name: "https-credentials",
			},
			URL: repoUrl,
		}
		return nil
	})
	if err != nil {
		return err
	}
	kustomization := &kustomizev1.Kustomization{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace.Name}}
	_, err = controllerutil.CreateOrUpdate(ctx, kubeClient, kustomization, func() error {
		kustomization.Spec = kustomizev1.KustomizationSpec{
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
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func getRepository(repoURL, branchName string, overrideBranch bool, password string) (*gogit.Client, string, error) {
	checkoutBranch := defaultBranch
	if overrideBranch == false {
		checkoutBranch = branchName
	}

	tmpDir, err := os.MkdirTemp("", "*-repository")
	if err != nil {
		return nil, "", err
	}
	c, err := gogit.NewClient(tmpDir, &git.AuthOptions{
		Transport: git.HTTPS,
		Username:  "git",
		Password:  password,
	})
	if err != nil {
		return nil, "", err
	}

	_, err = c.Clone(context.Background(), repoURL, repository.CloneConfig{
		CheckoutStrategy: repository.CheckoutStrategy{
			Branch: checkoutBranch,
		},
	})
	if err != nil {
		return nil, "", err
	}

	err = c.SwitchBranch(context.Background(), branchName)
	if err != nil {
		return nil, "", err
	}

	return c, tmpDir, nil
}

func addFile(dir, path, content string) error {
	err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0777)
	if err != nil {
		return err
	}
	return nil
}

func commitAndPushAll(client *gogit.Client, files map[string]io.Reader, branchName string) error {
	repo, err := extgogit.PlainOpen(client.Path())
	if err != nil {
		return err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = wt.Checkout(&extgogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Force:  true,
	})
	if err != nil {
		return err
	}

	f := repository.WithFiles(files)
	_, err = client.Commit(git.Commit{
		Author: git.Signature{
			Name:  "git",
			Email: "test@example.com",
			When:  time.Now(),
		},
		Message: "add file",
	}, f)

	if err != nil {
		return err
	}

	err = client.Push(context.Background(), repository.PushConfig{})
	if err != nil {
		return err
	}

	return nil
}

func createTagAndPush(client *gogit.Client, branchName, newTag, password string) error {
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
		return err
	}

	sig := &object.Signature{
		Name:  "git",
		Email: "test@example.com",
		When:  time.Now(),
	}

	_, err = repo.CreateTag(newTag, ref.Hash(), &extgogit.CreateTagOptions{
		Tagger:  sig,
		Message: "create tag",
	})
	if err != nil {
		return err
	}

	auth := &http.BasicAuth{
		Username: "git",
		Password: password,
	}

	po := &extgogit.PushOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
		RefSpecs:   []gitconfig.RefSpec{gitconfig.RefSpec("refs/tags/*:refs/tags/*")},
		Auth:       auth,
	}
	if err := repo.Push(po); err != nil {
		return err
	}

	return nil
}

func getTestManifest(namespace string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: %s
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: foobar
  namespace: %s
`, namespace, namespace)
}
