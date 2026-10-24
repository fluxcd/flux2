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
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	swapi "github.com/fluxcd/source-watcher/api/v2/v1beta1"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

// resourceType maps a Flux kind to its kubectl-style names and, per
// verb, to the sub-command path that handles it. A verb missing from
// paths does not handle the kind.
type resourceType struct {
	gvk        schema.GroupVersionKind
	plural     string
	shortNames []string
	paths      map[string][]string
}

// resourceTypes lists the Flux kinds accepted in TYPE/NAME arguments.
var resourceTypes = []resourceType{
	{
		gvk:        helmv2.GroupVersion.WithKind(helmv2.HelmReleaseKind),
		plural:     "helmreleases",
		shortNames: []string{"hr"},
		paths: map[string][]string{
			"get":       {"helmreleases"},
			"reconcile": {"helmrelease"},
			"suspend":   {"helmrelease"},
			"resume":    {"helmrelease"},
			"export":    {"helmrelease"},
			"delete":    {"helmrelease"},
		},
	},
	{
		gvk:        kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind),
		plural:     "kustomizations",
		shortNames: []string{"ks"},
		paths: map[string][]string{
			"get":       {"kustomizations"},
			"reconcile": {"kustomization"},
			"suspend":   {"kustomization"},
			"resume":    {"kustomization"},
			"export":    {"kustomization"},
			"delete":    {"kustomization"},
		},
	},
	{
		gvk:        sourcev1.GroupVersion.WithKind(sourcev1.GitRepositoryKind),
		plural:     "gitrepositories",
		shortNames: []string{"gitrepo"},
		paths: map[string][]string{
			"get":       {"sources", "git"},
			"reconcile": {"source", "git"},
			"suspend":   {"source", "git"},
			"resume":    {"source", "git"},
			"export":    {"source", "git"},
			"delete":    {"source", "git"},
		},
	},
	{
		gvk:        sourcev1.GroupVersion.WithKind(sourcev1.OCIRepositoryKind),
		plural:     "ocirepositories",
		shortNames: []string{"ocirepo"},
		paths: map[string][]string{
			"get":       {"sources", "oci"},
			"reconcile": {"source", "oci"},
			"suspend":   {"source", "oci"},
			"resume":    {"source", "oci"},
			"export":    {"source", "oci"},
			"delete":    {"source", "oci"},
		},
	},
	{
		gvk:        sourcev1.GroupVersion.WithKind(sourcev1.HelmRepositoryKind),
		plural:     "helmrepositories",
		shortNames: []string{"helmrepo"},
		paths: map[string][]string{
			"get":       {"sources", "helm"},
			"reconcile": {"source", "helm"},
			"suspend":   {"source", "helm"},
			"resume":    {"source", "helm"},
			"export":    {"source", "helm"},
			"delete":    {"source", "helm"},
		},
	},
	{
		gvk:        sourcev1.GroupVersion.WithKind(sourcev1.HelmChartKind),
		plural:     "helmcharts",
		shortNames: []string{"hc"},
		paths: map[string][]string{
			"get":       {"sources", "chart"},
			"reconcile": {"source", "chart"},
			"suspend":   {"source", "chart"},
			"resume":    {"source", "chart"},
			"export":    {"source", "chart"},
			"delete":    {"source", "chart"},
		},
	},
	{
		gvk:    sourcev1.GroupVersion.WithKind(sourcev1.BucketKind),
		plural: "buckets",
		paths: map[string][]string{
			"get":       {"sources", "bucket"},
			"reconcile": {"source", "bucket"},
			"suspend":   {"source", "bucket"},
			"resume":    {"source", "bucket"},
			"export":    {"source", "bucket"},
			"delete":    {"source", "bucket"},
		},
	},
	{
		gvk:        sourcev1.GroupVersion.WithKind(sourcev1.ExternalArtifactKind),
		plural:     "externalartifacts",
		shortNames: []string{"ea"},
		paths: map[string][]string{
			"get":    {"sources", "external"},
			"export": {"source", "external"},
		},
	},
	{
		gvk:        swapi.GroupVersion.WithKind(swapi.ArtifactGeneratorKind),
		plural:     "artifactgenerators",
		shortNames: []string{"ag"},
		paths: map[string][]string{
			"get":       {"artifacts", "generators"},
			"reconcile": {"artifact", "generator"},
			"suspend":   {"artifact", "generator"},
			"resume":    {"artifact", "generator"},
			"export":    {"artifact", "generator"},
		},
	},
	{
		gvk:        imagev1.GroupVersion.WithKind(imagev1.ImageRepositoryKind),
		plural:     "imagerepositories",
		shortNames: []string{"imgrepo", "imagerepo"},
		paths: map[string][]string{
			"get":       {"images", "repository"},
			"reconcile": {"image", "repository"},
			"suspend":   {"image", "repository"},
			"resume":    {"image", "repository"},
			"export":    {"image", "repository"},
			"delete":    {"image", "repository"},
		},
	},
	{
		gvk:        imagev1.GroupVersion.WithKind(imagev1.ImagePolicyKind),
		plural:     "imagepolicies",
		shortNames: []string{"imgpol", "imagepol"},
		paths: map[string][]string{
			"get":       {"images", "policy"},
			"reconcile": {"image", "policy"},
			"suspend":   {"image", "policy"},
			"resume":    {"image", "policy"},
			"export":    {"image", "policy"},
			"delete":    {"image", "policy"},
		},
	},
	{
		gvk:        autov1.GroupVersion.WithKind(autov1.ImageUpdateAutomationKind),
		plural:     "imageupdateautomations",
		shortNames: []string{"iua", "imgupd", "imgauto"},
		paths: map[string][]string{
			"get":       {"images", "update"},
			"reconcile": {"image", "update"},
			"suspend":   {"image", "update"},
			"resume":    {"image", "update"},
			"export":    {"image", "update"},
			"delete":    {"image", "update"},
		},
	},
	{
		gvk:    notificationv1.GroupVersion.WithKind(notificationv1.AlertKind),
		plural: "alerts",
		paths: map[string][]string{
			"get":     {"alerts"},
			"suspend": {"alert"},
			"resume":  {"alert"},
			"export":  {"alert"},
			"delete":  {"alert"},
		},
	},
	// The CLI names Provider resources alert-providers.
	{
		gvk:        notificationv1.GroupVersion.WithKind(notificationv1.ProviderKind),
		plural:     "providers",
		shortNames: []string{"alert-provider", "alert-providers"},
		paths: map[string][]string{
			"get":     {"alert-providers"},
			"suspend": {"alert-provider"},
			"resume":  {"alert-provider"},
			"export":  {"alert-provider"},
			"delete":  {"alert-provider"},
		},
	},
	{
		gvk:    notificationv1.GroupVersion.WithKind(notificationv1.ReceiverKind),
		plural: "receivers",
		paths: map[string][]string{
			"get":       {"receivers"},
			"reconcile": {"receiver"},
			"suspend":   {"receiver"},
			"resume":    {"receiver"},
			"export":    {"receiver"},
			"delete":    {"receiver"},
		},
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
	return rt.paths[verb.Name()]
}

// isResourceVerb reports whether cmd is a direct child of root that
// handles at least one kind in the resource type table.
func isResourceVerb(root, cmd *cobra.Command) bool {
	if cmd.Parent() != root {
		return false
	}
	for _, rt := range resourceTypes {
		if rt.commandPath(cmd) != nil {
			return true
		}
	}
	return false
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

// validateName rejects a NAME from arg that would turn into a flag
// after the rewrite, e.g. "gitrepo/--all".
func validateName(name, arg string) error {
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("invalid resource name %q in %q", name, arg)
	}
	return nil
}

// resolveResourceType returns the entry for a TYPE[.VERSION][.GROUP]
// type, or nil if TYPE is not in the table. It returns an error if the
// group or version do not match the entry.
func resolveResourceType(typ string) (*resourceType, error) {
	base, suffix, qualified := strings.Cut(typ, ".")
	rt := lookupResourceType(base)
	if rt == nil {
		return nil, nil
	}
	if qualified && !rt.matchesSuffix(suffix) {
		return nil, fmt.Errorf("%q is not a known Flux resource type", typ)
	}
	return rt, nil
}

// normalizeResourceArgs rewrites kubectl-style resource arguments of
// the verbs in the resource type table into the form the sub-commands
// accept, e.g. "get hr/podinfo" becomes "get helmreleases podinfo" and
// "suspend gitrepo/a gitrepo/b" becomes "suspend source git a b".
// A TYPE in the space form is mapped too, e.g. "reconcile
// kustomizations podinfo" becomes "reconcile kustomization podinfo".
// Arguments it does not recognize are returned unchanged.
func normalizeResourceArgs(root *cobra.Command, args []string) ([]string, error) {
	verb, _, err := root.Find(args)
	if err != nil || !isResourceVerb(root, verb) {
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
	// A plain TYPE reaches here only when cobra found no sub-command
	// for it, e.g. the plural in "reconcile kustomizations podinfo".
	if !hasName {
		typ = after[i]
	}

	rt, err := resolveResourceType(typ)
	if err != nil {
		return nil, err
	}
	if rt == nil {
		return args, nil
	}
	if hasName {
		if err := validateName(name, after[i]); err != nil {
			return nil, err
		}
	}

	path := rt.commandPath(verb)
	if path == nil {
		return nil, fmt.Errorf("%s does not support %s resources", verb.CommandPath(), rt.gvk.Kind)
	}

	rest := slices.Clone(after[i+1:])
	if hasName {
		sub, _, err := verb.Find(path)
		if err != nil {
			return nil, err
		}
		for _, j := range operandIndices(sub, rest) {
			arg := rest[j]
			t, n, ok := parseResourceArg(arg)
			if !ok {
				return nil, fmt.Errorf("cannot mix TYPE/NAME and NAME arguments, got %q", arg)
			}
			other, err := resolveResourceType(t)
			if err != nil {
				return nil, err
			}
			if other != rt {
				return nil, fmt.Errorf("all TYPE/NAME arguments must be the same type, got %q", arg)
			}
			if err := validateName(n, arg); err != nil {
				return nil, err
			}
			rest[j] = n
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

// operandIndices parses args the way pflag will when cmd runs and
// returns the indices of the positional arguments. It parses
// into a copy of the flag definitions, so the command's flag values
// are left untouched. Parse errors are ignored and left for cobra to
// report.
func operandIndices(cmd *cobra.Command, args []string) []int {
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

	// Stop at each operand to record its position, then continue after it.
	flags.SetInterspersed(false)
	var indices []int
	for pos := 0; pos < len(args); {
		if err := flags.Parse(args[pos:]); err != nil {
			return nil
		}
		rest := flags.Args()
		if len(rest) == 0 {
			break
		}
		start := len(args) - len(rest)
		// Everything after "--" is an operand.
		if flags.ArgsLenAtDash() == 0 {
			for j := start; j < len(args); j++ {
				indices = append(indices, j)
			}
			break
		}
		indices = append(indices, start)
		pos = start + 1
	}
	return indices
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
