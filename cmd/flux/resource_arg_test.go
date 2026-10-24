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

	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ssautil "github.com/fluxcd/pkg/ssa/utils"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

type normalizeCase struct {
	name    string
	args    []string
	want    []string
	wantErr string
}

// resourceVerbs are the verbs that accept TYPE/NAME arguments.
var resourceVerbs = []*cobra.Command{getCmd, reconcileCmd, suspendCmd, resumeCmd, exportCmd, deleteCmd}

func joinArgs(parts ...[]string) []string {
	var out []string
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// kindCases generates, for every table entry and verb, the
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
		for _, verb := range resourceVerbs {
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
		{name: "literal suspend gitrepo", args: []string{"suspend", "gitrepo/x"}, want: []string{"suspend", "source", "git", "x"}},
		{name: "literal resume imgpol", args: []string{"resume", "imgpol/x"}, want: []string{"resume", "image", "policy", "x"}},
		{name: "literal delete ks silent", args: []string{"delete", "ks/x", "--silent"},
			want: []string{"delete", "kustomization", "x", "--silent"}},
		{name: "literal delete ks silent shorthand", args: []string{"delete", "-s", "ks/x"},
			want: []string{"delete", "-s", "kustomization", "x"}},
		{name: "literal export ag", args: []string{"export", "ag/x"}, want: []string{"export", "artifact", "generator", "x"}},
		{name: "literal reconcile ag", args: []string{"reconcile", "ag/x"}, want: []string{"reconcile", "artifact", "generator", "x"}},
		{name: "literal suspend ag", args: []string{"suspend", "ag/a", "ag/b"}, want: []string{"suspend", "artifact", "generator", "a", "b"}},
		{name: "literal resume ag qualified", args: []string{"resume", "artifactgenerators.source.extensions.fluxcd.io/x"},
			want: []string{"resume", "artifact", "generator", "x"}},
		{name: "literal export ea", args: []string{"export", "ea/x"}, want: []string{"export", "source", "external", "x"}},
		{name: "literal suspend provider", args: []string{"suspend", "provider/x"}, want: []string{"suspend", "alert-provider", "x"}},
		{name: "literal resume alert qualified", args: []string{"resume", "alerts.notification.toolkit.fluxcd.io/x"},
			want: []string{"resume", "alert", "x"}},
		{name: "literal export iua qualified no name", args: []string{"export", "imageupdateautomations.image.toolkit.fluxcd.io", "--all"},
			want: []string{"export", "image", "update", "--all"}},

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
			want: []string{"get", "-l", "app=x", "sources", "git"}},
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
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},

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
		// plain TYPE that cobra does not resolve
		{name: "plain short name", args: []string{"get", "gitrepo", "x"}, want: []string{"get", "sources", "git", "x"}},
		{name: "reconcile plural", args: []string{"reconcile", "kustomizations", "x"},
			want: []string{"reconcile", "kustomization", "x"}},
		{name: "reconcile plural with flags", args: []string{"reconcile", "-n", "ns", "helmreleases", "x", "--with-source"},
			want: []string{"reconcile", "-n", "ns", "helmrelease", "x", "--with-source"}},
		{name: "suspend plural several names", args: []string{"suspend", "kustomizations", "a", "b"},
			want: []string{"suspend", "kustomization", "a", "b"}},
		{name: "delete kind", args: []string{"delete", "GitRepository", "x", "--silent"},
			want: []string{"delete", "source", "git", "x", "--silent"}},
		{name: "plain type without name", args: []string{"get", "ocirepositories"}, want: []string{"get", "sources", "oci"}},
		{name: "plain type unsupported verb", args: []string{"reconcile", "alerts", "x"},
			wantErr: "flux reconcile does not support Alert resources"},
		{name: "plain unknown word unchanged", args: []string{"get", "podinfo", "x"}, want: []string{"get", "podinfo", "x"}},
		{name: "resolved alias unchanged", args: []string{"reconcile", "ks", "x"}, want: []string{"reconcile", "ks", "x"}},
		{name: "nested space form unchanged", args: []string{"get", "sources", "git", "x"}, want: []string{"get", "sources", "git", "x"}},
		{name: "reconcile space form unchanged", args: []string{"reconcile", "source", "oci", "x"},
			want: []string{"reconcile", "source", "oci", "x"}},
		{name: "unknown type unchanged", args: []string{"get", "foo/bar"}, want: []string{"get", "foo/bar"}},
		{name: "unknown qualified type unchanged", args: []string{"get", "foos.example.com/bar"}, want: []string{"get", "foos.example.com/bar"}},
		{name: "get all unchanged", args: []string{"get", "all/x"}, want: []string{"get", "all/x"}},
		{name: "no args unchanged", args: []string{"get"}, want: []string{"get"}},
		{name: "help unchanged", args: []string{"get", "--help"}, want: []string{"get", "--help"}},
		{name: "reconcile help unchanged", args: []string{"reconcile", "-h"}, want: []string{"reconcile", "-h"}},
		{name: "other command unchanged", args: []string{"create", "hr/x"}, want: []string{"create", "hr/x"}},
		{name: "other command with qualified type unchanged", args: []string{"tree", "kustomizations.kustomize.toolkit.fluxcd.io/x"},
			want: []string{"tree", "kustomizations.kustomize.toolkit.fluxcd.io/x"}},
		{name: "sub-command unchanged", args: []string{"suspend", "source", "gitrepo/x"},
			want: []string{"suspend", "source", "gitrepo/x"}},
		{name: "suspend space form unchanged", args: []string{"suspend", "hr", "a", "b"}, want: []string{"suspend", "hr", "a", "b"}},
		{name: "delete nested space form unchanged", args: []string{"delete", "source", "git", "x", "-s"},
			want: []string{"delete", "source", "git", "x", "-s"}},
		{name: "suspend all unchanged", args: []string{"suspend", "ks", "--all"}, want: []string{"suspend", "ks", "--all"}},
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
		{name: "version of another kind in group", args: []string{"get", "receivers.v1beta3.notification.toolkit.fluxcd.io/x"},
			wantErr: `"receivers.v1beta3.notification.toolkit.fluxcd.io" is not a known Flux resource type`},

		// unsupported verb
		{name: "reconcile alert", args: []string{"reconcile", "alert/x"},
			wantErr: "flux reconcile does not support Alert resources"},
		{name: "reconcile external artifact", args: []string{"reconcile", "ea/x"},
			wantErr: "flux reconcile does not support ExternalArtifact resources"},
		{name: "reconcile provider", args: []string{"reconcile", "alert-provider/x"},
			wantErr: "flux reconcile does not support Provider resources"},
		{name: "suspend external artifact", args: []string{"suspend", "ea/x"},
			wantErr: "flux suspend does not support ExternalArtifact resources"},
		{name: "delete artifact generator", args: []string{"delete", "ag/x"},
			wantErr: "flux delete does not support ArtifactGenerator resources"},

		// multiple resources
		{name: "two type/name args", args: []string{"get", "hr/a", "hr/b"},
			want: []string{"get", "helmreleases", "a", "b"}},
		{name: "type/name then name", args: []string{"get", "hr/a", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
		{name: "reconcile mixed kinds", args: []string{"reconcile", "gitrepo/a", "ks/b"},
			wantErr: `all TYPE/NAME arguments must be the same type, got "ks/b"`},
		{name: "extra arg after value flag", args: []string{"get", "hr/a", "--status-selector", "ready=false", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
		{name: "extra arg after bool flag", args: []string{"reconcile", "hr/a", "--with-source", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
		{name: "extra arg after sub-command bool flag", args: []string{"get", "hr/a", "--show-source", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
		{name: "extra arg after verb bool shorthand", args: []string{"get", "hr/a", "-A", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
		{name: "extra arg after local value flag", args: []string{"get", "hr/a", "-l", "app=x", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
		{name: "bool and value shorthand group after resource", args: []string{"get", "hr/a", "-Al", "!app"},
			want: []string{"get", "helmreleases", "a", "-Al", "!app"}},
		{name: "bool and value shorthand group with namespace", args: []string{"get", "hr/a", "-An", "ns"},
			want: []string{"get", "helmreleases", "a", "-An", "ns"}},
		{name: "extra arg after shorthand group", args: []string{"get", "hr/a", "-An", "ns", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
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

		// several resources on get, reconcile, export and delete
		{name: "export two type/name args", args: []string{"export", "hr/a", "hr/b"},
			want: []string{"export", "helmrelease", "a", "b"}},
		{name: "reconcile two type/name args", args: []string{"reconcile", "hr/a", "--with-source", "helmreleases.helm.toolkit.fluxcd.io/b"},
			want: []string{"reconcile", "helmrelease", "a", "--with-source", "b"}},
		{name: "export type/name then name", args: []string{"export", "gitrepo/a", "--all", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
		{name: "delete two type/name args", args: []string{"delete", "ks/a", "-s", "ks/b"},
			want: []string{"delete", "kustomization", "a", "-s", "b"}},
		{name: "delete name after double dash", args: []string{"delete", "ks/a", "--", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},

		// multi-name verbs
		{name: "suspend same kind", args: []string{"suspend", "hr/a", "hr/b", "hr/c"},
			want: []string{"suspend", "helmrelease", "a", "b", "c"}},
		{name: "suspend same kind different spellings", args: []string{"suspend", "hr/a", "HelmRelease/b", "helmreleases/c",
			"helmreleases.helm.toolkit.fluxcd.io/d", "HelmRelease.V2.helm.toolkit.fluxcd.io/e"},
			want: []string{"suspend", "helmrelease", "a", "b", "c", "d", "e"}},
		{name: "suspend nested kind", args: []string{"suspend", "gitrepo/a", "gitrepositories.source.toolkit.fluxcd.io/b"},
			want: []string{"suspend", "source", "git", "a", "b"}},
		{name: "resume nested kind", args: []string{"resume", "imgpol/a", "imagepolicy/b"},
			want: []string{"resume", "image", "policy", "a", "b"}},
		{name: "suspend flags interleaved", args: []string{"suspend", "hr/a", "--timeout", "1m", "helmreleases.helm.toolkit.fluxcd.io/b"},
			want: []string{"suspend", "helmrelease", "a", "--timeout", "1m", "b"}},
		{name: "resume wait flag interleaved", args: []string{"resume", "hr/a", "--wait=false", "helmreleases.helm.toolkit.fluxcd.io/b"},
			want: []string{"resume", "helmrelease", "a", "--wait=false", "b"}},
		{name: "resume bool flag between", args: []string{"resume", "ks/a", "--wait", "ks/b"},
			want: []string{"resume", "kustomization", "a", "--wait", "b"}},
		{name: "resume value flags between", args: []string{"resume", "-n", "ns", "ks/a", "--timeout", "ks/x", "ks/b", "--context", "c", "ks/c"},
			want: []string{"resume", "-n", "ns", "kustomization", "a", "--timeout", "ks/x", "b", "--context", "c", "c"}},
		{name: "delete shorthand group", args: []string{"delete", "hr/a", "-sn", "ns"},
			want: []string{"delete", "helmrelease", "a", "-sn", "ns"}},
		{name: "delete extra arg after shorthand group", args: []string{"delete", "hr/a", "-sn", "ns", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
		{name: "suspend flag as later name", args: []string{"suspend", "gitrepo/a", "gitrepo/--all"},
			wantErr: `invalid resource name "--all" in "gitrepo/--all"`},
		{name: "suspend namespace flag as later name", args: []string{"suspend", "-n", "staging", "ks/a", "ks/--namespace=production"},
			wantErr: `invalid resource name "--namespace=production" in "ks/--namespace=production"`},
		{name: "suspend shorthand with equals", args: []string{"suspend", "hr/a", "-n=ns", "hr/b"},
			want: []string{"suspend", "helmrelease", "a", "-n=ns", "b"}},
		{name: "suspend help between", args: []string{"suspend", "hr/a", "-h", "-n", "ns", "hr/b"},
			want: []string{"suspend", "helmrelease", "a", "-h", "-n", "ns", "b"}},
		{name: "suspend duplicate name", args: []string{"suspend", "hr/a", "hr/a"},
			want: []string{"suspend", "helmrelease", "a", "a"}},
		{name: "suspend all with type/name", args: []string{"suspend", "ks/a", "--all"},
			want: []string{"suspend", "kustomization", "a", "--all"}},
		{name: "suspend double dash", args: []string{"suspend", "hr/a", "--", "hr/b", "hr/c"},
			want: []string{"suspend", "helmrelease", "a", "--", "b", "c"}},
		{name: "suspend flag value double dash", args: []string{"suspend", "hr/a", "-n", "--", "hr/b"},
			want: []string{"suspend", "helmrelease", "a", "-n", "--", "b"}},
		{name: "suspend flag after double dash is a name", args: []string{"suspend", "hr/a", "--", "-n"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "-n"`},
		{name: "suspend different kind", args: []string{"suspend", "hr/a", "ks/b"},
			wantErr: `all TYPE/NAME arguments must be the same type, got "ks/b"`},
		{name: "resume different nested kind", args: []string{"resume", "gitrepo/a", "--wait=false", "ocirepo/b"},
			wantErr: `all TYPE/NAME arguments must be the same type, got "ocirepo/b"`},
		{name: "suspend unknown kind", args: []string{"suspend", "hr/a", "foo/b"},
			wantErr: `all TYPE/NAME arguments must be the same type, got "foo/b"`},
		{name: "suspend unsupported kind", args: []string{"suspend", "hr/a", "ea/b"},
			wantErr: `all TYPE/NAME arguments must be the same type, got "ea/b"`},
		{name: "suspend wrong group", args: []string{"suspend", "hr/a", "helmreleases.wrong.group/b"},
			wantErr: `"helmreleases.wrong.group" is not a known Flux resource type`},
		{name: "suspend wrong version", args: []string{"resume", "hr/a", "hr.v9.helm.toolkit.fluxcd.io/b"},
			wantErr: `"hr.v9.helm.toolkit.fluxcd.io" is not a known Flux resource type`},
		{name: "suspend plain name after", args: []string{"suspend", "hr/a", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
		{name: "resume plain name after flags", args: []string{"resume", "hr/a", "hr/b", "--wait=false", "-n", "ns", "c"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "c"`},
		{name: "suspend name after double dash", args: []string{"suspend", "hr/a", "--", "b"},
			wantErr: `cannot mix TYPE/NAME and NAME arguments, got "b"`},
		{name: "suspend qualified type with names unchanged", args: []string{"suspend", "helmreleases.helm.toolkit.fluxcd.io", "a", "b"},
			want: []string{"suspend", "helmrelease", "a", "b"}},
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
			"get alerts.v1.notification.toolkit.fluxcd.io/podinfo",
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
		{"suspend gitrepo/missing", "suspend source git missing"},
		{"suspend hr/missing hr/other", "suspend helmrelease missing other"},
		{"resume ks/missing --wait=false", "resume kustomization missing --wait=false"},
		{"resume imagepolicies.image.toolkit.fluxcd.io/missing", "resume image policy missing"},
		{"export ocirepo/missing", "export source oci missing"},
		{"export ag/missing", "export artifact generator missing"},
		{"delete hr/missing --silent", "delete helmrelease missing --silent"},
		{"delete gitrepositories.v1.source.toolkit.fluxcd.io/missing -s", "delete source git missing -s"},
		{"reconcile hr/missing", "reconcile helmrelease missing"},
		{"reconcile helmreleases.helm.toolkit.fluxcd.io/missing", "reconcile helmrelease missing"},
		{"reconcile gitrepo/missing", "reconcile source git missing"},
		{"reconcile ocirepositories.v1.source.toolkit.fluxcd.io/missing", "reconcile source oci missing"},
		{"reconcile ks/missing", "reconcile kustomization missing"},
		{"reconcile imgrepo/missing", "reconcile image repository missing"},
		{"reconcile imagepolicies.image.toolkit.fluxcd.io/missing", "reconcile image policy missing"},
		{"reconcile iua/missing", "reconcile image update missing"},
		{"reconcile receiver/missing", "reconcile receiver missing"},
		{"reconcile kustomizations missing", "reconcile kustomization missing"},
		{"reconcile helmreleases missing", "reconcile helmrelease missing"},
		{"reconcile gitrepo missing", "reconcile source git missing"},
		{"suspend kustomizations missing other", "suspend kustomization missing other"},
		{"get gitrepositories missing", "get sources git missing"},
	} {
		t.Run(tc.kubectlStyle, func(t *testing.T) {
			// suspend and resume report missing objects without failing.
			want, wantErr := executeCommand(tc.spaceForm + ns)
			if wantErr == nil && !strings.Contains(want, "not found") {
				t.Fatalf("expected a not found error or message, got %q", want)
			}
			got, gotErr := executeCommand(tc.kubectlStyle + ns)
			if fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
				t.Errorf("got error %v, want %v", gotErr, wantErr)
			}
			if got != want {
				t.Errorf("output differs from %q:\nwant:\n%s\ngot:\n%s", tc.spaceForm, want, got)
			}
		})
	}
}

func TestResourceArgsExportEquivalentToSpaceForm(t *testing.T) {
	tmpl := map[string]string{"ns": allocateNamespace("tkraexp")}
	testEnv.CreateObjectFile("./testdata/resource_arg/objects.yaml", tmpl, t)
	ns := " -n=" + tmpl["ns"]

	for _, tc := range []struct {
		spaceForm string
		forms     []string
	}{
		{"export source git podinfo", []string{
			"export gitrepo/podinfo",
			"export GitRepository/podinfo",
			"export gitrepositories.v1.source.toolkit.fluxcd.io/podinfo",
			"export gitrepositories.source.toolkit.fluxcd.io podinfo",
		}},
		{"export kustomization podinfo", []string{
			"export ks/podinfo",
			"export kustomizations.kustomize.toolkit.fluxcd.io/podinfo",
		}},
		{"export image policy podinfo", []string{
			"export imgpol/podinfo",
			"export imagepolicies.v1.image.toolkit.fluxcd.io/podinfo",
		}},
		{"export alert-provider slack", []string{
			"export provider/slack",
			"export providers.notification.toolkit.fluxcd.io/slack",
		}},
		{"export source git --all", []string{
			"export gitrepositories.source.toolkit.fluxcd.io --all",
		}},
	} {
		t.Run(tc.spaceForm, func(t *testing.T) {
			want, err := executeCommand(tc.spaceForm + ns)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(want, "kind:") {
				t.Fatalf("space form did not export an object:\n%s", want)
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

func TestResourceArgsMultipleNamesEquivalentToSpaceForm(t *testing.T) {
	tmpl := map[string]string{"ns": allocateNamespace("tkramn")}
	testEnv.CreateObjectFile("./testdata/resource_arg/objects.yaml", tmpl, t)
	ns := " -n=" + tmpl["ns"]

	for _, tc := range []struct {
		spaceForm string
		forms     []string
	}{
		{"get sources git podinfo flux-system", []string{
			"get gitrepo/podinfo gitrepo/flux-system",
			"get gitrepo/podinfo --no-header=false gitrepositories.source.toolkit.fluxcd.io/flux-system",
		}},
		{"export source git podinfo flux-system", []string{
			"export gitrepo/podinfo gitrepo/flux-system",
			"export GitRepository/podinfo gitrepositories.v1.source.toolkit.fluxcd.io/flux-system",
		}},
	} {
		t.Run(tc.spaceForm, func(t *testing.T) {
			want, err := executeCommand(tc.spaceForm + ns)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(want, "podinfo") || !strings.Contains(want, "flux-system") {
				t.Fatalf("space form did not return both objects:\n%s", want)
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

	t.Run("delete", func(t *testing.T) {
		if _, err := executeCommand("delete gitrepo/podinfo gitrepo/flux-system --silent" + ns); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"podinfo", "flux-system"} {
			var obj sourcev1.GitRepository
			err := testEnv.client.Get(context.Background(), client.ObjectKey{Namespace: tmpl["ns"], Name: name}, &obj)
			if !apierrors.IsNotFound(err) {
				t.Errorf("GitRepository %s: got %v, want NotFound", name, err)
			}
		}
	})
}

func TestResourceArgsSuspendResume(t *testing.T) {
	tmpl := map[string]string{"ns": allocateNamespace("tkrasr")}
	testEnv.CreateObjectFile("./testdata/resource_arg/objects.yaml", tmpl, t)
	ns := " -n=" + tmpl["ns"]

	suspended := func(t *testing.T, name string) bool {
		t.Helper()
		var obj sourcev1.GitRepository
		key := client.ObjectKey{Namespace: tmpl["ns"], Name: name}
		if err := testEnv.client.Get(context.Background(), key, &obj); err != nil {
			t.Fatal(err)
		}
		return obj.Spec.Suspend
	}
	run := func(t *testing.T, args string) {
		t.Helper()
		if _, err := executeCommand(args + ns); err != nil {
			t.Fatalf("%s: %v", args, err)
		}
	}

	run(t, "suspend gitrepo/podinfo")
	if !suspended(t, "podinfo") || suspended(t, "flux-system") {
		t.Fatal("suspend gitrepo/podinfo did not suspend only podinfo")
	}
	run(t, "resume gitrepo/podinfo --wait=false")
	if suspended(t, "podinfo") {
		t.Fatal("resume gitrepo/podinfo did not resume podinfo")
	}

	run(t, "suspend gitrepo/podinfo gitrepositories.source.toolkit.fluxcd.io/flux-system")
	if !suspended(t, "podinfo") || !suspended(t, "flux-system") {
		t.Fatal("suspend with two TYPE/NAME arguments did not suspend both")
	}
	run(t, "resume GitRepository/podinfo --wait=false gitrepo/flux-system")
	if suspended(t, "podinfo") || suspended(t, "flux-system") {
		t.Fatal("resume with two TYPE/NAME arguments did not resume both")
	}
}

// Resume waits for readiness only when given exactly one name and no
// --wait flag; the rewrite must keep the name count.
func TestResourceArgsResumeWait(t *testing.T) {
	tmpl := map[string]string{"ns": allocateNamespace("tkrarw")}
	testEnv.CreateObjectFile("./testdata/resource_arg/objects.yaml", tmpl, t)
	ns := " -n=" + tmpl["ns"] + " --timeout=2s"

	resume := func(t *testing.T, args string) (string, error) {
		t.Helper()
		if _, err := executeCommand("suspend source git podinfo flux-system" + ns); err != nil {
			t.Fatal(err)
		}
		return executeCommand(args + ns)
	}

	for _, tc := range []struct {
		args, spaceForm string
		wait            bool
	}{
		{"resume gitrepo/podinfo", "resume source git podinfo", true},
		{"resume gitrepo/podinfo gitrepo/flux-system", "resume source git podinfo flux-system", false},
		{"resume gitrepo/podinfo gitrepo/flux-system --wait", "resume source git podinfo flux-system --wait", true},
		{"resume gitrepo/podinfo --wait=false", "resume source git podinfo --wait=false", false},
	} {
		t.Run(tc.args, func(t *testing.T) {
			want, wantErr := resume(t, tc.spaceForm)
			got, gotErr := resume(t, tc.args)
			if fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
				t.Errorf("got error %v, want %v", gotErr, wantErr)
			}
			if waited := strings.Contains(got, "waiting for"); waited != tc.wait {
				t.Errorf("waited = %v, want %v:\n%s", waited, tc.wait, got)
			}
			if strings.Contains(want, "waiting for") != tc.wait {
				t.Errorf("space form waited = %v, want %v:\n%s", !tc.wait, tc.wait, want)
			}
		})
	}
}

func TestResourceArgsDelete(t *testing.T) {
	tmpl := map[string]string{"ns": allocateNamespace("tkradel")}
	testEnv.CreateObjectFile("./testdata/resource_arg/objects.yaml", tmpl, t)
	ns := " -n=" + tmpl["ns"]

	for _, tc := range []struct{ del, get, getSpaceForm string }{
		{"delete gitrepo/podinfo --silent", "get gitrepo/podinfo", "get sources git podinfo"},
		{"delete -s imgpol/podinfo", "get imgpol/podinfo", "get images policy podinfo"},
	} {
		t.Run(tc.del, func(t *testing.T) {
			if _, err := executeCommand(tc.del + ns); err != nil {
				t.Fatal(err)
			}
			_, wantErr := executeCommand(tc.getSpaceForm + ns)
			if wantErr == nil {
				t.Fatalf("%q found the deleted object", tc.getSpaceForm)
			}
			_, gotErr := executeCommand(tc.get + ns)
			if gotErr == nil || gotErr.Error() != wantErr.Error() {
				t.Errorf("got error %v, want %v", gotErr, wantErr)
			}
		})
	}

	var obj sourcev1.GitRepository
	err := testEnv.client.Get(context.Background(), client.ObjectKey{Namespace: tmpl["ns"], Name: "podinfo"}, &obj)
	if !apierrors.IsNotFound(err) {
		t.Errorf("got %v, want a not found error", err)
	}
	if err := testEnv.client.Get(context.Background(), client.ObjectKey{Namespace: tmpl["ns"], Name: "flux-system"}, &obj); err != nil {
		t.Errorf("flux-system was deleted: %v", err)
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

// TestResourceArgTableCommands checks that every path in the resource
// type table resolves to a runnable command, and that every runnable
// command under a verb, except the "all" commands, is in the table.
func TestResourceArgTableCommands(t *testing.T) {
	covered := map[string]bool{}
	for _, rt := range resourceTypes {
		for verb, path := range rt.paths {
			if !slices.ContainsFunc(resourceVerbs, func(c *cobra.Command) bool { return c.Name() == verb }) {
				t.Errorf("%s: unknown verb %q", rt.gvk.Kind, verb)
				continue
			}
			args := append([]string{verb}, path...)
			covered[strings.Join(args, " ")] = true
			t.Run(rt.gvk.Kind+"/"+verb, func(t *testing.T) {
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

	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, c := range cmd.Commands() {
			walk(c)
		}
		path := strings.TrimPrefix(cmd.CommandPath(), rootCmd.Name()+" ")
		if cmd.Runnable() && cmd.Name() != "all" && !covered[path] {
			t.Errorf("%q has no resource type table entry", path)
		}
	}
	for _, verb := range resourceVerbs {
		walk(verb)
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
		if len(rt.paths) == 0 {
			t.Errorf("%s has no command paths", rt.gvk.Kind)
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
