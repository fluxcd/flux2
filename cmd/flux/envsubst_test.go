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
