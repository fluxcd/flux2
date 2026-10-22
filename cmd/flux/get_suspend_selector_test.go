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
	"io"
	"os"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	autov1 "github.com/fluxcd/image-automation-controller/api/v1"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
	notificationv1b3 "github.com/fluxcd/notification-controller/api/v1beta3"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	swapi "github.com/fluxcd/source-watcher/api/v2/v1beta1"
)

type suspendedItemReporter interface {
	isSuspended(i int) bool
}

func TestGetAdaptersReportSuspension(t *testing.T) {
	oci := sourcev1.OCIRepository{}
	oci.Spec.Suspend = true
	bucket := sourcev1.Bucket{}
	bucket.Spec.Suspend = true
	gitRepository := sourcev1.GitRepository{}
	gitRepository.Spec.Suspend = true
	helmRepository := sourcev1.HelmRepository{}
	helmRepository.Spec.Suspend = true
	helmChart := sourcev1.HelmChart{}
	helmChart.Spec.Suspend = true
	helmRelease := helmv2.HelmRelease{}
	helmRelease.Spec.Suspend = true
	kustomization := kustomizev1.Kustomization{}
	kustomization.Spec.Suspend = true
	receiver := notificationv1.Receiver{}
	receiver.Spec.Suspend = true
	alert := notificationv1b3.Alert{}
	alert.Spec.Suspend = true
	provider := notificationv1b3.Provider{}
	provider.Spec.Suspend = true
	imageRepository := imagev1.ImageRepository{}
	imageRepository.Spec.Suspend = true
	imagePolicy := imagev1.ImagePolicy{}
	imagePolicy.Spec.Suspend = true
	imageUpdate := autov1.ImageUpdateAutomation{}
	imageUpdate.Spec.Suspend = true
	disabledGenerator := swapi.ArtifactGenerator{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				swapi.ReconcileAnnotation: strings.ToUpper(swapi.DisabledValue),
			},
		},
	}

	tests := []struct {
		name     string
		list     suspendedItemReporter
		expected []bool
	}{
		{name: "OCIRepository", list: &ociRepositoryListAdapter{&sourcev1.OCIRepositoryList{Items: []sourcev1.OCIRepository{{}, oci}}}, expected: []bool{false, true}},
		{name: "Bucket", list: &bucketListAdapter{&sourcev1.BucketList{Items: []sourcev1.Bucket{{}, bucket}}}, expected: []bool{false, true}},
		{name: "GitRepository", list: &gitRepositoryListAdapter{&sourcev1.GitRepositoryList{Items: []sourcev1.GitRepository{{}, gitRepository}}}, expected: []bool{false, true}},
		{name: "HelmRepository", list: &helmRepositoryListAdapter{&sourcev1.HelmRepositoryList{Items: []sourcev1.HelmRepository{{}, helmRepository}}}, expected: []bool{false, true}},
		{name: "HelmChart", list: &helmChartListAdapter{&sourcev1.HelmChartList{Items: []sourcev1.HelmChart{{}, helmChart}}}, expected: []bool{false, true}},
		{name: "ExternalArtifact", list: &externalArtifactListAdapter{&sourcev1.ExternalArtifactList{Items: []sourcev1.ExternalArtifact{{}, {}}}}, expected: []bool{false, false}},
		{name: "HelmRelease", list: helmReleaseListAdapter{&helmv2.HelmReleaseList{Items: []helmv2.HelmRelease{{}, helmRelease}}}, expected: []bool{false, true}},
		{name: "Kustomization", list: kustomizationListAdapter{&kustomizev1.KustomizationList{Items: []kustomizev1.Kustomization{{}, kustomization}}}, expected: []bool{false, true}},
		{name: "Receiver", list: receiverListAdapter{&notificationv1.ReceiverList{Items: []notificationv1.Receiver{{}, receiver}}}, expected: []bool{false, true}},
		{name: "Alert", list: alertListAdapter{&notificationv1b3.AlertList{Items: []notificationv1b3.Alert{{}, alert}}}, expected: []bool{false, true}},
		{name: "Provider", list: alertProviderListAdapter{&notificationv1b3.ProviderList{Items: []notificationv1b3.Provider{{}, provider}}}, expected: []bool{false, true}},
		{name: "ImageRepository", list: imageRepositoryListAdapter{&imagev1.ImageRepositoryList{Items: []imagev1.ImageRepository{{}, imageRepository}}}, expected: []bool{false, true}},
		{name: "ImagePolicy", list: imagePolicyListAdapter{&imagev1.ImagePolicyList{Items: []imagev1.ImagePolicy{{}, imagePolicy}}}, expected: []bool{false, true}},
		{name: "ImageUpdateAutomation", list: imageUpdateAutomationListAdapter{&autov1.ImageUpdateAutomationList{Items: []autov1.ImageUpdateAutomation{{}, imageUpdate}}}, expected: []bool{false, true}},
		{name: "ArtifactGenerator", list: artifactGeneratorListAdapter{&swapi.ArtifactGeneratorList{Items: []swapi.ArtifactGenerator{{}, disabledGenerator}}}, expected: []bool{false, true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, expected := range tt.expected {
				if got := tt.list.isSuspended(i); got != expected {
					t.Fatalf("isSuspended(%d) = %v, expected %v", i, got, expected)
				}
			}
		})
	}
}

func TestWatchUntilSuspendSelector(t *testing.T) {
	tests := []struct {
		name          string
		apiType       apiType
		derive        deriveType
		active        runtime.Object
		suspended     runtime.Object
		activeName    string
		suspendedName string
	}{
		{
			name:    "GitRepository",
			apiType: gitRepositoryType,
			derive: func(obj runtime.Object) (summarisable, error) {
				item, ok := obj.(*sourcev1.GitRepository)
				if !ok {
					return nil, fmt.Errorf("unexpected type %T", obj)
				}
				return &gitRepositoryListAdapter{&sourcev1.GitRepositoryList{Items: []sourcev1.GitRepository{*item}}}, nil
			},
			active: &sourcev1.GitRepository{
				ObjectMeta: metav1.ObjectMeta{Name: "git-active"},
			},
			suspended: &sourcev1.GitRepository{
				ObjectMeta: metav1.ObjectMeta{Name: "git-suspended"},
				Spec:       sourcev1.GitRepositorySpec{Suspend: true},
			},
			activeName:    "git-active",
			suspendedName: "git-suspended",
		},
		{
			name:    "ArtifactGenerator",
			apiType: artifactGeneratorType,
			derive: func(obj runtime.Object) (summarisable, error) {
				item, ok := obj.(*swapi.ArtifactGenerator)
				if !ok {
					return nil, fmt.Errorf("unexpected type %T", obj)
				}
				return artifactGeneratorListAdapter{&swapi.ArtifactGeneratorList{Items: []swapi.ArtifactGenerator{*item}}}, nil
			},
			active: &swapi.ArtifactGenerator{
				ObjectMeta: metav1.ObjectMeta{Name: "generator-active"},
			},
			suspended: &swapi.ArtifactGenerator{
				ObjectMeta: metav1.ObjectMeta{
					Name: "generator-disabled",
					Annotations: map[string]string{
						swapi.ReconcileAnnotation: swapi.DisabledValue,
					},
				},
			},
			activeName:    "generator-active",
			suspendedName: "generator-disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			get := &getCommand{apiType: tt.apiType, funcMap: make(typeMap)}
			if err := get.funcMap.registerCommand(tt.apiType.kind, tt.derive); err != nil {
				t.Fatalf("failed registering watch sink: %v", err)
			}

			previousArgs := getArgs
			getArgs = newGetFlags()
			getArgs.noHeader = true
			if err := getArgs.suspendSelector.Set("suspended"); err != nil {
				t.Fatalf("failed setting suspend selector: %v", err)
			}
			t.Cleanup(func() {
				getArgs = previousArgs
			})

			output := captureWatchOutput(t, get, tt.active, tt.suspended)
			requireOutputContains(t, output, tt.suspendedName)
			requireOutputExcludes(t, output, tt.activeName)
		})
	}
}

func captureWatchOutput(t *testing.T, get *getCommand, events ...runtime.Object) string {
	t.Helper()

	fake := watch.NewFakeWithChanSize(len(events), false)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed creating stdout pipe: %v", err)
	}
	previousStdout := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = previousStdout
		_ = reader.Close()
		_ = writer.Close()
	})

	done := make(chan error, 1)
	go func() {
		_, err := watchUntil(context.Background(), fake, get)
		done <- err
	}()
	for _, event := range events {
		fake.Add(event)
	}
	fake.Stop()
	var watchErr error
	select {
	case watchErr = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for watch to stop")
	}
	if !errors.Is(watchErr, watchtools.ErrWatchClosed) {
		t.Fatalf("watchUntil() error = %v, expected %v", watchErr, watchtools.ErrWatchClosed)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed closing stdout writer: %v", err)
	}
	os.Stdout = previousStdout
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed reading stdout: %v", err)
	}
	return string(output)
}

func TestGetCmdSuspendSelector(t *testing.T) {
	tmpl := createSuspendSelectorObjects(t)
	baseCommand := "get sources git -n " + tmpl["fluxns"]

	unfiltered := runGetCommand(t, baseCommand)
	requireOutputContains(t, unfiltered, "active-ready", "active-failed", "suspended-ready", "suspended-failed")
	any := runGetCommand(t, baseCommand+" --suspend-selector any")
	if any != unfiltered {
		t.Fatalf("expected explicit any output to match default:\nany:\n%s\ndefault:\n%s", any, unfiltered)
	}
	requireOutputContains(t, any, "active-ready", "active-failed", "suspended-ready", "suspended-failed")

	suspended := runGetCommand(t, baseCommand+" --suspend-selector suspended")
	requireOutputContains(t, suspended, "suspended-ready", "suspended-failed")
	requireOutputExcludes(t, suspended, "active-ready", "active-failed")

	active := runGetCommand(t, baseCommand+" --suspend-selector active")
	requireOutputContains(t, active, "active-ready", "active-failed")
	requireOutputExcludes(t, active, "suspended-ready", "suspended-failed")

	intersection := runGetCommand(t, baseCommand+
		" --label-selector test.fluxcd.io/group=blue"+
		" --status-selector Ready=True"+
		" --suspend-selector suspended")
	requireOutputContains(t, intersection, "suspended-ready")
	requireOutputExcludes(t, intersection, "active-ready", "active-failed", "suspended-failed")

	noMatch := runGetCommand(t, baseCommand+
		" --label-selector test.fluxcd.io/group=blue"+
		" --status-selector Ready=False"+
		" --suspend-selector suspended")
	if noMatch == "" {
		t.Fatal("expected a header-only table for a single-kind selector with no matches")
	}
	requireOutputExcludes(t, noMatch, "active-ready", "active-failed", "suspended-ready", "suspended-failed")

	namedMismatch := runGetCommand(t, "get sources git active-ready -n "+tmpl["fluxns"]+" --suspend-selector suspended")
	if namedMismatch == "" {
		t.Fatal("expected a header-only table for a named object excluded by the selector")
	}
	requireOutputExcludes(t, namedMismatch, "active-ready")

	// executeCommand must restore the default after a preceding explicit value.
	if reset := runGetCommand(t, baseCommand); reset != unfiltered {
		t.Fatalf("expected selector to reset to any:\nreset:\n%s\ndefault:\n%s", reset, unfiltered)
	}

	_, err := executeCommand("get all --suspend-selector unknown --kubeconfig /does/not/exist")
	if err == nil || !strings.Contains(err.Error(), "unsupported suspend selector 'unknown'") {
		t.Fatalf("expected invalid selector error before cluster access, got %v", err)
	}
}

func TestGetCmdSuspendSelectorSpecialResources(t *testing.T) {
	tmpl := createSuspendSelectorObjects(t)

	provider := runGetCommand(t, "get alert-providers -n "+tmpl["fluxns"]+" --suspend-selector suspended")
	requireOutputContains(t, provider, "provider-suspended")
	requireOutputExcludes(t, provider, "provider-active")
	providerActive := runGetCommand(t, "get alert-providers -n "+tmpl["fluxns"]+" --suspend-selector active")
	requireOutputContains(t, providerActive, "provider-active")
	requireOutputExcludes(t, providerActive, "provider-suspended")

	policy := runGetCommand(t, "get image policy -n "+tmpl["fluxns"]+" --suspend-selector suspended")
	requireOutputContains(t, policy, "policy-suspended")
	requireOutputExcludes(t, policy, "policy-active")
	policyActive := runGetCommand(t, "get image policy -n "+tmpl["fluxns"]+" --suspend-selector active")
	requireOutputContains(t, policyActive, "policy-active")
	requireOutputExcludes(t, policyActive, "policy-suspended")

	generators := runGetCommand(t, "get artifact generators -n "+tmpl["fluxns"]+" --suspend-selector suspended")
	requireOutputContains(t, generators, "generator-disabled")
	requireOutputExcludes(t, generators, "generator-active")
	generatorsActive := runGetCommand(t, "get artifact generators -n "+tmpl["fluxns"]+" --suspend-selector active")
	requireOutputContains(t, generatorsActive, "generator-active")
	requireOutputExcludes(t, generatorsActive, "generator-disabled")

	external := runGetCommand(t, "get sources external -n "+tmpl["fluxns"]+" --suspend-selector active")
	requireOutputContains(t, external, "external-active")
	externalSuspended := runGetCommand(t, "get sources external -n "+tmpl["fluxns"]+" --suspend-selector suspended")
	requireOutputExcludes(t, externalSuspended, "external-active")
}

func TestGetAllCmdSuspendSelector(t *testing.T) {
	tmpl := createSuspendSelectorObjects(t)
	preservationTests := []struct {
		command  string
		expected []string
	}{
		{command: "get sources all", expected: []string{"active-ready", "suspended-ready", "external-active"}},
		{command: "get images all", expected: []string{"policy-active", "policy-suspended"}},
		{command: "get all", expected: []string{"active-ready", "suspended-ready", "policy-active", "policy-suspended", "provider-active", "provider-suspended"}},
	}
	for _, tt := range preservationTests {
		t.Run(tt.command+" any preserves default", func(t *testing.T) {
			baseCommand := tt.command + " -n " + tmpl["fluxns"]
			unfiltered := runGetCommand(t, baseCommand)
			requireOutputContains(t, unfiltered, tt.expected...)
			any := runGetCommand(t, baseCommand+" --suspend-selector any")
			if any != unfiltered {
				t.Fatalf("expected explicit any output to match default for %q:\nany:\n%s\ndefault:\n%s", tt.command, any, unfiltered)
			}
		})
	}

	sourcesSuspended := runGetCommand(t, "get sources all -n "+tmpl["fluxns"]+" --suspend-selector suspended")
	requireOutputContains(t, sourcesSuspended, "suspended-ready", "suspended-failed")
	requireOutputExcludes(t, sourcesSuspended, "active-ready", "active-failed", "external-active")

	sourcesActive := runGetCommand(t, "get sources all -n "+tmpl["fluxns"]+" --suspend-selector active")
	requireOutputContains(t, sourcesActive, "active-ready", "active-failed", "external-active")
	requireOutputExcludes(t, sourcesActive, "suspended-ready", "suspended-failed")

	imagesSuspended := runGetCommand(t, "get images all -n "+tmpl["fluxns"]+" --suspend-selector suspended")
	requireOutputContains(t, imagesSuspended, "policy-suspended")
	requireOutputExcludes(t, imagesSuspended, "policy-active")
	imagesActive := runGetCommand(t, "get images all -n "+tmpl["fluxns"]+" --suspend-selector active")
	requireOutputContains(t, imagesActive, "policy-active")
	requireOutputExcludes(t, imagesActive, "policy-suspended")

	allSuspended := runGetCommand(t, "get all -n "+tmpl["fluxns"]+" --suspend-selector suspended")
	requireOutputContains(t, allSuspended, "suspended-ready", "suspended-failed", "policy-suspended", "provider-suspended")
	requireOutputExcludes(t, allSuspended, "active-ready", "active-failed", "policy-active", "provider-active", "external-active", "generator-disabled")
	allActive := runGetCommand(t, "get all -n "+tmpl["fluxns"]+" --suspend-selector active")
	requireOutputContains(t, allActive, "active-ready", "active-failed", "policy-active", "provider-active", "external-active")
	requireOutputExcludes(t, allActive, "suspended-ready", "suspended-failed", "policy-suspended", "provider-suspended", "generator-active")

	passiveActive := runGetCommand(t, "get sources all -n "+tmpl["passivens"]+" --suspend-selector active")
	requireOutputContains(t, passiveActive, "passive-only")
	passiveOnly := runGetCommand(t, "get sources all -n "+tmpl["passivens"]+" --suspend-selector suspended")
	if passiveOnly != "" {
		t.Fatalf("expected no aggregate output for a namespace without suspended resources, got:\n%s", passiveOnly)
	}
}

func createSuspendSelectorObjects(t *testing.T) map[string]string {
	t.Helper()
	tmpl := map[string]string{
		"fluxns":    allocateNamespace("suspend-selector"),
		"passivens": allocateNamespace("suspend-selector-passive"),
	}
	testEnv.CreateObjectFile("./testdata/get/suspend_objects.yaml", tmpl, t)
	return tmpl
}

func runGetCommand(t *testing.T, command string) string {
	t.Helper()
	output, err := executeCommand(command)
	if err != nil {
		t.Fatalf("%q failed: %v", command, err)
	}
	return output
}

func requireOutputContains(t *testing.T, output string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(output, value) {
			t.Errorf("expected output to contain %q:\n%s", value, output)
		}
	}
}

func requireOutputExcludes(t *testing.T, output string, values ...string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(output, value) {
			t.Errorf("expected output not to contain %q:\n%s", value, output)
		}
	}
}
