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

package plugin

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// commandFunc is an alias to allow DI in tests.
var commandFunc = exec.Command

// CompleteFunc returns a ValidArgsFunction that delegates completion
// to the plugin binary via Cobra's __complete protocol.
func CompleteFunc(pluginPath string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		completeArgs := append([]string{"__complete"}, args...)
		completeArgs = append(completeArgs, toComplete)

		out, err := commandFunc(pluginPath, completeArgs...).Output()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		return parseCompletionOutput(string(out))
	}
}

// parseCompletionOutput parses Cobra's __complete output format.
// Each line is a completion, last line is :<directive_int>.
func parseCompletionOutput(out string) ([]string, cobra.ShellCompDirective) {
	out = strings.TrimRight(out, "\n")
	if out == "" {
		return nil, cobra.ShellCompDirectiveError
	}
	lines := strings.Split(out, "\n")

	// Last line is the directive in format ":N"
	lastLine := lines[len(lines)-1]
	completions := lines[:len(lines)-1]

	directive := cobra.ShellCompDirectiveDefault
	if strings.HasPrefix(lastLine, ":") {
		if val, err := strconv.Atoi(lastLine[1:]); err == nil {
			directive = cobra.ShellCompDirective(val)
		}
	}

	var results []string
	for _, c := range completions {
		if c == "" {
			continue
		}
		results = append(results, c)
	}

	return results, directive
}
