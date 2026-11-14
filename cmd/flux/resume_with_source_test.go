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
	"testing"

	"k8s.io/apimachinery/pkg/types"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

const resumeWithSourceObjects = "./testdata/resume_with_source/objects.yaml"

func assertUnsuspended(t *testing.T, namespace string, names ...string) {
	t.Helper()
	for _, name := range names {
		obj := &kustomizev1.Kustomization{}
		key := types.NamespacedName{Namespace: namespace, Name: name}
		if err := testEnv.client.Get(context.Background(), key, obj); err != nil {
			t.Fatalf("get Kustomization %s/%s: %v", namespace, name, err)
		}
		if obj.Spec.Suspend {
			t.Errorf("Kustomization %s/%s is still suspended", namespace, name)
		}
	}
}

func TestResumeKustomizationWithSource(t *testing.T) {
	t.Run("reconciles the source before resuming", func(t *testing.T) {
		namespace, _, handled := setupReconcileNames(t, resumeWithSourceObjects)
		cmd := cmdTestCase{
			args:   "resume kustomization ks-a ks-b --with-source -n " + namespace + " --timeout=30s",
			assert: assertSuccess(),
		}
		cmd.runTestCmd(t)

		assertAnnotated(t, namespace, newGitRepo, "app")
		assertHandled(t, handled, sourcev1.GitRepositoryKind, 1, "app")
		assertUnsuspended(t, namespace, "ks-a", "ks-b")
	})

	t.Run("reconciles a shared source once", func(t *testing.T) {
		namespace, _, handled := setupReconcileNames(t, resumeWithSourceObjects)
		cmd := cmdTestCase{
			args:   "resume kustomization ks-a ks-b --with-source -n " + namespace + " --timeout=30s",
			assert: assertSuccess(),
		}
		cmd.runTestCmd(t)

		assertHandled(t, handled, sourcev1.GitRepositoryKind, 1, "app")
	})

	t.Run("leaves the source alone without the flag", func(t *testing.T) {
		namespace, _, handled := setupReconcileNames(t, resumeWithSourceObjects)
		cmd := cmdTestCase{
			args:   "resume kustomization ks-a ks-b -n " + namespace + " --timeout=30s",
			assert: assertSuccess(),
		}
		cmd.runTestCmd(t)

		obj := &sourcev1.GitRepository{}
		key := types.NamespacedName{Namespace: namespace, Name: "app"}
		if err := testEnv.client.Get(context.Background(), key, obj); err != nil {
			t.Fatalf("get GitRepository %s/app: %v", namespace, err)
		}
		if v := obj.GetAnnotations()["reconcile.fluxcd.io/requestedAt"]; v != "" {
			t.Errorf("GitRepository %s/app was annotated without the flag: %q", namespace, v)
		}
		assertHandled(t, handled, sourcev1.GitRepositoryKind, 0, "app")
		assertUnsuspended(t, namespace, "ks-a", "ks-b")
	})
}
