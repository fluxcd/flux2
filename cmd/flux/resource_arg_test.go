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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	ssautil "github.com/fluxcd/pkg/ssa/utils"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

type normalizeCase struct {
	name    string
	args    []string
	want    []string
	wantErr string
}

func joinArgs(parts ...[]string) []string {
	var out []string
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// kindCases generates, for every table entry and both verbs, the
// TYPE/NAME, TYPE.GROUP and TYPE.VERSION.GROUP forms with and without
// a NAME.
func kindCases() []normalizeCase {
	var cases []normalizeCase
	for _, rt := range resourceTypes {
		kind, group, version := rt.gvk.Kind, rt.gvk.Group, rt.gvk.Version
		names := append([]string{strings.ToLower(kind), rt.plural, kind, strings.ToUpper(rt.plural)}, rt.shortNames...)
		qualified := []string{
			rt.plural + "." + group,
			rt.plural + "." + version + "." + group,
			strings.ToLower(kind) + "." + group,
			kind + "." + strings.ToUpper(version+"."+group),
		}
		for _, verb := range []*cobra.Command{getCmd, reconcileCmd} {
			v := verb.Name()
			path := rt.commandPath(verb)
			unsupported := fmt.Sprintf("flux %s does not support %s resources", v, kind)
			add := func(name string, args, rewritten []string) {
				c := normalizeCase{name: name, args: args}
				if path == nil {
					c.wantErr = unsupported
				} else {
					c.want = rewritten
				}
				cases = append(cases, c)
			}
			for _, typ := range append(names, qualified...) {
				add(v+" "+typ+"/x",
					[]string{v, typ + "/x"},
					joinArgs([]string{v}, path, []string{"x"}))
			}
			for _, typ := range qualified {
				add(v+" "+typ+" x",
					[]string{v, typ, "x"},
					joinArgs([]string{v}, path, []string{"x"}))
				add(v+" "+typ,
					[]string{v, typ},
					joinArgs([]string{v}, path))
			}
		}
	}
	return cases
}

func TestNormalizeResourceArgs(t *testing.T) {
	tests := []normalizeCase{
		// literal rewrites, independent of the table
		{name: "literal gitrepo", args: []string{"get", "gitrepo/x"}, want: []string{"get", "sources", "git", "x"}},
		{name: "literal imgpol", args: []string{"reconcile", "imgpol/x"}, want: []string{"reconcile", "image", "policy", "x"}},
		{name: "literal provider", args: []string{"get", "provider/x"}, want: []string{"get", "alert-providers", "x"}},
		{name: "literal ag", args: []string{"get", "ag/x"}, want: []string{"get", "artifacts", "generators", "x"}},
		{name: "literal hc", args: []string{"reconcile", "hc/x"}, want: []string{"reconcile", "source", "chart", "x"}},
		{name: "literal hr qualified", args: []string{"reconcile", "helmreleases.v2.helm.toolkit.fluxcd.io/x"},
			want: []string{"reconcile", "helmrelease", "x"}},

		// flags before the verb
		{name: "long flag before verb", args: []string{"--context", "c", "get", "hr/x"},
			want: []string{"--context", "c", "get", "helmreleases", "x"}},
		{name: "short flag before verb", args: []string{"-n", "ns", "reconcile", "gitrepo/x"},
			want: []string{"-n", "ns", "reconcile", "source", "git", "x"}},
		{name: "flag with equals before verb", args: []string{"--namespace=ns", "get", "imgpol/x"},
			want: []string{"--namespace=ns", "get", "images", "policy", "x"}},
		{name: "bool flag before verb", args: []string{"--verbose", "get", "ks/x"},
			want: []string{"--verbose", "get", "kustomizations", "x"}},

		// flags between verb and resource
		{name: "global flags after verb", args: []string{"get", "--context", "c", "-n", "ns", "hr/my-release"},
			want: []string{"get", "--context", "c", "-n", "ns", "helmreleases", "my-release"}},
		{name: "bool shorthand after verb", args: []string{"get", "-A", "hr/x"},
			want: []string{"get", "-A", "helmreleases", "x"}},
		{name: "bool long flag after verb", args: []string{"get", "--no-header", "hr/x"},
			want: []string{"get", "--no-header", "helmreleases", "x"}},
		{name: "value flag value looks like type/name", args: []string{"get", "--status-selector", "hr/x", "gitrepo/y"},
			want: []string{"get", "--status-selector", "hr/x", "sources", "git", "y"}},
		{name: "verb value flag after verb", args: []string{"get", "--status-selector", "ready=false", "ocirepo/x"},
			want: []string{"get", "--status-selector", "ready=false", "sources", "oci", "x"}},
		{name: "verb shorthand value flag after verb", args: []string{"get", "-l", "app=x", "gitrepo"},
			want: []string{"get", "-l", "app=x", "gitrepo"}},
		{name: "timeout after verb", args: []string{"reconcile", "--timeout", "1m", "ks/x"},
			want: []string{"reconcile", "--timeout", "1m", "kustomization", "x"}},
		{name: "shorthand group after verb", args: []string{"get", "-Aw", "hr/x"},
			want: []string{"get", "-Aw", "helmreleases", "x"}},
		{name: "shorthand with equals after verb", args: []string{"get", "-n=ns", "hr/x"},
			want: []string{"get", "-n=ns", "helmreleases", "x"}},

		// flags after the resource
		{name: "bool flag after resource", args: []string{"get", "hr/x", "--show-source"},
			want: []string{"get", "helmreleases", "x", "--show-source"}},
		{name: "value flag after resource", args: []string{"get", "hr/x", "--status-selector", "ready=false"},
			want: []string{"get", "helmreleases", "x", "--status-selector", "ready=false"}},
		{name: "short value flag after resource", args: []string{"get", "gitrepo/x", "-n", "ns"},
			want: []string{"get", "sources", "git", "x", "-n", "ns"}},
		{name: "context after resource", args: []string{"get", "imgrepo/x", "--context", "c"},
			want: []string{"get", "images", "repository", "x", "--context", "c"}},
		{name: "reconcile with-source", args: []string{"reconcile", "hr/podinfo", "--with-source"},
			want: []string{"reconcile", "helmrelease", "podinfo", "--with-source"}},
		{name: "reconcile timeout after resource", args: []string{"reconcile", "ks/x", "--timeout", "1m", "--with-source"},
			want: []string{"reconcile", "kustomization", "x", "--timeout", "1m", "--with-source"}},
		{name: "flags everywhere", args: []string{"-n", "ns", "get", "-A", "HelmRelease/x", "--status-selector", "ready=true", "--no-header"},
			want: []string{"-n", "ns", "get", "-A", "helmreleases", "x", "--status-selector", "ready=true", "--no-header"}},
		{name: "qualified type with flags then name", args: []string{"get", "helmreleases.helm.toolkit.fluxcd.io", "-n", "ns", "x"},
			want: []string{"get", "helmreleases", "-n", "ns", "x"}},

		// double dash
		{name: "double dash unchanged", args: []string{"get", "--", "hr/x"}, want: []string{"get", "--", "hr/x"}},
		{name: "name after double dash", args: []string{"get", "hr/a", "--", "b"},
			wantErr: `only one TYPE/NAME argument is supported, got "b"`},

		// case-insensitivity
		{name: "uppercase short name", args: []string{"get", "HR/x"}, want: []string{"get", "helmreleases", "x"}},
		{name: "mixed case plural qualified", args: []string{"reconcile", "GitRepositories.source.toolkit.fluxcd.io/x"},
			want: []string{"reconcile", "source", "git", "x"}},
		{name: "uppercase group", args: []string{"get", "hr.HELM.TOOLKIT.FLUXCD.IO/x"}, want: []string{"get", "helmreleases", "x"}},
		{name: "uppercase version and group", args: []string{"get", "ks.V1.Kustomize.Toolkit.Fluxcd.Io", "x"},
			want: []string{"get", "kustomizations", "x"}},
		{name: "kind name", args: []string{"get", "ImageUpdateAutomation/x"}, want: []string{"get", "images", "update", "x"}},
		{name: "name case preserved", args: []string{"get", "hr/MyRelease"}, want: []string{"get", "helmreleases", "MyRelease"}},

		// unchanged
		{name: "space form unchanged", args: []string{"get", "hr", "my-release"}, want: []string{"get", "hr", "my-release"}},
		{name: "space form multiple names unchanged", args: []string{"get", "hr", "a", "b"}, want: []string{"get", "hr", "a", "b"}},
		{name: "plain type unchanged", args: []string{"get", "gitrepo", "x"}, want: []string{"get", "gitrepo", "x"}},
		{name: "nested space form unchanged", args: []string{"get", "sources", "git", "x"}, want: []string{"get", "sources", "git", "x"}},
		{name: "reconcile space form unchanged", args: []string{"reconcile", "source", "oci", "x"},
			want: []string{"reconcile", "source", "oci", "x"}},
		{name: "unknown type unchanged", args: []string{"get", "foo/bar"}, want: []string{"get", "foo/bar"}},
		{name: "unknown qualified type unchanged", args: []string{"get", "foos.example.com/bar"}, want: []string{"get", "foos.example.com/bar"}},
		{name: "get all unchanged", args: []string{"get", "all/x"}, want: []string{"get", "all/x"}},
		{name: "no args unchanged", args: []string{"get"}, want: []string{"get"}},
		{name: "help unchanged", args: []string{"get", "--help"}, want: []string{"get", "--help"}},
		{name: "reconcile help unchanged", args: []string{"reconcile", "-h"}, want: []string{"reconcile", "-h"}},
		{name: "other command unchanged", args: []string{"suspend", "hr/x"}, want: []string{"suspend", "hr/x"}},
		{name: "other command with qualified type unchanged", args: []string{"resume", "helmreleases.helm.toolkit.fluxcd.io/x"},
			want: []string{"resume", "helmreleases.helm.toolkit.fluxcd.io/x"}},
		{name: "empty args unchanged", args: []string{}, want: []string{}},

		// wrong group or version
		{name: "flag as name", args: []string{"get", "hr/--all-namespaces"},
			wantErr: `invalid resource name "--all-namespaces" in "hr/--all-namespaces"`},
		{name: "flag with value as name", args: []string{"reconcile", "gitrepo/--namespace=prod"},
			wantErr: `invalid resource name "--namespace=prod" in "gitrepo/--namespace=prod"`},
		{name: "shorthand as name", args: []string{"get", "ks/-A"},
			wantErr: `invalid resource name "-A" in "ks/-A"`},
		{name: "dash as name", args: []string{"get", "hr/-"},
			wantErr: `invalid resource name "-" in "hr/-"`},
		{name: "empty group", args: []string{"get", "hr./x"},
			wantErr: `"hr." is not a known Flux resource type`},
		{name: "empty group without name", args: []string{"get", "hr.", "x"},
			wantErr: `"hr." is not a known Flux resource type`},
		{name: "unknown group", args: []string{"get", "helmreleases.wrong.group/x"},
			wantErr: `"helmreleases.wrong.group" is not a known Flux resource type`},
		{name: "other kind group", args: []string{"get", "gitrepositories.helm.toolkit.fluxcd.io/x"},
			wantErr: `"gitrepositories.helm.toolkit.fluxcd.io" is not a known Flux resource type`},
		{name: "unknown version", args: []string{"get", "helmreleases.v9.helm.toolkit.fluxcd.io/x"},
			wantErr: `"helmreleases.v9.helm.toolkit.fluxcd.io" is not a known Flux resource type`},
		{name: "unknown version without name", args: []string{"reconcile", "kustomizations.v1beta2.kustomize.toolkit.fluxcd.io", "x"},
			wantErr: `"kustomizations.v1beta2.kustomize.toolkit.fluxcd.io" is not a known Flux resource type`},
		{name: "version of another kind in group", args: []string{"get", "alerts.v1.notification.toolkit.fluxcd.io/x"},
			wantErr: `"alerts.v1.notification.toolkit.fluxcd.io" is not a known Flux resource type`},

		// unsupported verb
		{name: "reconcile alert", args: []string{"reconcile", "alert/x"},
			wantErr: "flux reconcile does not support Alert resources"},
		{name: "reconcile external artifact", args: []string{"reconcile", "ea/x"},
			wantErr: "flux reconcile does not support ExternalArtifact resources"},
		{name: "reconcile artifact generator", args: []string{"reconcile", "ag/x"},
			wantErr: "flux reconcile does not support ArtifactGenerator resources"},
		{name: "reconcile provider", args: []string{"reconcile", "alert-provider/x"},
			wantErr: "flux reconcile does not support Provider resources"},

		// multiple resources
		{name: "two type/name args", args: []string{"get", "hr/a", "hr/b"},
			wantErr: `only one TYPE/NAME argument is supported, got "hr/b"`},
		{name: "type/name then name", args: []string{"get", "hr/a", "b"},
			wantErr: `only one TYPE/NAME argument is supported, got "b"`},
		{name: "reconcile mixed kinds", args: []string{"reconcile", "gitrepo/a", "ks/b"},
			wantErr: `only one TYPE/NAME argument is supported, got "ks/b"`},
		{name: "extra arg after value flag", args: []string{"get", "hr/a", "--status-selector", "ready=false", "b"},
			wantErr: `only one TYPE/NAME argument is supported, got "b"`},
		{name: "extra arg after bool flag", args: []string{"reconcile", "hr/a", "--with-source", "b"},
			wantErr: `only one TYPE/NAME argument is supported, got "b"`},
		{name: "extra arg after sub-command bool flag", args: []string{"get", "hr/a", "--show-source", "b"},
			wantErr: `only one TYPE/NAME argument is supported, got "b"`},
		{name: "extra arg after verb bool shorthand", args: []string{"get", "hr/a", "-A", "b"},
			wantErr: `only one TYPE/NAME argument is supported, got "b"`},
		{name: "extra arg after local value flag", args: []string{"get", "hr/a", "-l", "app=x", "b"},
			wantErr: `only one TYPE/NAME argument is supported, got "b"`},
		{name: "bool and value shorthand group after resource", args: []string{"get", "hr/a", "-Al", "!app"},
			want: []string{"get", "helmreleases", "a", "-Al", "!app"}},
		{name: "bool and value shorthand group with namespace", args: []string{"get", "hr/a", "-An", "ns"},
			want: []string{"get", "helmreleases", "a", "-An", "ns"}},
		{name: "extra arg after shorthand group", args: []string{"get", "hr/a", "-An", "ns", "b"},
			wantErr: `only one TYPE/NAME argument is supported, got "b"`},
		{name: "help flag before value flag", args: []string{"get", "hr/a", "--help", "--namespace", "flux-system"},
			want: []string{"get", "helmreleases", "a", "--help", "--namespace", "flux-system"}},
		{name: "help shorthand before value flag", args: []string{"reconcile", "gitrepo/a", "-h", "-n", "flux-system"},
			want: []string{"reconcile", "source", "git", "a", "-h", "-n", "flux-system"}},
		{name: "missing flag value left for cobra", args: []string{"get", "hr/a", "-n"},
			want: []string{"get", "helmreleases", "a", "-n"}},
		{name: "flag values not counted", args: []string{"get", "hr/a", "--status-selector", "ready=false", "-l", "app=x", "-n", "ns"},
			want: []string{"get", "helmreleases", "a", "--status-selector", "ready=false", "-l", "app=x", "-n", "ns"}},
		{name: "timeout value not counted", args: []string{"reconcile", "gitrepo/a", "--timeout", "1m"},
			want: []string{"reconcile", "source", "git", "a", "--timeout", "1m"}},
		{name: "qualified type with space names unchanged count", args: []string{"get", "helmreleases.helm.toolkit.fluxcd.io", "a", "b"},
			want: []string{"get", "helmreleases", "a", "b"}},
	}
	tests = append(tests, kindCases()...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := slices.Clone(tt.args)
			got, err := normalizeResourceArgs(rootCmd, tt.args)
			if !slices.Equal(tt.args, in) {
				t.Errorf("input args were modified: %q", tt.args)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("got %q, want error %q", got, tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("got error %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResourceArgsEquivalentToSpaceForm(t *testing.T) {
	tmpl := map[string]string{"ns": allocateNamespace("tkrae")}
	testEnv.CreateObjectFile("./testdata/get/helmrelease_objects.yaml", tmpl, t)
	ns := " -n=" + tmpl["ns"]

	want, err := executeCommand("get helmrelease podinfo" + ns)
	if err != nil {
		t.Fatal(err)
	}

	for _, args := range []string{
		"get hr/podinfo",
		"get HelmRelease/podinfo",
		"get helmreleases/podinfo",
		"get helmreleases.helm.toolkit.fluxcd.io/podinfo",
		"get helmreleases.v2.helm.toolkit.fluxcd.io/podinfo",
		"get helmreleases.v2.helm.toolkit.fluxcd.io podinfo",
		"get -n=" + tmpl["ns"] + " hr/podinfo",
	} {
		t.Run(args, func(t *testing.T) {
			got, err := executeCommand(args + ns)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("output differs from space form:\nwant:\n%s\ngot:\n%s", want, got)
			}
		})
	}

	t.Run("flags after the resource still apply", func(t *testing.T) {
		got, err := executeCommand("get hr/podinfo --show-source" + ns)
		if err != nil {
			t.Fatal(err)
		}
		wantSource, err := executeCommand("get helmrelease podinfo --show-source" + ns)
		if err != nil {
			t.Fatal(err)
		}
		if got != wantSource {
			t.Errorf("output differs from space form:\nwant:\n%s\ngot:\n%s", wantSource, got)
		}
		if !strings.Contains(got, "SOURCE") {
			t.Errorf("--show-source was not applied:\n%s", got)
		}
	})
}

func TestResourceArgsEquivalentToSpaceFormAllKinds(t *testing.T) {
	tmpl := map[string]string{"ns": allocateNamespace("tkraek")}
	testEnv.CreateObjectFile("./testdata/resource_arg/objects.yaml", tmpl, t)
	ns := " -n=" + tmpl["ns"]

	for _, tc := range []struct {
		spaceForm string
		object    string
		forms     []string
	}{
		{"get sources git podinfo", "podinfo", []string{
			"get gitrepo/podinfo",
			"get GitRepository/podinfo",
			"get gitrepositories.source.toolkit.fluxcd.io/podinfo",
			"get gitrepositories.v1.source.toolkit.fluxcd.io podinfo",
		}},
		{"get sources oci podinfo", "podinfo", []string{
			"get ocirepo/podinfo",
			"get ocirepositories.source.toolkit.fluxcd.io/podinfo",
			"get OCIRepository.v1.source.toolkit.fluxcd.io/podinfo",
		}},
		{"get sources helm podinfo", "podinfo", []string{
			"get helmrepo/podinfo",
			"get helmrepositories/podinfo",
			"get helmrepositories.v1.source.toolkit.fluxcd.io/podinfo",
		}},
		{"get kustomizations podinfo", "podinfo", []string{
			"get ks/podinfo",
			"get Kustomization/podinfo",
			"get kustomizations.kustomize.toolkit.fluxcd.io/podinfo",
			"get kustomizations.v1.kustomize.toolkit.fluxcd.io podinfo",
		}},
		{"get images repository podinfo", "podinfo", []string{
			"get imgrepo/podinfo",
			"get imagerepo/podinfo",
			"get imagerepositories.image.toolkit.fluxcd.io/podinfo",
		}},
		{"get images policy podinfo", "podinfo", []string{
			"get imgpol/podinfo",
			"get imagepolicy/podinfo",
			"get imagepolicies.v1.image.toolkit.fluxcd.io/podinfo",
		}},
		{"get alert-providers slack", "slack", []string{
			"get provider/slack",
			"get alert-provider/slack",
			"get providers.notification.toolkit.fluxcd.io/slack",
		}},
		{"get alerts podinfo", "podinfo", []string{
			"get alert/podinfo",
			"get alerts.v1beta3.notification.toolkit.fluxcd.io/podinfo",
		}},
		// listing without a name
		{"get sources git", "podinfo", []string{
			"get gitrepositories.source.toolkit.fluxcd.io",
		}},
		{"get images policy", "podinfo", []string{
			"get imagepolicies.v1.image.toolkit.fluxcd.io",
		}},
		// flags between the verb and the resource and after it
		{"get sources git podinfo --status-selector ready=true", "podinfo", []string{
			"get --status-selector ready=true gitrepo/podinfo",
			"get gitrepo/podinfo --status-selector ready=true",
		}},
	} {
		t.Run(tc.spaceForm, func(t *testing.T) {
			want, err := executeCommand(tc.spaceForm + ns)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(want, tc.object) {
				t.Fatalf("space form did not return %q:\n%s", tc.object, want)
			}
			for _, args := range tc.forms {
				t.Run(args, func(t *testing.T) {
					got, err := executeCommand(args + ns)
					if err != nil {
						t.Fatal(err)
					}
					if got != want {
						t.Errorf("output differs from %q:\nwant:\n%s\ngot:\n%s", tc.spaceForm, want, got)
					}
				})
			}
		})
	}
}

func TestResourceArgsSameErrorsAsSpaceForm(t *testing.T) {
	tmpl := map[string]string{"ns": allocateNamespace("tksae")}
	testEnv.CreateObjectFile("./testdata/get/helmrelease_objects.yaml", tmpl, t)
	ns := " -n=" + tmpl["ns"]

	for _, tc := range []struct{ kubectlStyle, spaceForm string }{
		{"get hr/missing", "get helmrelease missing"},
		{"reconcile hr/missing", "reconcile helmrelease missing"},
		{"reconcile helmreleases.helm.toolkit.fluxcd.io/missing", "reconcile helmrelease missing"},
		{"reconcile gitrepo/missing", "reconcile source git missing"},
		{"reconcile ocirepositories.v1.source.toolkit.fluxcd.io/missing", "reconcile source oci missing"},
		{"reconcile ks/missing", "reconcile kustomization missing"},
		{"reconcile imgrepo/missing", "reconcile image repository missing"},
		{"reconcile imagepolicies.image.toolkit.fluxcd.io/missing", "reconcile image policy missing"},
		{"reconcile iua/missing", "reconcile image update missing"},
		{"reconcile receiver/missing", "reconcile receiver missing"},
	} {
		t.Run(tc.kubectlStyle, func(t *testing.T) {
			_, wantErr := executeCommand(tc.spaceForm + ns)
			if wantErr == nil {
				t.Fatal("expected an error for a missing object")
			}
			_, gotErr := executeCommand(tc.kubectlStyle + ns)
			if gotErr == nil || gotErr.Error() != wantErr.Error() {
				t.Errorf("got error %v, want %v", gotErr, wantErr)
			}
		})
	}
}

// TestResourceArgsErrors checks that normalization errors reach the
// command output.
func TestResourceArgsErrors(t *testing.T) {
	cmd := cmdTestCase{
		args:   "reconcile alert/x",
		assert: assertError("flux reconcile does not support Alert resources"),
	}
	cmd.runTestCmd(t)
}

// TestResourceArgTableCommands checks that every get and reconcile
// path in the resource type table resolves to a runnable command.
func TestResourceArgTableCommands(t *testing.T) {
	for _, rt := range resourceTypes {
		for _, verb := range []*cobra.Command{getCmd, reconcileCmd} {
			path := rt.commandPath(verb)
			if path == nil {
				continue
			}
			t.Run(rt.gvk.Kind+"/"+verb.Name(), func(t *testing.T) {
				args := append([]string{verb.Name()}, path...)
				cmd, rest, err := rootCmd.Find(args)
				if err != nil {
					t.Fatalf("Find(%q) error: %v", args, err)
				}
				want := rootCmd.Name() + " " + strings.Join(args, " ")
				if got := cmd.CommandPath(); got != want || len(rest) != 0 {
					t.Fatalf("Find(%q) = %q with rest %q, want %q", args, got, rest, want)
				}
				if !cmd.Runnable() {
					t.Errorf("%q is not runnable", want)
				}
			})
		}
	}
}

// TestResourceArgTableNames checks that the table kinds are registered
// in the scheme and that no type name is claimed by two kinds.
func TestResourceArgTableNames(t *testing.T) {
	scheme := utils.NewScheme()
	seen := map[string]string{}
	for _, rt := range resourceTypes {
		if !scheme.Recognizes(rt.gvk) {
			t.Errorf("%s is not registered in the scheme", rt.gvk)
		}
		if rt.get == nil && rt.reconcile == nil {
			t.Errorf("%s has no get or reconcile command", rt.gvk.Kind)
		}
		names := append([]string{strings.ToLower(rt.gvk.Kind), rt.plural}, rt.shortNames...)
		for _, n := range names {
			if other, ok := seen[n]; ok && other != rt.gvk.Kind {
				t.Errorf("type name %q used by %s and %s", n, other, rt.gvk.Kind)
			}
			seen[n] = rt.gvk.Kind
			if got := lookupResourceType(strings.ToUpper(n)); got == nil || got.gvk != rt.gvk {
				t.Errorf("lookupResourceType(%q) does not return %s", strings.ToUpper(n), rt.gvk.Kind)
			}
		}
	}
}

// TestResourceArgTableMatchesCRDs checks the table against the CRDs
// bundled in cmd/flux/manifests.
func TestResourceArgTableMatchesCRDs(t *testing.T) {
	files, err := filepath.Glob("manifests/*.yaml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no manifests found: %v", err)
	}
	crds := map[schema.GroupKind]apiextensionsv1.CustomResourceDefinition{}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		objs, err := ssautil.ReadObjects(f)
		f.Close()
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for _, obj := range objs {
			if obj.GetKind() != "CustomResourceDefinition" {
				continue
			}
			var crd apiextensionsv1.CustomResourceDefinition
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &crd); err != nil {
				t.Fatalf("%s: %v", obj.GetName(), err)
			}
			if strings.HasSuffix(crd.Spec.Group, ".fluxcd.io") {
				crds[schema.GroupKind{Group: crd.Spec.Group, Kind: crd.Spec.Names.Kind}] = crd
			}
		}
	}

	scheme := utils.NewScheme()
	seen := map[schema.GroupKind]bool{}
	for _, rt := range resourceTypes {
		gk := rt.gvk.GroupKind()
		seen[gk] = true
		crd, ok := crds[gk]
		if !ok {
			t.Errorf("%s: no CRD in manifests", gk)
			continue
		}
		if crd.Spec.Names.Plural != rt.plural {
			t.Errorf("%s: plural %q, CRD has %q", gk, rt.plural, crd.Spec.Names.Plural)
		}
		for _, sn := range crd.Spec.Names.ShortNames {
			if !slices.Contains(rt.shortNames, sn) {
				t.Errorf("%s: CRD short name %q missing from table", gk, sn)
			}
		}
		for _, v := range crd.Spec.Versions {
			if v.Served && !scheme.Recognizes(gk.WithVersion(v.Name)) {
				t.Errorf("%s: served version %s not registered in the scheme", gk, v.Name)
			}
		}
	}
	for gk := range crds {
		if !seen[gk] {
			t.Errorf("%s: CRD has no resource type table entry", gk)
		}
	}
}
