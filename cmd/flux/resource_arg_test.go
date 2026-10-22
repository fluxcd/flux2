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
	"slices"
	"strings"
	"testing"
)

func TestParseResourceArg(t *testing.T) {
	tests := []struct {
		arg      string
		wantType string
		wantName string
		wantOK   bool
	}{
		{"hr/my-release", "hr", "my-release", true},
		{"helmreleases.helm.toolkit.fluxcd.io/my-release", "helmreleases.helm.toolkit.fluxcd.io", "my-release", true},
		{"helmreleases.v2.helm.toolkit.fluxcd.io/my-release", "helmreleases.v2.helm.toolkit.fluxcd.io", "my-release", true},
		{"my-release", "", "", false},
		{"", "", "", false},
		{"hr/", "", "", false},
		{"/my-release", "", "", false},
		{"hr/a/b", "", "", false},
		{"--context=a/b", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			typ, name, ok := parseResourceArg(tt.arg)
			if typ != tt.wantType || name != tt.wantName || ok != tt.wantOK {
				t.Errorf("parseResourceArg(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.arg, typ, name, ok, tt.wantType, tt.wantName, tt.wantOK)
			}
		})
	}
}

func TestSplitResourceType(t *testing.T) {
	tests := []struct {
		in         string
		wantType   string
		wantSuffix string
	}{
		{"hr", "hr", ""},
		{"helmreleases.helm.toolkit.fluxcd.io", "helmreleases", "helm.toolkit.fluxcd.io"},
		{"helmreleases.v2.helm.toolkit.fluxcd.io", "helmreleases", "v2.helm.toolkit.fluxcd.io"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			typ, suffix := splitResourceType(tt.in)
			if typ != tt.wantType || suffix != tt.wantSuffix {
				t.Errorf("splitResourceType(%q) = (%q, %q), want (%q, %q)",
					tt.in, typ, suffix, tt.wantType, tt.wantSuffix)
			}
		})
	}
}

func TestNormalizeResourceArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []string
		wantErr bool
	}{
		{"get type/name", []string{"get", "hr/my-release"}, []string{"get", "helmreleases", "my-release"}, false},
		{"get with global flags", []string{"get", "--context", "c", "-n", "ns", "hr/my-release"},
			[]string{"get", "--context", "c", "-n", "ns", "helmreleases", "my-release"}, false},
		{"get with flag before command", []string{"--context", "c", "get", "hr/x"},
			[]string{"--context", "c", "get", "helmreleases", "x"}, false},
		{"get with bool flag", []string{"get", "-A", "hr/x"}, []string{"get", "-A", "helmreleases", "x"}, false},
		{"get with trailing flag", []string{"get", "hr/x", "--show-source"},
			[]string{"get", "helmreleases", "x", "--show-source"}, false},
		{"get qualified type", []string{"get", "helmreleases.helm.toolkit.fluxcd.io/x"},
			[]string{"get", "helmreleases", "x"}, false},
		{"get qualified type with version and space", []string{"get", "helmreleases.v2.helm.toolkit.fluxcd.io", "x"},
			[]string{"get", "helmreleases", "x"}, false},
		{"get unknown group", []string{"get", "helmreleases.wrong.group/x"}, nil, true},
		{"reconcile type/name", []string{"reconcile", "hr/podinfo", "--with-source"},
			[]string{"reconcile", "helmrelease", "podinfo", "--with-source"}, false},
		{"reconcile qualified plural", []string{"reconcile", "helmreleases.helm.toolkit.fluxcd.io/podinfo"},
			[]string{"reconcile", "helmrelease", "podinfo"}, false},
		{"space form unchanged", []string{"get", "hr", "my-release"}, []string{"get", "hr", "my-release"}, false},
		{"unknown type unchanged", []string{"get", "foo/bar"}, []string{"get", "foo/bar"}, false},
		{"get all unchanged", []string{"get", "all/x"}, []string{"get", "all/x"}, false},
		{"no args unchanged", []string{"get"}, []string{"get"}, false},
		{"help unchanged", []string{"get", "--help"}, []string{"get", "--help"}, false},
		{"other command unchanged", []string{"suspend", "hr/x"}, []string{"suspend", "hr/x"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeResourceArgs(rootCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
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

func TestResourceArgsSameErrorsAsSpaceForm(t *testing.T) {
	tmpl := map[string]string{"ns": allocateNamespace("tksae")}
	testEnv.CreateObjectFile("./testdata/get/helmrelease_objects.yaml", tmpl, t)
	ns := " -n=" + tmpl["ns"]

	for _, tc := range []struct{ kubectlStyle, spaceForm string }{
		{"get hr/missing", "get helmrelease missing"},
		{"reconcile hr/missing", "reconcile helmrelease missing"},
		{"reconcile helmreleases.helm.toolkit.fluxcd.io/missing", "reconcile helmrelease missing"},
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

func TestResourceArgsUnknownGroup(t *testing.T) {
	cmd := cmdTestCase{
		args:   "get helmreleases.wrong.group/x",
		assert: assertError(`"helmreleases.wrong.group" is not a known Flux resource type`),
	}
	cmd.runTestCmd(t)
}
