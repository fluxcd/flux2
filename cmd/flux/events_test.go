/*
Copyright 2023 The Kubernetes Authors.
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
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	eventv1 "github.com/fluxcd/pkg/apis/event/v1beta1"
	"github.com/fluxcd/pkg/ssa"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var objects = `
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: flux-system
  namespace: flux-system
spec:
  interval: 5m0s
  path: ./infrastructure/
  prune: true
  sourceRef:
    kind: GitRepository
    name: flux-system
---
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: podinfo
  namespace: default
spec:
  interval: 5m0s
  path: ./infrastructure/
  prune: true
  sourceRef:
    kind: GitRepository
    name: flux-system
    namespace: flux-system
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: flux-system
  namespace: flux-system
spec:
  interval: 5m0s
  ref:
    branch: main
  secretRef:
    name: flux-system
  timeout: 1m0s
  url: ssh://git@github.com/example/repo
---
apiVersion: helm.toolkit.fluxcd.io/v2beta2
kind: HelmRelease
metadata:
  name: podinfo
  namespace: default
spec:
  chart:
    spec:
      chart: podinfo
      reconcileStrategy: ChartVersion
      sourceRef:
        kind: HelmRepository
        name: podinfo
        namespace: flux-system
      version: '*'
  interval: 5m0s
---
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: HelmRepository
metadata:
  name: podinfo
  namespace: flux-system
spec:
  interval: 1m0s
  url: https://stefanprodan.github.io/podinfo
---
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: HelmChart
metadata:
  name: default-podinfo
  namespace: flux-system
spec:
  chart: podinfo
  interval: 1m0s
  reconcileStrategy: ChartVersion
  sourceRef:
    kind: HelmRepository
    name: podinfo-chart
  version: '*'
---
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Alert
metadata:
  name: webapp
  namespace: flux-system
spec:
  eventSeverity: info
  eventSources:
  - kind: GitRepository
    name: '*'
  providerRef:
    name: slack
---
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Provider
metadata:
  name: slack
  namespace: flux-system
spec:
  address: https://hooks.slack.com/services/mock
  type: slack
---
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImagePolicy
metadata:
  name: podinfo
  namespace: default
spec:
  imageRepositoryRef:
    name: acr-podinfo
    namespace: flux-system
  policy:
    semver:
      range: 5.0.x
---
apiVersion: v1
kind: Namespace
metadata:
  name: flux-system`

func Test_getObjectRef(t *testing.T) {
	g := NewWithT(t)
	objs, err := ssa.ReadObjects(strings.NewReader(objects))
	g.Expect(err).To(Not(HaveOccurred()))

	builder := fake.NewClientBuilder().WithScheme(utils.NewScheme())
	for _, obj := range objs {
		builder = builder.WithObjects(obj)
	}
	c := builder.Build()

	tests := []struct {
		name      string
		selector  string
		namespace string
		want      []string
		wantErr   bool
	}{
		{
			name:      "Source Ref for Kustomization",
			selector:  "Kustomization/flux-system",
			namespace: "flux-system",
			want:      []string{"GitRepository/flux-system.flux-system"},
		},
		{
			name:      "Crossnamespace Source Ref for Kustomization",
			selector:  "Kustomization/podinfo",
			namespace: "default",
			want:      []string{"GitRepository/flux-system.flux-system"},
		},
		{
			name:      "Source Ref for HelmRelease",
			selector:  "HelmRelease/podinfo",
			namespace: "default",
			want:      []string{"HelmRepository/podinfo.flux-system", "HelmChart/default-podinfo.flux-system"},
		},
		{
			name:      "Source Ref for Alert",
			selector:  "Alert/webapp",
			namespace: "flux-system",
			want:      []string{"Provider/slack.flux-system"},
		},
		{
			name:      "Source Ref for ImagePolicy",
			selector:  "ImagePolicy/podinfo",
			namespace: "default",
			want:      []string{"ImageRepository/acr-podinfo.flux-system"},
		},
		{
			name:      "Source Ref for ImagePolicy (lowercased)",
			selector:  "imagepolicy/podinfo",
			namespace: "default",
			want:      []string{"ImageRepository/acr-podinfo.flux-system"},
		},
		{
			name:      "Empty Ref for Provider",
			selector:  "Provider/slack",
			namespace: "flux-system",
			want:      nil,
		},
		{
			name:     "Non flux resource",
			selector: "Namespace/flux-system",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			kind, name := getKindNameFromSelector(tt.selector)
			infoRef, err := fluxKindMap.getRefInfo(kind)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				return
			}
			got, err := getObjectRef(context.Background(), c, infoRef, name, tt.namespace)

			g.Expect(err).To(Not(HaveOccurred()))
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_getRows(t *testing.T) {
	g := NewWithT(t)
	objs, err := ssa.ReadObjects(strings.NewReader(objects))
	g.Expect(err).To(Not(HaveOccurred()))

	builder := fake.NewClientBuilder().WithScheme(utils.NewScheme())
	for _, obj := range objs {
		builder = builder.WithObjects(obj)
	}
	eventList := &corev1.EventList{}
	for _, obj := range objs {
		infoEvent := createEvent(obj, eventv1.EventSeverityInfo, "Info Message", "Info Reason")
		warningEvent := createEvent(obj, eventv1.EventSeverityError, "Error Message", "Error Reason")
		eventList.Items = append(eventList.Items, infoEvent, warningEvent)
	}
	builder = builder.WithLists(eventList)
	builder.WithIndex(&corev1.Event{}, "involvedObject.kind/name", kindNameIndexer)
	builder.WithIndex(&corev1.Event{}, "involvedObject.kind", kindIndexer)
	c := builder.Build()

	tests := []struct {
		name        string
		selector    string
		refSelector string
		namespace   string
		refNs       string
		expected    [][]string
	}{
		{
			name:      "events from all namespaces",
			selector:  "",
			namespace: "",
			expected: [][]string{
				{"default", "<unknown>", "error", "Error Reason", "HelmRelease/podinfo", "Error Message"},
				{"default", "<unknown>", "info", "Info Reason", "HelmRelease/podinfo", "Info Message"},
				{"default", "<unknown>", "error", "Error Reason", "ImagePolicy/podinfo", "Error Message"},
				{"default", "<unknown>", "info", "Info Reason", "ImagePolicy/podinfo", "Info Message"},
				{"default", "<unknown>", "error", "Error Reason", "Kustomization/podinfo", "Error Message"},
				{"default", "<unknown>", "info", "Info Reason", "Kustomization/podinfo", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "Alert/webapp", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "Alert/webapp", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "GitRepository/flux-system", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "GitRepository/flux-system", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "HelmChart/default-podinfo", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "HelmChart/default-podinfo", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "HelmRepository/podinfo", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "HelmRepository/podinfo", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "Kustomization/flux-system", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "Kustomization/flux-system", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "Provider/slack", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "Provider/slack", "Info Message"},
			},
		},
		{
			name:      "events from default namespaces",
			selector:  "",
			namespace: "default",
			expected: [][]string{
				{"<unknown>", "error", "Error Reason", "HelmRelease/podinfo", "Error Message"},
				{"<unknown>", "info", "Info Reason", "HelmRelease/podinfo", "Info Message"},
				{"<unknown>", "error", "Error Reason", "ImagePolicy/podinfo", "Error Message"},
				{"<unknown>", "info", "Info Reason", "ImagePolicy/podinfo", "Info Message"},
				{"<unknown>", "error", "Error Reason", "Kustomization/podinfo", "Error Message"},
				{"<unknown>", "info", "Info Reason", "Kustomization/podinfo", "Info Message"},
			},
		},
		{
			name:      "Kustomization with crossnamespaced GitRepository",
			selector:  "Kustomization/podinfo",
			namespace: "default",
			expected: [][]string{
				{"default", "<unknown>", "error", "Error Reason", "Kustomization/podinfo", "Error Message"},
				{"default", "<unknown>", "info", "Info Reason", "Kustomization/podinfo", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "GitRepository/flux-system", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "GitRepository/flux-system", "Info Message"},
			},
		},
		{
			name:     "All Kustomization (lowercased selector)",
			selector: "kustomization",
			expected: [][]string{
				{"default", "<unknown>", "error", "Error Reason", "Kustomization/podinfo", "Error Message"},
				{"default", "<unknown>", "info", "Info Reason", "Kustomization/podinfo", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "Kustomization/flux-system", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "Kustomization/flux-system", "Info Message"},
			},
		},
		{
			name:      "HelmRelease with crossnamespaced HelmRepository",
			selector:  "HelmRelease/podinfo",
			namespace: "default",
			expected: [][]string{
				{"default", "<unknown>", "error", "Error Reason", "HelmRelease/podinfo", "Error Message"},
				{"default", "<unknown>", "info", "Info Reason", "HelmRelease/podinfo", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "HelmRepository/podinfo", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "HelmRepository/podinfo", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "HelmChart/default-podinfo", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "HelmChart/default-podinfo", "Info Message"},
			},
		},
		{
			name:      "HelmRelease with crossnamespaced HelmRepository (lowercased)",
			selector:  "helmrelease/podinfo",
			namespace: "default",
			expected: [][]string{
				{"default", "<unknown>", "error", "Error Reason", "HelmRelease/podinfo", "Error Message"},
				{"default", "<unknown>", "info", "Info Reason", "HelmRelease/podinfo", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "HelmRepository/podinfo", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "HelmRepository/podinfo", "Info Message"},
				{"flux-system", "<unknown>", "error", "Error Reason", "HelmChart/default-podinfo", "Error Message"},
				{"flux-system", "<unknown>", "info", "Info Reason", "HelmChart/default-podinfo", "Info Message"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			var refs []string
			var refNs, refKind, refName string
			var clientOpts = []client.ListOption{client.InNamespace(tt.namespace)}
			if tt.selector != "" {
				kind, name := getKindNameFromSelector(tt.selector)
				infoRef, err := fluxKindMap.getRefInfo(kind)
				clientOpts = append(clientOpts, getTestListOpt(infoRef.gvk.Kind, name))
				if name != "" {
					g.Expect(err).To(Not(HaveOccurred()))
					refs, err = getObjectRef(context.Background(), c, infoRef, name, tt.namespace)
					g.Expect(err).To(Not(HaveOccurred()))
				}
			}

			g.Expect(err).To(Not(HaveOccurred()))

			var refOpts [][]client.ListOption
			for _, ref := range refs {
				refKind, refName, refNs = utils.ParseObjectKindNameNamespace(ref)
				refOpts = append(refOpts, []client.ListOption{client.InNamespace(refNs), getTestListOpt(refKind, refName)})
			}

			showNs := tt.namespace == "" || (refNs != "" && refNs != tt.namespace)
			rows, err := getRows(context.Background(), c, clientOpts, refOpts, showNs)
			g.Expect(err).To(Not(HaveOccurred()))
			g.Expect(rows).To(ConsistOf(tt.expected))
		})
	}
}

func getTestListOpt(kind, name string) client.ListOption {
	var sel fields.Selector
	if name == "" {
		sel = fields.OneTermEqualSelector("involvedObject.kind", kind)
	} else {
		sel = fields.OneTermEqualSelector("involvedObject.kind/name", fmt.Sprintf("%s/%s", kind, name))
	}
	return client.MatchingFieldsSelector{Selector: sel}
}

func createEvent(obj client.Object, eventType, msg, reason string) corev1.Event {
	return corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: obj.GetNamespace(),
			// name of event needs to be unique
			Name: obj.GetNamespace() + obj.GetNamespace() + obj.GetObjectKind().GroupVersionKind().Kind + eventType,
		},
		Reason:  reason,
		Message: msg,
		Type:    eventType,
		InvolvedObject: corev1.ObjectReference{
			Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		},
	}
}

func kindNameIndexer(obj client.Object) []string {
	e, ok := obj.(*corev1.Event)
	if !ok {
		panic(fmt.Sprintf("Expected a Event, got %T", e))
	}

	return []string{fmt.Sprintf("%s/%s", e.InvolvedObject.Kind, e.InvolvedObject.Name)}
}

func kindIndexer(obj client.Object) []string {
	e, ok := obj.(*corev1.Event)
	if !ok {
		panic(fmt.Sprintf("Expected a Event, got %T", e))
	}

	return []string{e.InvolvedObject.Kind}
}
