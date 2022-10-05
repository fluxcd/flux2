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
	"os"
	"os/exec"
	"path/filepath"
	"time"

	git2go "github.com/libgit2/git2go/v33"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	automationv1beta1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	reflectorv1beta1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	notiv1beta1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
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
	err = reflectorv1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", nil, err
	}
	err = automationv1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", nil, err
	}
	err = notiv1beta1.AddToScheme(scheme.Scheme)
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
	httpsCredentials := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "https-credentials", Namespace: "flux-system"}}
	_, err = controllerutil.CreateOrUpdate(ctx, kubeClient, httpsCredentials, func() error {
		httpsCredentials.StringData = map[string]string{
			"username": "git",
			"password": azdoPat,
		}
		return nil
	})
	if err != nil {
		return err
	}
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

	// Install Flux and push files to git repository
	repo, repoDir, err := getRepository(repoUrl, defaultBranch, true, azdoPat)
	if err != nil {
		return err
	}
	err = runCommand(ctx, repoDir, "mkdir -p ./clusters/e2e/flux-system")
	if err != nil {
		return err
	}
	err = runCommand(ctx, repoDir, "flux install --components-extra=\"image-reflector-controller,image-automation-controller\" --export > ./clusters/e2e/flux-system/gotk-components.yaml")
	if err != nil {
		return err
	}
	err = runCommand(ctx, repoDir, fmt.Sprintf("flux create source git flux-system --git-implementation=libgit2 --url=%s --branch=%s --secret-ref=https-credentials --interval=1m  --export > ./clusters/e2e/flux-system/gotk-sync.yaml", repoUrl, defaultBranch))
	if err != nil {
		return err
	}
	err = runCommand(ctx, repoDir, "flux create kustomization flux-system --source=flux-system --path='./clusters/e2e' --prune=true --interval=1m --export >> ./clusters/e2e/flux-system/gotk-sync.yaml")
	if err != nil {
		return err
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
	err = runCommand(ctx, repoDir, fmt.Sprintf("echo \"%s\" > ./clusters/e2e/flux-system/kustomization.yaml", kustomizeYaml))
	if err != nil {
		return err
	}
	err = commitAndPushAll(repo, defaultBranch, azdoPat)
	if err != nil {
		return err
	}
	// Need to apply CRDs first to make sure that the sync resources will apply properly
	err = runCommand(ctx, repoDir, fmt.Sprintf("kubectl --kubeconfig=%s apply -f ./clusters/e2e/flux-system/gotk-components.yaml", kubeconfigPath))
	if err != nil {
		return err
	}
	err = runCommand(ctx, repoDir, fmt.Sprintf("kubectl --kubeconfig=%s apply -k ./clusters/e2e/flux-system/", kubeconfigPath))
	if err != nil {
		return err
	}
	return nil
}

func runCommand(ctx context.Context, dir, command string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
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
			GitImplementation: sourcev1.LibGit2Implementation,
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

func getRepository(url, branchName string, overrideBranch bool, password string) (*git2go.Repository, string, error) {
	checkoutBranch := defaultBranch
	if overrideBranch == false {
		checkoutBranch = branchName
	}

	tmpDir, err := os.MkdirTemp("", "*-repository")
	if err != nil {
		return nil, "", err
	}
	repo, err := git2go.Clone(url, tmpDir, &git2go.CloneOptions{
		FetchOptions: git2go.FetchOptions{
			RemoteCallbacks: git2go.RemoteCallbacks{
				CredentialsCallback: credentialCallback("git", password),
			},
		},
		CheckoutBranch: checkoutBranch,
		CheckoutOptions: git2go.CheckoutOpts{
			Strategy: git2go.CheckoutSafe,
		},
	})
	if err != nil {
		return nil, "", err
	}
	// Nothing to do further if correct branch is checked out
	if checkoutBranch == branchName {
		return repo, tmpDir, nil
	}
	head, err := repo.Head()
	if err != nil {
		return nil, "", err
	}
	headCommit, err := repo.LookupCommit(head.Target())
	if err != nil {
		return nil, "", err
	}
	_, err = repo.CreateBranch(branchName, headCommit, true)
	if err != nil {
		return nil, "", err
	}
	return repo, tmpDir, nil
}

func addFile(dir, path, content string) error {
	err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0777)
	if err != nil {
		return err
	}
	return nil
}

func commitAndPushAll(repo *git2go.Repository, branchName, password string) error {
	idx, err := repo.Index()
	if err != nil {
		return err
	}
	err = idx.AddAll([]string{}, git2go.IndexAddDefault, nil)
	if err != nil {
		return err
	}
	treeId, err := idx.WriteTree()
	if err != nil {
		return err
	}
	err = idx.Write()
	if err != nil {
		return err
	}
	tree, err := repo.LookupTree(treeId)
	if err != nil {
		return err
	}
	branch, err := repo.LookupBranch(branchName, git2go.BranchLocal)
	if err != nil {
		return err
	}
	commitTarget, err := repo.LookupCommit(branch.Target())
	if err != nil {
		return err
	}
	sig := &git2go.Signature{
		Name:  "git",
		Email: "test@example.com",
		When:  time.Now(),
	}
	_, err = repo.CreateCommit(fmt.Sprintf("refs/heads/%s", branchName), sig, sig, "add file", tree, commitTarget)
	if err != nil {
		return err
	}
	origin, err := repo.Remotes.Lookup("origin")
	if err != nil {
		return err
	}
	err = origin.Push([]string{fmt.Sprintf("+refs/heads/%s", branchName)}, &git2go.PushOptions{
		RemoteCallbacks: git2go.RemoteCallbacks{
			CredentialsCallback: credentialCallback("git", password),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func createTagAndPush(repo *git2go.Repository, branchName, tag, password string) error {
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
