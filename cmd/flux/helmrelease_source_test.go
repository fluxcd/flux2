//go:build unit
// +build unit

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
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

func helmReleaseSourceFixtures() map[string]helmv2.HelmRelease {
	meta := metav1.ObjectMeta{Name: "podinfo", Namespace: "apps"}
	return map[string]helmv2.HelmRelease{
		"chartRef": {
			ObjectMeta: meta,
			Spec: helmv2.HelmReleaseSpec{
				ChartRef: &helmv2.CrossNamespaceSourceReference{
					Kind:      sourcev1.OCIRepositoryKind,
					Name:      "podinfo-chart",
					Namespace: "flux-system",
				},
			},
		},
		"HelmChart template": {
			ObjectMeta: meta,
			Spec: helmv2.HelmReleaseSpec{
				Chart: &helmv2.HelmChartTemplate{
					Spec: helmv2.HelmChartTemplateSpec{
						Chart: "podinfo",
						SourceRef: &helmv2.CrossNamespaceObjectReference{
							Kind:      sourcev1.HelmRepositoryKind,
							Name:      "podinfo",
							Namespace: "flux-system",
						},
					},
				},
			},
		},
		"HelmChart template without source namespace": {
			ObjectMeta: meta,
			Spec: helmv2.HelmReleaseSpec{
				Chart: &helmv2.HelmChartTemplate{
					Spec: helmv2.HelmChartTemplateSpec{
						Chart: "podinfo",
						SourceRef: &helmv2.CrossNamespaceObjectReference{
							Kind: sourcev1.GitRepositoryKind,
							Name: "podinfo",
						},
					},
				},
			},
		},
		"OCIRepository template": {
			ObjectMeta: meta,
			Spec: helmv2.HelmReleaseSpec{
				Chart: &helmv2.HelmChartTemplate{
					Kind: sourcev1.OCIRepositoryKind,
					Spec: helmv2.HelmChartTemplateSpec{
						OCIRepositorySpec: sourcev1.OCIRepositorySpec{
							URL: "oci://ghcr.io/stefanprodan/charts/podinfo",
						},
					},
				},
			},
		},
	}
}

func TestGetHelmReleaseSource(t *testing.T) {
	want := map[string]string{
		"chartRef":           "OCIRepository/flux-system/podinfo-chart",
		"HelmChart template": "HelmRepository/flux-system/podinfo",
		"HelmChart template without source namespace": "GitRepository/apps/podinfo",
		"OCIRepository template":                      "OCIRepository/apps/apps-podinfo",
	}

	for name, hr := range helmReleaseSourceFixtures() {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(getHelmReleaseSource(hr)).To(Equal(want[name]))
		})
	}
}

func TestHelmReleaseReconcileSource(t *testing.T) {
	want := map[string]struct {
		ref     sourceReference
		apiType apiType
	}{
		"chartRef": {
			ref:     sourceReference{kind: sourcev1.OCIRepositoryKind, name: "podinfo-chart", namespace: "flux-system"},
			apiType: ociRepositoryType,
		},
		"HelmChart template": {
			ref:     sourceReference{kind: sourcev1.HelmChartKind, name: "apps-podinfo", namespace: "flux-system"},
			apiType: helmChartType,
		},
		"HelmChart template without source namespace": {
			ref:     sourceReference{kind: sourcev1.HelmChartKind, name: "apps-podinfo", namespace: "apps"},
			apiType: helmChartType,
		},
		"OCIRepository template": {
			ref:     sourceReference{kind: sourcev1.OCIRepositoryKind, name: "apps-podinfo", namespace: "apps"},
			apiType: ociRepositoryType,
		},
	}

	for name, hr := range helmReleaseSourceFixtures() {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			src, ref := helmReleaseAdapter{&hr}.getSource()
			g.Expect(ref).To(Equal(want[name].ref))
			switch cmd := src.(type) {
			case reconcileCommand:
				g.Expect(cmd.apiType).To(Equal(want[name].apiType))
			case reconcileWithSourceCommand:
				g.Expect(cmd.apiType).To(Equal(want[name].apiType))
			default:
				t.Fatalf("unexpected reconcile command type %T", src)
			}
		})
	}
}
