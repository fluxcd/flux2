/*
Copyright 2023 The Flux authors

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
	"testing"

	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/ssa"
)

func Test_getFluxClusterInfo(t *testing.T) {
	g := NewWithT(t)
	f, err := os.Open("./testdata/cluster_info/gitrepositories.yaml")
	g.Expect(err).To(BeNil())

	objs, err := ssa.ReadObjects(f)
	g.Expect(err).To(Not(HaveOccurred()))
	gitrepo := objs[0]

	tests := []struct {
		name     string
		labels   map[string]string
		wantErr  bool
		wantInfo fluxClusterInfo
	}{
		{
			name:    "no git repository CRD present",
			wantErr: true,
		},
		{
			name: "CRD with kustomize-controller labels",
			labels: map[string]string{
				fmt.Sprintf("%s/name", kustomizev1.GroupVersion.Group):      "flux-system",
				fmt.Sprintf("%s/namespace", kustomizev1.GroupVersion.Group): "flux-system",
				"app.kubernetes.io/version":                                 "v2.1.0",
			},
			wantInfo: fluxClusterInfo{
				version:      "v2.1.0",
				bootstrapped: true,
			},
		},
		{
			name: "CRD with kustomize-controller labels and managed-by label",
			labels: map[string]string{
				fmt.Sprintf("%s/name", kustomizev1.GroupVersion.Group):      "flux-system",
				fmt.Sprintf("%s/namespace", kustomizev1.GroupVersion.Group): "flux-system",
				"app.kubernetes.io/version":                                 "v2.1.0",
				"app.kubernetes.io/managed-by":                              "flux",
			},
			wantInfo: fluxClusterInfo{
				version:      "v2.1.0",
				bootstrapped: true,
				managedBy:    "flux",
			},
		},
		{
			name: "CRD with only managed-by label",
			labels: map[string]string{
				"app.kubernetes.io/version":    "v2.1.0",
				"app.kubernetes.io/managed-by": "helm",
			},
			wantInfo: fluxClusterInfo{
				version:   "v2.1.0",
				managedBy: "helm",
			},
		},
		{
			name:     "CRD with no labels",
			labels:   map[string]string{},
			wantInfo: fluxClusterInfo{},
		},
		{
			name: "CRD with only version label",
			labels: map[string]string{
				"app.kubernetes.io/version": "v2.1.0",
			},
			wantInfo: fluxClusterInfo{
				version: "v2.1.0",
			},
		},
		{
			name: "CRD with version and part-of labels",
			labels: map[string]string{
				"app.kubernetes.io/version": "v2.1.0",
				"app.kubernetes.io/part-of": "flux",
			},
			wantInfo: fluxClusterInfo{
				version: "v2.1.0",
				partOf:  "flux",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			newscheme := runtime.NewScheme()
			apiextensionsv1.AddToScheme(newscheme)
			builder := fake.NewClientBuilder().WithScheme(newscheme)
			if tt.labels != nil {
				gitrepo.SetLabels(tt.labels)
				builder = builder.WithRuntimeObjects(gitrepo)
			}

			client := builder.Build()
			info, err := getFluxClusterInfo(context.Background(), client)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(errors.IsNotFound(err)).To(BeTrue())
			} else {
				g.Expect(err).To(Not(HaveOccurred()))
			}

			g.Expect(info).To(BeEquivalentTo(tt.wantInfo))
		})
	}
}
