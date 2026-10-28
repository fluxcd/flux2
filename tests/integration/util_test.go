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
	"testing"
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

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	automationv1 "github.com/fluxcd/image-automation-controller/api/v1"
	reflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1"
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
		Name: "flux-system",
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

	bootstrapCmd := fmt.Sprintf("%s bootstrap git  --url=%s %s --kubeconfig=%s --path=clusters/e2e "+
		" --components-extra image-reflector-controller,image-automation-controller",
		fluxBin, repoURL, bootstrapArgs, kubeconfigPath)

	return tftestenv.RunCommand(ctx, "./", bootstrapCmd, tftestenv.RunCommandOptions{
		Timeout: 15 * time.Minute,
	})
}

func runFluxCheck(ctx context.Context) error {
	checkCmd := fmt.Sprintf("%s check --kubeconfig %s", fluxBin, kubeconfigPath)
	return tftestenv.RunCommand(ctx, "./", checkCmd, tftestenv.RunCommandOptions{
		AttachConsole: true,
	})
}

func uninstallFlux(ctx context.Context) error {
	uninstallCmd := fmt.Sprintf("%s uninstall --kubeconfig %s -s", fluxBin, kubeconfigPath)
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
		Name: name,
	}
	if err := testEnv.Create(ctx, &namespace); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	secret := corev1.Secret{
		Name:      "git-credentials",
		Namespace: name,
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
		Name: name, Namespace: namespace.Name,
		Spec: *gitSpec,
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
		Name: name, Namespace: namespace.Name,
		Spec: *ksSpec,
	}

	return testEnv.Create(ctx, kustomization)
}

func tearDownFluxConfig(ctx context.Context, name string) error {
	var allErr []error

	source := &sourcev1.GitRepository{Name: name, Namespace: name}
	if err := testEnv.Delete(ctx, source); err != nil {
		allErr = append(allErr, err)
	}

	kustomization := &kustomizev1.Kustomization{Name: name, Namespace: name}
	if err := testEnv.Delete(ctx, kustomization); err != nil {
		allErr = append(allErr, err)
	}

	namespace := corev1.Namespace{
		Name: name,
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
		Branch: branchName,
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
	errMsg := "object not ready"
	readyMsg := conditions.GetMessage(from, meta.ReadyCondition)
	if readyMsg != "" {
		errMsg += ": " + readyMsg
	}
	return errors.New(errMsg)
}

// dumpDiagnostics prints Flux object states and controller logs when a test
// has failed. It should be registered via t.Cleanup so that it runs after the
// test body completes.
func dumpDiagnostics(t *testing.T, ctx context.Context, namespace string) {
	t.Helper()
	if !t.Failed() {
		return
	}

	t.Log("=== Diagnostics dump (test failed) ===")
	dumpFluxObjects(t, ctx, namespace)
	dumpControllerLogs(t, ctx)
	t.Log("=== End diagnostics dump ===")
}

// dumpFluxObjects lists Flux custom resources in the given namespace and prints
// their status conditions.
func dumpFluxObjects(t *testing.T, ctx context.Context, namespace string) {
	t.Helper()
	listOpts := &client.ListOptions{Namespace: namespace}

	gitRepos := &sourcev1.GitRepositoryList{}
	if err := testEnv.List(ctx, gitRepos, listOpts); err == nil {
		for _, r := range gitRepos.Items {
			logObjectStatus(t, "GitRepository", r.Name, r.Namespace, r.Status.Conditions)
		}
	}

	helmRepos := &sourcev1.HelmRepositoryList{}
	if err := testEnv.List(ctx, helmRepos, listOpts); err == nil {
		for _, r := range helmRepos.Items {
			logObjectStatus(t, "HelmRepository", r.Name, r.Namespace, r.Status.Conditions)
		}
	}

	helmCharts := &sourcev1.HelmChartList{}
	if err := testEnv.List(ctx, helmCharts, listOpts); err == nil {
		for _, r := range helmCharts.Items {
			logObjectStatus(t, "HelmChart", r.Name, r.Namespace, r.Status.Conditions)
		}
	}

	kustomizations := &kustomizev1.KustomizationList{}
	if err := testEnv.List(ctx, kustomizations, listOpts); err == nil {
		for _, r := range kustomizations.Items {
			logObjectStatus(t, "Kustomization", r.Name, r.Namespace, r.Status.Conditions)
		}
	}

	helmReleases := &helmv2.HelmReleaseList{}
	if err := testEnv.List(ctx, helmReleases, listOpts); err == nil {
		for _, r := range helmReleases.Items {
			logObjectStatus(t, "HelmRelease", r.Name, r.Namespace, r.Status.Conditions)
		}
	}

	imageRepos := &reflectorv1.ImageRepositoryList{}
	if err := testEnv.List(ctx, imageRepos, listOpts); err == nil {
		for _, r := range imageRepos.Items {
			logObjectStatus(t, "ImageRepository", r.Name, r.Namespace, r.Status.Conditions)
		}
	}

	imagePolicies := &reflectorv1.ImagePolicyList{}
	if err := testEnv.List(ctx, imagePolicies, listOpts); err == nil {
		for _, r := range imagePolicies.Items {
			logObjectStatus(t, "ImagePolicy", r.Name, r.Namespace, r.Status.Conditions)
		}
	}

	imageAutomations := &automationv1.ImageUpdateAutomationList{}
	if err := testEnv.List(ctx, imageAutomations, listOpts); err == nil {
		for _, r := range imageAutomations.Items {
			logObjectStatus(t, "ImageUpdateAutomation", r.Name, r.Namespace, r.Status.Conditions)
		}
	}
}

// logObjectStatus prints the status conditions of a Flux object.
func logObjectStatus(t *testing.T, kind, name, namespace string, conditions []metav1.Condition) {
	t.Helper()
	t.Logf("  %s/%s (ns: %s):", kind, name, namespace)
	for _, c := range conditions {
		t.Logf("    %s: %s — %s (since %s)", c.Type, c.Status, c.Message, c.LastTransitionTime.Format(time.RFC3339))
	}
}

// dumpControllerLogs prints the logs of all Flux controller pods in the
// flux-system namespace.
func dumpControllerLogs(t *testing.T, ctx context.Context) {
	t.Helper()

	podList, err := testEnv.ClientGo.CoreV1().Pods("flux-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Logf("failed to list flux-system pods: %v", err)
		return
	}

	for _, pod := range podList.Items {
		logs, err := testEnv.ClientGo.
			CoreV1().
			Pods(pod.Namespace).
			GetLogs(pod.Name, &corev1.PodLogOptions{}).
			DoRaw(ctx)
		if err != nil {
			t.Logf("failed to get logs for pod %s: %v", pod.Name, err)
			continue
		}
		t.Logf("=== Logs for pod %s ===\n%s", pod.Name, string(logs))
	}
}

// logNamespacePods logs the state of all pods in the given namespace,
// including container statuses and recent events. Useful for understanding
// why a Helm install is stuck.
func logNamespacePods(t *testing.T, ctx context.Context, namespace string) {
	t.Helper()

	podList := &corev1.PodList{}
	if err := testEnv.List(ctx, podList, &client.ListOptions{Namespace: namespace}); err != nil {
		t.Logf("  failed to list pods in %s: %v", namespace, err)
		return
	}
	if len(podList.Items) == 0 {
		t.Logf("  no pods in namespace %s", namespace)
		return
	}
	for _, pod := range podList.Items {
		t.Logf("  pod %s: phase=%s", pod.Name, pod.Status.Phase)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				t.Logf("    container %s: waiting — %s: %s", cs.Name, cs.State.Waiting.Reason, cs.State.Waiting.Message)
			} else if cs.State.Terminated != nil {
				t.Logf("    container %s: terminated — %s (exit %d)", cs.Name, cs.State.Terminated.Reason, cs.State.Terminated.ExitCode)
			} else if cs.State.Running != nil {
				t.Logf("    container %s: running (ready=%v)", cs.Name, cs.Ready)
			}
		}
	}

	// Log recent events in the namespace for scheduling/pull failures.
	events, err := testEnv.ClientGo.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Logf("  failed to list events in %s: %v", namespace, err)
		return
	}
	if len(events.Items) > 0 {
		t.Logf("  events in namespace %s:", namespace)
		for _, e := range events.Items {
			t.Logf("    %s %s/%s: %s — %s", e.Type, e.InvolvedObject.Kind, e.InvolvedObject.Name, e.Reason, e.Message)
		}
	}
}
