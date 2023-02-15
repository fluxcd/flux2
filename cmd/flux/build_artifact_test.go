/*
Copyright 2022 The Flux authors

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
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func Test_saveReaderToFile(t *testing.T) {
	g := NewWithT(t)

	testString := `apiVersion: v1
kind: ConfigMap
metadata:
  name: myapp
data:
  foo: bar`

	tests := []struct {
		name      string
		string    string
		expectErr bool
	}{
		{
			name:   "yaml",
			string: testString,
		},
		{
			name:   "yaml with carriage return",
			string: testString + "\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := saveReaderToFile(strings.NewReader(tt.string))
			g.Expect(err).To(BeNil())

			t.Cleanup(func() { _ = os.Remove(tmpFile) })

			b, err := os.ReadFile(tmpFile)
			if tt.expectErr {
				g.Expect(err).To(Not(BeNil()))
				return
			}

			g.Expect(err).To(BeNil())
			g.Expect(string(b)).To(BeEquivalentTo(testString))
		})

	}
}
