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
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	autov1 "github.com/fluxcd/image-automation-controller/api/v1"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
	notificationv1b3 "github.com/fluxcd/notification-controller/api/v1beta3"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	swapi "github.com/fluxcd/source-watcher/api/v2/v1beta1"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

// resourceType maps a Flux kind to its kubectl-style names and to the
// flux get and reconcile sub-commands that handle it.
type resourceType struct {
	gvk        schema.GroupVersionKind
	plural     string
	shortNames []string
	get        []string
	reconcile  []string
}

// resourceTypes lists the Flux kinds accepted in TYPE/NAME arguments.
var resourceTypes = []resourceType{
	{
		gvk:        helmv2.GroupVersion.WithKind(helmv2.HelmReleaseKind),
		plural:     "helmreleases",
		shortNames: []string{"hr"},
		get:        []string{"helmreleases"},
		reconcile:  []string{"helmrelease"},
	},
	{
		gvk:        kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind),
		plural:     "kustomizations",
		shortNames: []string{"ks"},
		get:        []string{"kustomizations"},
		reconcile:  []string{"kustomization"},
	},
	{
		gvk:        sourcev1.GroupVersion.WithKind(sourcev1.GitRepositoryKind),
		plural:     "gitrepositories",
		shortNames: []string{"gitrepo"},
		get:        []string{"sources", "git"},
		reconcile:  []string{"source", "git"},
	},
	{
		gvk:        sourcev1.GroupVersion.WithKind(sourcev1.OCIRepositoryKind),
		plural:     "ocirepositories",
		shortNames: []string{"ocirepo"},
		get:        []string{"sources", "oci"},
		reconcile:  []string{"source", "oci"},
	},
	{
		gvk:        sourcev1.GroupVersion.WithKind(sourcev1.HelmRepositoryKind),
		plural:     "helmrepositories",
		shortNames: []string{"helmrepo"},
		get:        []string{"sources", "helm"},
		reconcile:  []string{"source", "helm"},
	},
	{
		gvk:        sourcev1.GroupVersion.WithKind(sourcev1.HelmChartKind),
		plural:     "helmcharts",
		shortNames: []string{"hc"},
		get:        []string{"sources", "chart"},
		reconcile:  []string{"source", "chart"},
	},
	{
		gvk:       sourcev1.GroupVersion.WithKind(sourcev1.BucketKind),
		plural:    "buckets",
		get:       []string{"sources", "bucket"},
		reconcile: []string{"source", "bucket"},
	},
	{
		gvk:        sourcev1.GroupVersion.WithKind(sourcev1.ExternalArtifactKind),
		plural:     "externalartifacts",
		shortNames: []string{"ea"},
		get:        []string{"sources", "external"},
	},
	{
		gvk:        swapi.GroupVersion.WithKind(swapi.ArtifactGeneratorKind),
		plural:     "artifactgenerators",
		shortNames: []string{"ag"},
		get:        []string{"artifacts", "generators"},
	},
	{
		gvk:        imagev1.GroupVersion.WithKind(imagev1.ImageRepositoryKind),
		plural:     "imagerepositories",
		shortNames: []string{"imgrepo", "imagerepo"},
		get:        []string{"images", "repository"},
		reconcile:  []string{"image", "repository"},
	},
	{
		gvk:        imagev1.GroupVersion.WithKind(imagev1.ImagePolicyKind),
		plural:     "imagepolicies",
		shortNames: []string{"imgpol", "imagepol"},
		get:        []string{"images", "policy"},
		reconcile:  []string{"image", "policy"},
	},
	{
		gvk:        autov1.GroupVersion.WithKind(autov1.ImageUpdateAutomationKind),
		plural:     "imageupdateautomations",
		shortNames: []string{"iua", "imgupd", "imgauto"},
		get:        []string{"images", "update"},
		reconcile:  []string{"image", "update"},
	},
	{
		gvk:    notificationv1b3.GroupVersion.WithKind(notificationv1b3.AlertKind),
		plural: "alerts",
		get:    []string{"alerts"},
	},
	// The CLI names Provider resources alert-providers.
	{
		gvk:        notificationv1b3.GroupVersion.WithKind(notificationv1b3.ProviderKind),
		plural:     "providers",
		shortNames: []string{"alert-provider", "alert-providers"},
		get:        []string{"alert-providers"},
	},
	{
		gvk:       notificationv1.GroupVersion.WithKind(notificationv1.ReceiverKind),
		plural:    "receivers",
		get:       []string{"receivers"},
		reconcile: []string{"receiver"},
	},
}

// lookupResourceType returns the entry whose kind, plural or short
// name matches typ, ignoring case, or nil if there is none.
func lookupResourceType(typ string) *resourceType {
	typ = strings.ToLower(typ)
	for i := range resourceTypes {
		rt := &resourceTypes[i]
		if typ == strings.ToLower(rt.gvk.Kind) || typ == rt.plural || slices.Contains(rt.shortNames, typ) {
			return rt
		}
	}
	return nil
}

// matchesSuffix reports whether suffix, either GROUP or VERSION.GROUP,
// refers to the entry's API group and a version registered for its kind.
func (rt *resourceType) matchesSuffix(suffix string) bool {
	suffix = strings.ToLower(suffix)
	if suffix == rt.gvk.Group {
		return true
	}
	version, group, ok := strings.Cut(suffix, ".")
	if !ok || group != rt.gvk.Group {
		return false
	}
	return utils.NewScheme().Recognizes(rt.gvk.GroupKind().WithVersion(version))
}

// commandPath returns the sub-command path for verb, or nil if the
// verb does not handle this kind.
func (rt *resourceType) commandPath(verb *cobra.Command) []string {
	if verb == getCmd {
		return rt.get
	}
	return rt.reconcile
}

// parseResourceArg splits a kubectl-style TYPE[.VERSION][.GROUP]/NAME
// argument into its type and name. It reports false if arg is not in
// that form.
func parseResourceArg(arg string) (typ, name string, ok bool) {
	typ, name, found := strings.Cut(arg, "/")
	if !found || typ == "" || name == "" || strings.Contains(name, "/") {
		return "", "", false
	}
	return typ, name, true
}

// normalizeResourceArgs rewrites kubectl-style resource arguments of
// the get and reconcile commands into the form the sub-commands accept,
// e.g. "get hr/podinfo" becomes "get helmreleases podinfo" and
// "reconcile gitrepo/podinfo" becomes "reconcile source git podinfo".
// Arguments it does not recognize are returned unchanged.
func normalizeResourceArgs(root *cobra.Command, args []string) ([]string, error) {
	verb, _, err := root.Find(args)
	if err != nil || (verb != getCmd && verb != reconcileCmd) {
		return args, nil
	}

	v := nextPositional(commandFlags(root), args)
	if v < 0 || args[v] == "--" {
		return args, nil
	}
	before, after := args[:v+1], args[v+1:]

	i := nextPositional(commandFlags(verb), after)
	if i < 0 || after[i] == "--" {
		return args, nil
	}

	typ, name, hasName := parseResourceArg(after[i])
	if !hasName {
		typ = after[i]
	}
	base, suffix, qualified := strings.Cut(typ, ".")
	if !hasName && !qualified {
		return args, nil
	}

	rt := lookupResourceType(base)
	if rt == nil {
		return args, nil
	}
	// A name must not turn into a flag after the rewrite.
	if hasName && strings.HasPrefix(name, "-") {
		return nil, fmt.Errorf("invalid resource name %q in %q", name, after[i])
	}
	if qualified && !rt.matchesSuffix(suffix) {
		return nil, fmt.Errorf("%q is not a known Flux resource type", typ)
	}

	path := rt.commandPath(verb)
	if path == nil {
		return nil, fmt.Errorf("%s does not support %s resources", verb.CommandPath(), rt.gvk.Kind)
	}

	rest := after[i+1:]
	if hasName {
		sub, _, err := verb.Find(path)
		if err != nil {
			return nil, err
		}
		if extra, ok := firstOperand(sub, rest); ok {
			return nil, fmt.Errorf("only one TYPE/NAME argument is supported, got %q", extra)
		}
	}

	out := make([]string, 0, len(args)+len(path))
	out = append(out, before...)
	out = append(out, after[:i]...)
	out = append(out, path...)
	if hasName {
		out = append(out, name)
	}
	out = append(out, rest...)
	return out, nil
}

// firstOperand parses args the way pflag will when cmd runs and returns
// the first positional argument, if any. It parses into a copy of the
// flag definitions, so the command's flag values are left untouched.
// Parse errors are ignored and left for cobra to report.
func firstOperand(cmd *cobra.Command, args []string) (string, bool) {
	flags := pflag.NewFlagSet(cmd.Name(), pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.ParseErrorsAllowlist.UnknownFlags = true
	commandFlags(cmd).VisitAll(func(f *pflag.Flag) {
		flags.AddFlag(&pflag.Flag{
			Name:        f.Name,
			Shorthand:   f.Shorthand,
			NoOptDefVal: f.NoOptDefVal,
			Value:       discardValue(f.Value.Type()),
		})
	})
	// cobra adds the help flag lazily, before the command runs.
	if flags.Lookup("help") == nil {
		shorthand := "h"
		if flags.ShorthandLookup(shorthand) != nil {
			shorthand = ""
		}
		flags.AddFlag(&pflag.Flag{Name: "help", Shorthand: shorthand, NoOptDefVal: "true", Value: discardValue("bool")})
	}
	if err := flags.Parse(args); err != nil || flags.NArg() == 0 {
		return "", false
	}
	return flags.Arg(0), true
}

// discardValue is a pflag.Value that accepts and drops any value.
type discardValue string

func (v discardValue) String() string   { return "" }
func (v discardValue) Set(string) error { return nil }
func (v discardValue) Type() string     { return string(v) }

// commandFlags returns all flags cmd accepts, its own and inherited.
func commandFlags(cmd *cobra.Command) *pflag.FlagSet {
	flags := pflag.NewFlagSet(cmd.Name(), pflag.ContinueOnError)
	flags.AddFlagSet(cmd.LocalFlags())
	flags.AddFlagSet(cmd.InheritedFlags())
	return flags
}

// nextPositional returns the index of the first argument in args that
// is neither a flag nor a flag value, or of the "--" terminator if that
// comes first. It returns -1 if there is neither. Flag values are
// skipped the same way cobra does when it looks up sub-commands.
func nextPositional(flags *pflag.FlagSet, args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return i
		case strings.HasPrefix(a, "--"):
			if !strings.Contains(a, "=") && !hasNoOptDefVal(flags.Lookup(a[2:])) {
				i++
			}
		case strings.HasPrefix(a, "-"):
			if len(a) == 2 && a[1] != '=' && !hasNoOptDefVal(flags.ShorthandLookup(a[1:])) {
				i++
			}
		case a != "":
			return i
		}
	}
	return -1
}

// hasNoOptDefVal reports whether f is a known flag that takes no value,
// such as a boolean flag.
func hasNoOptDefVal(f *pflag.Flag) bool {
	return f != nil && f.NoOptDefVal != ""
}
