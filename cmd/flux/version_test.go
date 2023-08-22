/*
Copyright 2021 The Flux authors

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
	"testing"

	. "github.com/onsi/gomega"
)

func TestSplitImageStr(t *testing.T) {
	tests := []struct {
		url          string
		expectedName string
		expectedTag  string
	}{
		{
			url:          "fluxcd/notification-controller:v1.0.0",
			expectedName: "notification-controller",
			expectedTag:  "v1.0.0",
		},
		{
			url:          "ghcr.io/fluxcd/kustomize-controller:v1.0.0",
			expectedName: "kustomize-controller",
			expectedTag:  "v1.0.0",
		},
		{
			url:          "reg.internal:8080/fluxcd/source-controller:v1.0.0",
			expectedName: "source-controller",
			expectedTag:  "v1.0.0",
		},
		{
			url:          "fluxcd/source-controller:v1.0.1@sha256:49921d1c7b100650dd654a32df1f6e626b54dfe9707d7bb7bdf43fb7c81f1baf",
			expectedName: "source-controller",
			expectedTag:  "v1.0.1@sha256:49921d1c7b100650dd654a32df1f6e626b54dfe9707d7bb7bdf43fb7c81f1baf",
		},
	}

	for _, tt := range tests {
		g := NewWithT(t)
		n, t, err := splitImageStr(tt.url)
		g.Expect(err).To(Not(HaveOccurred()))
		g.Expect(n).To(BeEquivalentTo(tt.expectedName))
		g.Expect(t).To(BeEquivalentTo(tt.expectedTag))
	}
}
