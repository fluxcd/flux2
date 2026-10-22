//go:build unit
// +build unit

/*
Copyright 2026 The Flux authors

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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

// fakeReconciler marks GitRepositories, HelmCharts, Kustomizations and
// HelmReleases in the namespace as reconciled whenever their reconcile
// request annotation changes. The returned func reports how many reconcile
// requests were handled for the object with the given kind and name.
func fakeReconciler(t *testing.T, namespace string) func(kind, name string) int {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		<-done
	})

	var mu sync.Mutex
	handled := make(map[string]int)

	handle := func(kind string, obj client.Object, lastHandled *string, observedGeneration *int64, conditions *[]metav1.Condition) {
		requested := obj.GetAnnotations()[meta.ReconcileRequestAnnotation]
		if requested == "" || requested == *lastHandled {
			return
		}
		*lastHandled = requested
		*observedGeneration = obj.GetGeneration()
		apimeta.SetStatusCondition(conditions, metav1.Condition{
			Type:    meta.ReadyCondition,
			Status:  metav1.ConditionTrue,
			Reason:  meta.SucceededReason,
			Message: "reconciled",
		})
		if err := testEnv.client.Status().Update(ctx, obj); err == nil {
			mu.Lock()
			handled[kind+"/"+obj.GetName()]++
			mu.Unlock()
		}
	}

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}

			var repos sourcev1.GitRepositoryList
			if err := testEnv.client.List(ctx, &repos, client.InNamespace(namespace)); err == nil {
				for i := range repos.Items {
					o := &repos.Items[i]
					handle(sourcev1.GitRepositoryKind, o, &o.Status.LastHandledReconcileAt, &o.Status.ObservedGeneration, &o.Status.Conditions)
				}
			}

			var charts sourcev1.HelmChartList
			if err := testEnv.client.List(ctx, &charts, client.InNamespace(namespace)); err == nil {
				for i := range charts.Items {
					o := &charts.Items[i]
					handle(sourcev1.HelmChartKind, o, &o.Status.LastHandledReconcileAt, &o.Status.ObservedGeneration, &o.Status.Conditions)
				}
			}

			var kss kustomizev1.KustomizationList
			if err := testEnv.client.List(ctx, &kss, client.InNamespace(namespace)); err == nil {
				for i := range kss.Items {
					o := &kss.Items[i]
					handle(kustomizev1.KustomizationKind, o, &o.Status.LastHandledReconcileAt, &o.Status.ObservedGeneration, &o.Status.Conditions)
				}
			}

			var hrs helmv2.HelmReleaseList
			if err := testEnv.client.List(ctx, &hrs, client.InNamespace(namespace)); err == nil {
				for i := range hrs.Items {
					o := &hrs.Items[i]
					handle(helmv2.HelmReleaseKind, o, &o.Status.LastHandledReconcileAt, &o.Status.ObservedGeneration, &o.Status.Conditions)
				}
			}
		}
	}()

	return func(kind, name string) int {
		mu.Lock()
		defer mu.Unlock()
		return handled[kind+"/"+name]
	}
}

// assertGoldenError checks the output against a golden template and the
// error against the expected message.
func assertGoldenError(goldenFile string, tmpl map[string]string, expectedErr string) assertFunc {
	return func(output string, err error) error {
		if err == nil {
			return fmt.Errorf("expected error %q but was success", expectedErr)
		}
		if err.Error() != expectedErr {
			return fmt.Errorf("expected error %q but got %q", expectedErr, err.Error())
		}
		content, readErr := os.ReadFile(goldenFile)
		if readErr != nil {
			return readErr
		}
		expected, tmplErr := executeTemplate(string(content), tmpl)
		if tmplErr != nil {
			return tmplErr
		}
		if output != expected {
			return fmt.Errorf("mismatch from golden file %q:\nexpected:\n%s\nactual:\n%s", goldenFile, expected, output)
		}
		return nil
	}
}

// assertAnnotated checks that the reconcile request annotation is set on
// each named object and returns the annotation values.
func assertAnnotated(t *testing.T, namespace string, newObj func() client.Object, names ...string) map[string]string {
	t.Helper()
	values := make(map[string]string, len(names))
	for _, name := range names {
		obj := newObj()
		if err := testEnv.client.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, obj); err != nil {
			t.Fatalf("get %s: %v", name, err)
		}
		v := obj.GetAnnotations()[meta.ReconcileRequestAnnotation]
		if v == "" {
			t.Errorf("%s %s/%s was not annotated", obj.GetObjectKind().GroupVersionKind().Kind, namespace, name)
		}
		values[name] = v
	}
	return values
}

func newGitRepo() client.Object       { return &sourcev1.GitRepository{} }
func newHelmChart() client.Object     { return &sourcev1.HelmChart{} }
func newKustomization() client.Object { return &kustomizev1.Kustomization{} }
func newHelmRelease() client.Object   { return &helmv2.HelmRelease{} }

// setupReconcileNames creates the objects from file in a new namespace
// watched by a fake reconciler.
func setupReconcileNames(t *testing.T, file string) (string, map[string]string, func(kind, name string) int) {
	t.Helper()
	namespace := allocateNamespace("reconcile-names")
	tmpl := map[string]string{"ns": namespace}
	testEnv.CreateObjectFile(file, tmpl, t)
	handled := fakeReconciler(t, namespace)
	return namespace, tmpl, handled
}

func TestReconcileNames(t *testing.T) {
	oldTimeout := rootArgs.timeout
	t.Cleanup(func() { rootArgs.timeout = oldTimeout })
	const objects = "./testdata/reconcile_names/objects.yaml"

	t.Run("several names with a duplicate", func(t *testing.T) {
		namespace, tmpl, _ := setupReconcileNames(t, objects)
		cmd := cmdTestCase{
			args:   "reconcile source git app-a app-b app-a -n " + namespace + " --timeout=30s",
			assert: assertGoldenTemplateFile("testdata/reconcile_names/git_several.golden", tmpl),
		}
		cmd.runTestCmd(t)
		assertAnnotated(t, namespace, newGitRepo, "app-a", "app-b")
	})

	t.Run("one missing among several", func(t *testing.T) {
		namespace, tmpl, _ := setupReconcileNames(t, objects)
		cmd := cmdTestCase{
			args: "reconcile source git app-a missing app-b -n " + namespace + " --timeout=30s",
			assert: assertGoldenError("testdata/reconcile_names/git_one_missing.golden", tmpl,
				"failed to reconcile 1 of 3 GitRepository objects"),
		}
		cmd.runTestCmd(t)
		assertAnnotated(t, namespace, newGitRepo, "app-a", "app-b")
	})

	t.Run("all missing", func(t *testing.T) {
		namespace, tmpl, _ := setupReconcileNames(t, objects)
		cmd := cmdTestCase{
			args: "reconcile source git missing-1 missing-2 -n " + namespace + " --timeout=30s",
			assert: assertGoldenError("testdata/reconcile_names/git_all_missing.golden", tmpl,
				"failed to reconcile 2 of 2 GitRepository objects"),
		}
		cmd.runTestCmd(t)
	})

	t.Run("single missing name keeps the API error", func(t *testing.T) {
		namespace, _, _ := setupReconcileNames(t, objects)
		_, err := executeCommand("reconcile source git missing -n " + namespace + " --timeout=30s")
		var statusErr *apierrors.StatusError
		if !errors.As(err, &statusErr) || err != error(statusErr) {
			t.Fatalf("expected unwrapped *StatusError, got %T: %v", err, err)
		}
		if !apierrors.IsNotFound(err) {
			t.Fatalf("expected not found error, got %v", err)
		}
		expected := `gitrepositories.source.toolkit.fluxcd.io "missing" not found`
		if err.Error() != expected {
			t.Fatalf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("single name output unchanged", func(t *testing.T) {
		namespace, tmpl, _ := setupReconcileNames(t, objects)
		cmd := cmdTestCase{
			args:   "reconcile source git app-b -n " + namespace + " --timeout=30s",
			assert: assertGoldenTemplateFile("testdata/reconcile_names/git_single.golden", tmpl),
		}
		cmd.runTestCmd(t)
	})

	t.Run("with source reconciles a shared source once", func(t *testing.T) {
		namespace, tmpl, handled := setupReconcileNames(t, objects)
		cmd := cmdTestCase{
			args:   "reconcile kustomization ks-a ks-b ks-c --with-source -n " + namespace + " --timeout=30s",
			assert: assertGoldenTemplateFile("testdata/reconcile_names/ks_with_source.golden", tmpl),
		}
		cmd.runTestCmd(t)
		assertAnnotated(t, namespace, newKustomization, "ks-a", "ks-b", "ks-c")
		assertAnnotated(t, namespace, newGitRepo, "app-a", "app-b")
		assertHandled(t, handled, sourcev1.GitRepositoryKind, 1, "app-a", "app-b")
	})

	t.Run("with source reconciles a source shared through HelmCharts once", func(t *testing.T) {
		namespace, tmpl, handled := setupReconcileNames(t, "./testdata/reconcile_names/helm_objects.yaml")
		cmd := cmdTestCase{
			args:   "reconcile helmrelease hr-a hr-b --with-source -n " + namespace + " --timeout=30s",
			assert: assertGoldenTemplateFile("testdata/reconcile_names/hr_with_source.golden", tmpl),
		}
		cmd.runTestCmd(t)
		assertAnnotated(t, namespace, newHelmRelease, "hr-a", "hr-b")
		assertAnnotated(t, namespace, newHelmChart, "chart-a", "chart-b")
		assertAnnotated(t, namespace, newGitRepo, "charts")
		assertHandled(t, handled, helmv2.HelmReleaseKind, 1, "hr-a", "hr-b")
		assertHandled(t, handled, sourcev1.HelmChartKind, 1, "chart-a", "chart-b")
		assertHandled(t, handled, sourcev1.GitRepositoryKind, 1, "charts")
	})

	t.Run("suspended among several", func(t *testing.T) {
		namespace, tmpl, _ := setupReconcileNames(t, objects)
		cmd := cmdTestCase{
			args: "reconcile kustomization ks-a ks-suspended ks-b -n " + namespace + " --timeout=30s",
			assert: assertGoldenError("testdata/reconcile_names/ks_suspended.golden", tmpl,
				"failed to reconcile 1 of 3 Kustomization objects"),
		}
		cmd.runTestCmd(t)
		assertAnnotated(t, namespace, newKustomization, "ks-a", "ks-b")
		obj := &kustomizev1.Kustomization{}
		if err := testEnv.client.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: "ks-suspended"}, obj); err != nil {
			t.Fatal(err)
		}
		if v := obj.GetAnnotations()[meta.ReconcileRequestAnnotation]; v != "" {
			t.Errorf("suspended object was annotated")
		}
	})

	t.Run("single suspended name keeps the error", func(t *testing.T) {
		namespace, _, _ := setupReconcileNames(t, objects)
		cmd := cmdTestCase{
			args:   "reconcile kustomization ks-suspended -n " + namespace + " --timeout=30s",
			assert: assertError("resource is suspended"),
		}
		cmd.runTestCmd(t)
	})
}

// assertHandled checks that each named object got the expected number of
// reconcile requests.
func assertHandled(t *testing.T, handled func(kind, name string) int, kind string, expected int, names ...string) {
	t.Helper()
	for _, name := range names {
		if n := handled(kind, name); n != expected {
			t.Errorf("%s %s got %d reconcile requests, expected %d", kind, name, n, expected)
		}
	}
}

func TestReconcileNamesTimeout(t *testing.T) {
	// no fake reconciler: every wait ends with a timeout
	oldTimeout := rootArgs.timeout
	t.Cleanup(func() { rootArgs.timeout = oldTimeout })
	namespace := allocateNamespace("reconcile-names-timeout")
	tmpl := map[string]string{"ns": namespace}
	testEnv.CreateObjectFile("./testdata/reconcile_names/objects.yaml", tmpl, t)

	t.Run("several names keep going after a timeout", func(t *testing.T) {
		cmd := cmdTestCase{
			args: "reconcile source git app-a app-b --timeout=1s -n " + namespace,
			assert: assertGoldenError("testdata/reconcile_names/git_timeout.golden", tmpl,
				"failed to reconcile 2 of 2 GitRepository objects"),
		}
		cmd.runTestCmd(t)
		assertAnnotated(t, namespace, newGitRepo, "app-a", "app-b")
	})

	t.Run("single name returns the timeout error", func(t *testing.T) {
		_, err := executeCommand("reconcile source git app-a --timeout=1s -n " + namespace)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context deadline exceeded, got %T: %v", err, err)
		}
	})
}
