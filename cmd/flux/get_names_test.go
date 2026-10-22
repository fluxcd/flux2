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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// assertGoldenFileWithError checks the output against a golden file
// and expects the command to fail with the given error.
func assertGoldenFileWithError(goldenFile, expectedErr string) assertFunc {
	return assert(
		assertError(expectedErr),
		func(output string, err error) error {
			want, readErr := os.ReadFile(goldenFile)
			if readErr != nil {
				return fmt.Errorf("Error reading golden file '%s': %s", goldenFile, readErr)
			}
			if diff := cmp.Diff(string(want), output); diff != "" {
				return fmt.Errorf("Mismatch from golden file '%s' (-want +got):\n%s", goldenFile, diff)
			}
			return nil
		})
}

func Test_GetCmdNames(t *testing.T) {
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	testEnv.CreateObjectFile("./testdata/get_names/objects.yaml", tmpl, t)

	notFound := func(name string) string {
		return fmt.Sprintf("GitRepository object '%s' not found in %q namespace", name, tmpl["fluxns"])
	}

	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "single name",
			args:   "get sources git podinfo",
			assert: assertGoldenFile("testdata/get_names/one.golden"),
		},
		{
			name:   "single name not found",
			args:   "get sources git missing",
			assert: assertGoldenFileWithError("testdata/get_names/empty.golden", notFound("missing")),
		},
		{
			name:   "two names",
			args:   "get sources git podinfo podinfo-shard2",
			assert: assertGoldenFile("testdata/get_names/two.golden"),
		},
		{
			name:   "two names in the order given",
			args:   "get sources git podinfo-shard2 podinfo",
			assert: assertGoldenFile("testdata/get_names/two-reversed.golden"),
		},
		{
			name:   "duplicate name",
			args:   "get sources git podinfo podinfo",
			assert: assertGoldenFile("testdata/get_names/one.golden"),
		},
		{
			name:   "duplicate among several names",
			args:   "get sources git podinfo podinfo-shard2 podinfo",
			assert: assertGoldenFile("testdata/get_names/two.golden"),
		},
		{
			name: "one missing among several",
			args: "get sources git podinfo missing podinfo-shard2",
			assert: assertGoldenFileWithError("testdata/get_names/two.golden",
				notFound("missing")),
		},
		{
			name: "several missing among several",
			args: "get sources git missing1 podinfo missing2",
			assert: assertGoldenFileWithError("testdata/get_names/one.golden",
				notFound("missing1")+"\n"+notFound("missing2")),
		},
		{
			name: "all missing",
			args: "get sources git missing1 missing2",
			assert: assertGoldenFileWithError("testdata/get_names/empty.golden",
				notFound("missing1")+"\n"+notFound("missing2")),
		},
		{
			name:   "status selector filters out all names",
			args:   "get sources git podinfo podinfo-shard2 --status-selector ready=false",
			assert: assertGoldenFile("testdata/get_names/header.golden"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.args + " -n " + tmpl["fluxns"],
				assert: tt.assert,
			}
			cmd.runTestCmd(t)
		})
	}
}

func Test_GetCmdNamesAllNamespaces(t *testing.T) {
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	testEnv.CreateObjectFile("./testdata/get_names/objects.yaml", tmpl, t)

	output, err := executeCommand("get sources git podinfo podinfo-shard2 -A")
	if err != nil {
		t.Fatalf("get with several names across all namespaces failed: %v", err)
	}
	for _, name := range []string{"podinfo", "podinfo-shard2"} {
		if !hasRow(output, tmpl["fluxns"], name) {
			t.Errorf("expected a row for %s/%s in output:\n%s", tmpl["fluxns"], name, output)
		}
	}
	if hasRow(output, tmpl["fluxns"], "podinfo-shard1") {
		t.Errorf("unexpected row for podinfo-shard1 in output:\n%s", output)
	}

	_, err = executeCommand("get sources git podinfo missing -A")
	if want := "GitRepository object 'missing' not found in any namespace"; err == nil || err.Error() != want {
		t.Errorf("expected error %q, got %v", want, err)
	}
}

func Test_GetAllCmdNames(t *testing.T) {
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	testEnv.CreateObjectFile("./testdata/get_names/objects.yaml", tmpl, t)

	output, err := executeCommand("get sources all podinfo podinfo-shard2 -n " + tmpl["fluxns"])
	if err != nil {
		t.Fatalf("get sources all with several names failed: %v", err)
	}
	for _, name := range []string{"gitrepository/podinfo", "gitrepository/podinfo-shard2"} {
		if !hasRow(output, name) {
			t.Errorf("expected a row for %s in output:\n%s", name, output)
		}
	}
	if hasRow(output, "gitrepository/podinfo-shard1") {
		t.Errorf("unexpected row for podinfo-shard1 in output:\n%s", output)
	}
}

func Test_GetAllCmdNamesMissing(t *testing.T) {
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	testEnv.CreateObjectFile("./testdata/get_names/objects.yaml", tmpl, t)

	notFound := func(name string) string {
		return fmt.Sprintf("object '%s' not found in %q namespace", name, tmpl["fluxns"])
	}

	for _, verb := range []string{"get all", "get sources all", "get images all"} {
		t.Run(verb, func(t *testing.T) {
			_, err := executeCommand(verb + " missing-a missing-b -n " + tmpl["fluxns"])
			if want := notFound("missing-a") + "\n" + notFound("missing-b"); err == nil || err.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}

			_, err = executeCommand(verb + " missing-a missing-a -n " + tmpl["fluxns"])
			if err != nil {
				t.Errorf("expected single-name behaviour for a duplicate name, got %v", err)
			}
		})
	}

	for _, verb := range []string{"get all", "get sources all"} {
		t.Run(verb+" one missing", func(t *testing.T) {
			output, err := executeCommand(verb + " podinfo missing-a -n " + tmpl["fluxns"])
			if want := notFound("missing-a"); err == nil || err.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if !hasRow(output, "gitrepository/podinfo") {
				t.Errorf("expected a row for gitrepository/podinfo in output:\n%s", output)
			}
			if strings.Contains(output, "not found") {
				t.Errorf("unexpected per-kind not found message in output:\n%s", output)
			}
		})
	}
}

// Test_GetCmdNamesRBAC checks that several names are fetched with
// name-scoped requests, allowed by a Role limited by resourceNames.
func Test_GetCmdNamesRBAC(t *testing.T) {
	ns := allocateNamespace("flux-system")
	tmpl := map[string]string{"fluxns": ns}
	testEnv.CreateObjectFile("./testdata/get_names/objects.yaml", tmpl, t)

	ctx := context.Background()
	userName := "get-names-" + ns
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: userName, Namespace: ns},
		Rules: []rbacv1.PolicyRule{{
			APIGroups:     []string{"source.toolkit.fluxcd.io"},
			Resources:     []string{"gitrepositories"},
			Verbs:         []string{"get", "list", "watch"},
			ResourceNames: []string{"podinfo", "podinfo-shard2"},
		}},
	}
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: userName, Namespace: ns},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: userName},
		Subjects:   []rbacv1.Subject{{APIGroup: "rbac.authorization.k8s.io", Kind: "User", Name: userName}},
	}
	for _, obj := range []client.Object{role, binding} {
		if err := testEnv.client.Create(ctx, obj); err != nil {
			t.Fatalf("failed to create %T: %v", obj, err)
		}
	}

	user, err := testEnv.testEnv.ControlPlane.AddUser(envtest.User{Name: userName}, nil)
	if err != nil {
		t.Fatalf("failed to add user: %v", err)
	}
	kubeConfig, err := user.KubeConfig()
	if err != nil {
		t.Fatalf("failed to get user kubeconfig: %v", err)
	}
	kubeConfigPath := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(kubeConfigPath, kubeConfig, 0o600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}
	prev := kubeconfigArgs.KubeConfig
	kubeconfigArgs.KubeConfig = &kubeConfigPath
	t.Cleanup(func() { kubeconfigArgs.KubeConfig = prev })

	// The Role does not allow listing all objects.
	if _, err := executeCommand("get sources git -n " + ns); err == nil || !apierrors.IsForbidden(err) {
		t.Fatalf("expected a forbidden error without names, got %v", err)
	}

	cmd := cmdTestCase{
		args:   "get sources git podinfo podinfo-shard2 -n " + ns,
		assert: assertGoldenFile("testdata/get_names/two.golden"),
	}
	cmd.runTestCmd(t)

	// The watch runs until the timeout; it must not be forbidden.
	_, err = executeCommand("get sources git podinfo podinfo-shard2 --watch --timeout 2s -n " + ns)
	if err == nil || apierrors.IsForbidden(err) || strings.Contains(err.Error(), "forbidden") {
		t.Errorf("expected the watch to time out, got %v", err)
	}
}

// hasRow reports whether output has a table row whose first columns are cols.
func hasRow(output string, cols ...string) bool {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= len(cols) && slices.Equal(fields[:len(cols)], cols) {
			return true
		}
	}
	return false
}
