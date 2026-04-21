/*
Copyright 2024 The Flux authors

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
	"bytes"
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func TestEnvsubst(t *testing.T) {
	g := NewWithT(t)
	input, err := os.ReadFile("testdata/envsubst/file.yaml")
	g.Expect(err).NotTo(HaveOccurred())

	t.Setenv("REPO_NAME", "test")

	output, err := executeCommandWithIn("envsubst", bytes.NewReader(input))
	g.Expect(err).NotTo(HaveOccurred())

	expected, err := os.ReadFile("testdata/envsubst/file.gold")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(output).To(Equal(string(expected)))
}

func TestEnvsubst_Strinct(t *testing.T) {
	g := NewWithT(t)
	input, err := os.ReadFile("testdata/envsubst/file.yaml")
	g.Expect(err).NotTo(HaveOccurred())

	_, err = executeCommandWithIn("envsubst --strict", bytes.NewReader(input))
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("variable not set (strict mode)"))
}

func TestEnvsubst_K8sAware(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		env     map[string]string
		input   string
		gold    string
		wantErr string
	}{
		{
			name:  "annotation disabled",
			args:  "envsubst --k8s-aware",
			env:   map[string]string{"REPO_NAME": "test"},
			input: "testdata/envsubst/k8s-aware.yaml",
			gold:  "testdata/envsubst/k8s-aware.gold",
		},
		{
			name:  "label disabled",
			args:  "envsubst --k8s-aware",
			input: "testdata/envsubst/k8s-aware-label.yaml",
			gold:  "testdata/envsubst/k8s-aware-label.gold",
		},
		{
			name:  "strict skips disabled resources",
			args:  "envsubst --strict --k8s-aware",
			env:   map[string]string{"REPO_NAME": "test"},
			input: "testdata/envsubst/k8s-aware.yaml",
			gold:  "testdata/envsubst/k8s-aware.gold",
		},
		{
			name:    "strict errors on enabled resource with missing var",
			args:    "envsubst --strict --k8s-aware",
			input:   "testdata/envsubst/k8s-aware.yaml",
			wantErr: "variable not set (strict mode)",
		},
		{
			name:  "bash script in disabled resource",
			args:  "envsubst --k8s-aware",
			env:   map[string]string{"APP_NAME": "myapp"},
			input: "testdata/envsubst/k8s-aware-bash.yaml",
			gold:  "testdata/envsubst/k8s-aware-bash.gold",
		},
		{
			name:  "strict with bash script in disabled resource",
			args:  "envsubst --strict --k8s-aware",
			env:   map[string]string{"APP_NAME": "myapp"},
			input: "testdata/envsubst/k8s-aware-bash.yaml",
			gold:  "testdata/envsubst/k8s-aware-bash.gold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			input, err := os.ReadFile(tt.input)
			g.Expect(err).NotTo(HaveOccurred())

			output, err := executeCommandWithIn(tt.args, bytes.NewReader(input))
			if tt.wantErr != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring(tt.wantErr))
				return
			}
			g.Expect(err).NotTo(HaveOccurred())

			expected, err := os.ReadFile(tt.gold)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal(string(expected)))
		})
	}
}
