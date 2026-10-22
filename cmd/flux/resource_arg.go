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
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

// parseResourceArg splits a kubectl-style TYPE[.VERSION][.GROUP]/NAME
// argument into its type and name. It reports false if arg is not in
// that form.
func parseResourceArg(arg string) (typ, name string, ok bool) {
	if strings.HasPrefix(arg, "-") {
		return "", "", false
	}
	typ, name, found := strings.Cut(arg, "/")
	if !found || typ == "" || name == "" || strings.Contains(name, "/") {
		return "", "", false
	}
	return typ, name, true
}

// splitResourceType splits TYPE[.VERSION][.GROUP] into the resource
// type and the optional API version and group suffix.
func splitResourceType(s string) (typ, suffix string) {
	typ, suffix, _ = strings.Cut(s, ".")
	return typ, suffix
}

// normalizeResourceArgs rewrites kubectl-style resource arguments of
// the get and reconcile commands into the form the sub-commands accept,
// e.g. "get hr/podinfo" becomes "get helmreleases podinfo". Arguments
// it does not recognize are returned unchanged.
func normalizeResourceArgs(root *cobra.Command, args []string) ([]string, error) {
	cmd, rest, err := root.Find(args)
	if err != nil || (cmd != getCmd && cmd != reconcileCmd) {
		return args, nil
	}

	// rest is args without the command name, find where it was removed.
	cut := 0
	for cut < len(rest) && args[cut] == rest[cut] {
		cut++
	}
	before, after := args[:cut+1], args[cut+1:]

	flags := pflag.NewFlagSet("", pflag.ContinueOnError)
	flags.AddFlagSet(root.PersistentFlags())
	flags.AddFlagSet(cmd.PersistentFlags())

	i := firstPositional(flags, after)
	if i < 0 {
		return args, nil
	}

	typ, name, hasName := parseResourceArg(after[i])
	if !hasName {
		typ = after[i]
	}
	base, suffix := splitResourceType(typ)
	if !hasName && suffix == "" {
		return args, nil
	}

	sub := findResourceCommand(cmd, base)
	if sub == nil {
		return args, nil
	}
	if suffix != "" && !resourceGroupMatches(base, suffix) {
		return nil, fmt.Errorf("%q is not a known Flux resource type", typ)
	}

	out := make([]string, 0, len(args)+1)
	out = append(out, before...)
	out = append(out, after[:i]...)
	out = append(out, sub.Name())
	if hasName {
		out = append(out, name)
	}
	out = append(out, after[i+1:]...)
	return out, nil
}

// firstPositional returns the index of the first argument that is not
// a flag or a flag value, or -1 if there is none.
func firstPositional(flags *pflag.FlagSet, args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return -1
		case strings.HasPrefix(a, "--"):
			if strings.Contains(a, "=") {
				continue
			}
			if f := flags.Lookup(a[2:]); f != nil && f.NoOptDefVal == "" {
				i++
			}
		case strings.HasPrefix(a, "-") && len(a) > 1:
			if len(a) == 2 {
				if f := flags.ShorthandLookup(a[1:]); f != nil && f.NoOptDefVal == "" {
					i++
				}
			}
		default:
			return i
		}
	}
	return -1
}

// findResourceCommand returns the runnable sub-command of parent that
// matches name, its alias, or its singular form.
func findResourceCommand(parent *cobra.Command, name string) *cobra.Command {
	for _, n := range []string{name, strings.TrimSuffix(name, "s")} {
		for _, c := range parent.Commands() {
			if c.Runnable() && c.Name() != "all" && (c.Name() == n || c.HasAlias(n)) {
				return c
			}
		}
	}
	return nil
}

// resourceGroupMatches reports whether suffix, either GROUP or
// VERSION.GROUP, belongs to a registered Flux kind named base.
func resourceGroupMatches(base, suffix string) bool {
	for gvk := range utils.NewScheme().AllKnownTypes() {
		kind := strings.ToLower(gvk.Kind)
		if base != kind && base != kind+"s" {
			continue
		}
		if suffix == gvk.Group || suffix == gvk.Version+"."+gvk.Group {
			return true
		}
	}
	return false
}
