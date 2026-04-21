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
	"testing"

	"github.com/spf13/cobra"
)

func TestParseCompletionOutput(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		expectedCompletions []string
		expectedDirective   cobra.ShellCompDirective
	}{
		{
			name:                "standard output",
			input:               "instance\nrset\nrsip\nall\n:4\n",
			expectedCompletions: []string{"instance", "rset", "rsip", "all"},
			expectedDirective:   cobra.ShellCompDirective(4),
		},
		{
			name:                "default directive",
			input:               "foo\nbar\n:0\n",
			expectedCompletions: []string{"foo", "bar"},
			expectedDirective:   cobra.ShellCompDirectiveDefault,
		},
		{
			name:                "with descriptions",
			input:               "get\tGet resources\nbuild\tBuild resources\n:4\n",
			expectedCompletions: []string{"get\tGet resources", "build\tBuild resources"},
			expectedDirective:   cobra.ShellCompDirective(4),
		},
		{
			name:                "empty completions",
			input:               ":4\n",
			expectedCompletions: nil,
			expectedDirective:   cobra.ShellCompDirective(4),
		},
		{
			name:                "empty input",
			input:               "",
			expectedCompletions: nil,
			expectedDirective:   cobra.ShellCompDirectiveError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, directive := parseCompletionOutput(tt.input)
			if directive != tt.expectedDirective {
				t.Errorf("directive: got %d, want %d", directive, tt.expectedDirective)
			}
			if len(completions) != len(tt.expectedCompletions) {
				t.Fatalf("completions count: got %d, want %d", len(completions), len(tt.expectedCompletions))
			}
			for i, c := range completions {
				if c != tt.expectedCompletions[i] {
					t.Errorf("completion[%d]: got %q, want %q", i, c, tt.expectedCompletions[i])
				}
			}
		})
	}
}
