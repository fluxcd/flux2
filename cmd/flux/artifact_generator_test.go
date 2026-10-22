//go:build unit

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
	"fmt"
	"testing"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/apis/meta"
	swapi "github.com/fluxcd/source-watcher/api/v2/v1beta1"
)

// fakeArtifactGeneratorReconciler handles the reconcile requests of the
// ArtifactGenerators in the namespace. Like source-watcher, it records a
// request as handled even when reconciliation is disabled, but only marks
// enabled objects as Ready for their current generation.
func fakeArtifactGeneratorReconciler(t *testing.T, namespace string) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		<-done
	})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}

			var list swapi.ArtifactGeneratorList
			if err := testEnv.client.List(ctx, &list, client.InNamespace(namespace)); err != nil {
				continue
			}
			for i := range list.Items {
				obj := &list.Items[i]
				requested := obj.GetAnnotations()[meta.ReconcileRequestAnnotation]
				if requested == "" || requested == obj.Status.LastHandledReconcileAt {
					continue
				}
				obj.Status.LastHandledReconcileAt = requested
				if !obj.IsDisabled() {
					apimeta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
						Type:               meta.ReadyCondition,
						Status:             metav1.ConditionTrue,
						ObservedGeneration: obj.GetGeneration(),
						Reason:             meta.SucceededReason,
						Message:            "reconciled",
					})
				}
				_ = testEnv.client.Status().Update(ctx, obj)
			}
		}
	}()
}

func getArtifactGenerator(namespace, name string) (*swapi.ArtifactGenerator, error) {
	obj := &swapi.ArtifactGenerator{}
	err := testEnv.client.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, obj)
	return obj, err
}

// assertArtifactGenerators runs check on the named objects after the command ran.
func assertArtifactGenerators(namespace string, names []string, check func(obj *swapi.ArtifactGenerator) error) assertFunc {
	return func(_ string, _ error) error {
		for _, name := range names {
			obj, err := getArtifactGenerator(namespace, name)
			if err != nil {
				return err
			}
			if err := check(obj); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		return nil
	}
}

func TestArtifactGeneratorCommands(t *testing.T) {
	oldTimeout := rootArgs.timeout
	t.Cleanup(func() { rootArgs.timeout = oldTimeout })

	tmpl := map[string]string{
		"ns": allocateNamespace("artifact-generator"),
	}
	testEnv.CreateObjectFile("./testdata/artifact_generator/objects.yaml", tmpl, t)
	ns := tmpl["ns"]
	one := []string{"podinfo-gen"}
	both := []string{"podinfo-gen", "podinfo-gen-2"}

	isDisabled := func(names []string, want bool) assertFunc {
		return assertArtifactGenerators(ns, names, func(obj *swapi.ArtifactGenerator) error {
			if obj.IsDisabled() != want {
				return fmt.Errorf("expected disabled %v, got annotations %v", want, obj.GetAnnotations())
			}
			return nil
		})
	}
	isRequested := func(names []string) assertFunc {
		return assertArtifactGenerators(ns, names, func(obj *swapi.ArtifactGenerator) error {
			if obj.GetAnnotations()[meta.ReconcileRequestAnnotation] == "" {
				return fmt.Errorf("expected a reconcile request, got annotations %v", obj.GetAnnotations())
			}
			return nil
		})
	}
	requestHandled := func(names []string) assertFunc {
		return assertArtifactGenerators(ns, names, func(obj *swapi.ArtifactGenerator) error {
			requested := obj.GetAnnotations()[meta.ReconcileRequestAnnotation]
			if requested == "" || requested != obj.Status.LastHandledReconcileAt {
				return fmt.Errorf("expected reconcile request %q to be handled, got %q", requested, obj.Status.LastHandledReconcileAt)
			}
			return nil
		})
	}

	// lastHandled records the handled request before a command runs, so
	// that newRequestHandled can tell that the command made a new one.
	lastHandled := map[string]string{}
	recordHandled := func(names []string) func(t *testing.T) {
		return func(t *testing.T) {
			for _, name := range names {
				obj, err := getArtifactGenerator(ns, name)
				if err != nil {
					t.Fatal(err)
				}
				lastHandled[name] = obj.Status.LastHandledReconcileAt
			}
		}
	}
	newRequestHandled := func(names []string) assertFunc {
		return assert(requestHandled(names), assertArtifactGenerators(ns, names, func(obj *swapi.ArtifactGenerator) error {
			if obj.Status.LastHandledReconcileAt == lastHandled[obj.GetName()] {
				return fmt.Errorf("expected a new reconcile request to be handled, got %q", obj.Status.LastHandledReconcileAt)
			}
			return nil
		}))
	}
	withReconciler := func(before func(t *testing.T)) func(t *testing.T) {
		return func(t *testing.T) {
			if before != nil {
				before(t)
			}
			fakeArtifactGeneratorReconciler(t, ns)
		}
	}

	// The cases run in order against the same objects.
	cases := []struct {
		name   string
		args   string
		before func(t *testing.T)
		assert assertFunc
	}{
		{
			name:   "suspend without name",
			args:   "suspend artifact generator -n " + ns,
			assert: assertError("artifactgenerator name is required"),
		},
		{
			name:   "resume without name",
			args:   "resume artifact generator -n " + ns,
			assert: assertError("artifactgenerator name is required"),
		},
		{
			name:   "reconcile without name",
			args:   "reconcile artifact generator -n " + ns,
			assert: assertError("ArtifactGenerator name is required"),
		},
		{
			name:   "suspend missing object",
			args:   "suspend artifact generator missing -n " + ns,
			assert: assertGoldenTemplateFile("testdata/artifact_generator/suspend_missing.golden", tmpl),
		},
		{
			name:   "suspend",
			args:   "suspend ag/podinfo-gen -n " + ns,
			assert: assert(assertGoldenTemplateFile("testdata/artifact_generator/suspend.golden", tmpl), isDisabled(one, true)),
		},
		{
			name:   "suspend duplicate names",
			args:   "suspend ag/podinfo-gen ag/podinfo-gen -n " + ns,
			assert: assert(assertGoldenTemplateFile("testdata/artifact_generator/suspend.golden", tmpl), isDisabled(one, true)),
		},
		{
			name:   "reconcile suspended object",
			args:   "reconcile artifact generator podinfo-gen -n " + ns,
			assert: assertError("resource is suspended"),
		},
		{
			// The Ready condition from before the suspension must not be
			// trusted until the controller handles the reconcile request.
			name:   "resume without controller",
			args:   "resume artifact generator podinfo-gen --timeout=1s -n " + ns,
			assert: assert(assertError("reconciliation failed for 1 ArtifactGenerator(s)"), isDisabled(one, false), isRequested(one)),
		},
		{
			name:   "suspend all",
			args:   "suspend artifact generator --all -n " + ns,
			assert: assert(assertGoldenTemplateFile("testdata/artifact_generator/suspend_two.golden", tmpl), isDisabled(both, true)),
		},
		{
			name:   "resume",
			args:   "resume ag/podinfo-gen --timeout=30s -n " + ns,
			before: withReconciler(recordHandled(one)),
			assert: assert(assertGoldenTemplateFile("testdata/artifact_generator/resume.golden", tmpl), isDisabled(one, false), newRequestHandled(one)),
		},
		{
			name:   "suspend several",
			args:   "suspend artifact generator podinfo-gen podinfo-gen-2 -n " + ns,
			assert: isDisabled(both, true),
		},
		{
			// With several names, resume does not wait unless --wait is set.
			name:   "resume several",
			args:   "resume artifact generator podinfo-gen podinfo-gen-2 -n " + ns,
			assert: assert(assertGoldenTemplateFile("testdata/artifact_generator/resume_two.golden", tmpl), isDisabled(both, false), isRequested(both)),
		},
		{
			name:   "suspend all again",
			args:   "suspend artifact generator --all -n " + ns,
			assert: isDisabled(both, true),
		},
		{
			// Without names, resume does not wait unless --wait is set.
			name:   "resume all",
			args:   "resume artifact generator --all -n " + ns,
			assert: assert(assertGoldenTemplateFile("testdata/artifact_generator/resume_two.golden", tmpl), isDisabled(both, false), isRequested(both)),
		},
		{
			name:   "reconcile",
			args:   "reconcile ag/podinfo-gen --timeout=30s -n " + ns,
			before: withReconciler(recordHandled(one)),
			assert: assert(assertSuccess(), newRequestHandled(one)),
		},
		{
			name:   "reconcile several",
			args:   "reconcile artifact generator podinfo-gen podinfo-gen-2 --timeout=30s -n " + ns,
			before: withReconciler(recordHandled(both)),
			assert: assert(assertSuccess(), newRequestHandled(both)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.before != nil {
				tc.before(t)
			}
			cmd := cmdTestCase{
				args:   tc.args,
				assert: tc.assert,
			}
			cmd.runTestCmd(t)
		})
	}
}

// TestArtifactGeneratorWait checks the resume and reconcile waits against
// states that kstatus alone accepts, as ArtifactGenerator has no
// status.observedGeneration.
func TestArtifactGeneratorWait(t *testing.T) {
	ctx := context.Background()
	tmpl := map[string]string{
		"ns": allocateNamespace("artifact-generator-wait"),
	}
	testEnv.CreateObjectFile("./testdata/artifact_generator/objects.yaml", tmpl, t)
	key := types.NamespacedName{Namespace: tmpl["ns"], Name: "podinfo-gen"}

	const requestedAt = "2026-10-22T00:00:00Z"

	// setState sets the reconcile request and the disabled annotation,
	// bumps the generation if newGeneration is set, and records the
	// request as handled with Ready=True for readyGeneration, or for the
	// current generation if readyGeneration is 0.
	setState := func(t *testing.T, disabled, newGeneration bool, readyGeneration int64) {
		t.Helper()
		obj := &swapi.ArtifactGenerator{}
		if err := testEnv.client.Get(ctx, key, obj); err != nil {
			t.Fatal(err)
		}
		annotations := map[string]string{meta.ReconcileRequestAnnotation: requestedAt}
		if disabled {
			annotations[swapi.ReconcileAnnotation] = swapi.DisabledValue
		}
		obj.SetAnnotations(annotations)
		if newGeneration {
			obj.Spec.OutputArtifacts[0].Name = fmt.Sprintf("podinfo-%d", obj.GetGeneration()+1)
		}
		if err := testEnv.client.Update(ctx, obj); err != nil {
			t.Fatal(err)
		}
		if readyGeneration == 0 {
			readyGeneration = obj.GetGeneration()
		}
		obj.Status.LastHandledReconcileAt = requestedAt
		apimeta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
			Type:               meta.ReadyCondition,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: readyGeneration,
			Reason:             meta.SucceededReason,
			Message:            "reconciled",
		})
		if err := testEnv.client.Status().Update(ctx, obj); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name    string
		setup   func(t *testing.T)
		want    bool
		wantErr string
	}{
		{
			name:  "ready for the current generation",
			setup: func(t *testing.T) { setState(t, false, false, 0) },
			want:  true,
		},
		{
			name:  "ready for an older generation",
			setup: func(t *testing.T) { setState(t, false, true, 1) },
			want:  false,
		},
		{
			name:    "suspended again before the request is handled",
			setup:   func(t *testing.T) { setState(t, true, false, 0) },
			wantErr: "resource is suspended",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			check := func(name string, got bool, err error) {
				t.Helper()
				if tt.wantErr != "" {
					if err == nil || err.Error() != tt.wantErr {
						t.Errorf("%s: expected error %q, got %v", name, tt.wantErr, err)
					}
					return
				}
				if err != nil {
					t.Fatalf("%s: unexpected error: %v", name, err)
				}
				if got != tt.want {
					t.Errorf("%s: expected %v, got %v", name, tt.want, got)
				}
			}

			resumeObj := &artifactGeneratorAdapter{&swapi.ArtifactGenerator{}}
			got, err := reconcileRequestHandledConditionFunc(testEnv.client, key, resumeObj, requestedAt)(ctx)
			check("resume", got, err)

			reconcileObj := artifactGeneratorAdapter{&swapi.ArtifactGenerator{}}
			got, err = reconciliationHandled(testEnv.client, key, reconcileObj, "")(ctx)
			check("reconcile", got, err)
		})
	}
}
