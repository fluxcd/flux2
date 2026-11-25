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
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/fluxcd/cli-utils/pkg/object"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/fluxcd/flux2/v2/internal/tree"
	"github.com/fluxcd/flux2/v2/internal/utils"
)

// limitedTreeClient bounds recursive lookups so a regression cannot exhaust the stack.
type limitedTreeClient struct {
	client.Client
	gets     []client.ObjectKey
	limit    int
	limitErr error
}

// Get records lookups and stops traversals that exceed the fixture's bound.
func (c *limitedTreeClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	c.gets = append(c.gets, key)
	if len(c.gets) > c.limit {
		return c.limitErr
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

// TestTreeKustomizationTraversal checks cycles and preservation of distinct inventory paths.
func TestTreeKustomizationTraversal(t *testing.T) {
	key := func(namespace, name string) client.ObjectKey {
		return client.ObjectKey{Namespace: namespace, Name: name}
	}
	ks := func(namespace, name string, children ...client.ObjectKey) *kustomizev1.Kustomization {
		k := &kustomizev1.Kustomization{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
		if children != nil {
			k.Status.Inventory = &kustomizev1.ResourceInventory{}
			for _, child := range children {
				k.Status.Inventory.Entries = append(k.Status.Inventory.Entries, kustomizev1.ResourceRef{
					ID:      child.Namespace + "_" + child.Name + "_kustomize.toolkit.fluxcd.io_Kustomization",
					Version: "v1",
				})
			}
		}
		return k
	}
	a, b, c, d := key("ns", "a"), key("ns", "b"), key("ns", "c"), key("ns", "d")
	remote := ks("ns", "b", a, b)
	remote.Spec.KubeConfig = &meta.KubeConfigReference{SecretRef: &meta.SecretKeyReference{Name: "remote"}}
	nonFlux := ks("ns", "a", b)
	nonFlux.Status.Inventory.Entries = append(nonFlux.Status.Inventory.Entries, kustomizev1.ResourceRef{
		ID: "ns_config__ConfigMap", Version: "v1",
	})
	cases := []struct {
		name         string
		root         *kustomizev1.Kustomization
		objects      []*kustomizev1.Kustomization
		want         string
		wantCompact  string
		gets         []client.ObjectKey
		wantNotFound bool
	}{
		{
			name: "two node cycle",
			root: ks("ns", "a", b), objects: []*kustomizev1.Kustomization{ks("ns", "b", a)},
			want: "Kustomization/ns/a\n└── Kustomization/ns/b\n", gets: []client.ObjectKey{b},
		},
		{
			name: "three node cycle",
			root: ks("ns", "a", b), objects: []*kustomizev1.Kustomization{ks("ns", "b", c), ks("ns", "c", a)},
			want: "Kustomization/ns/a\n└── Kustomization/ns/b\n    └── Kustomization/ns/c\n", gets: []client.ObjectKey{b, c},
		},
		{
			name: "cycle below root",
			root: ks("ns", "a", b), objects: []*kustomizev1.Kustomization{ks("ns", "b", c), ks("ns", "c", b)},
			want: "Kustomization/ns/a\n└── Kustomization/ns/b\n    └── Kustomization/ns/c\n", gets: []client.ObjectKey{b, c},
		},
		{
			name: "direct self reference",
			root: ks("ns", "a", a), want: "Kustomization/ns/a\n",
		},
		{
			name: "same name in different namespaces",
			root: ks("ns", "a", key("other", "a")), objects: []*kustomizev1.Kustomization{ks("other", "a", a)},
			want: "Kustomization/ns/a\n└── Kustomization/other/a\n", gets: []client.ObjectKey{key("other", "a")},
		},
		{
			name: "shared descendant",
			root: ks("ns", "a", b, c), objects: []*kustomizev1.Kustomization{ks("ns", "b", d), ks("ns", "c", d), ks("ns", "d")},
			want: "Kustomization/ns/a\n├── Kustomization/ns/b\n│   └── Kustomization/ns/d\n└── Kustomization/ns/c\n    └── Kustomization/ns/d\n", gets: []client.ObjectKey{b, d, c, d},
		},
		{
			name: "nil inventory",
			root: ks("ns", "a"), want: "Kustomization/ns/a\n",
		},
		{
			name: "empty inventory",
			root: ks("ns", "a", []client.ObjectKey{}...), want: "Kustomization/ns/a\n",
		},
		{
			name: "remote inventory preserves ancestor leaf",
			root: ks("ns", "a", b), objects: []*kustomizev1.Kustomization{remote},
			want: "Kustomization/ns/a\n└── Kustomization/ns/b\n    └── Kustomization/ns/a\n", gets: []client.ObjectKey{b},
		},
		{
			name: "compact filtering",
			root: nonFlux, objects: []*kustomizev1.Kustomization{ks("ns", "b", a)},
			want:        "Kustomization/ns/a\n├── Kustomization/ns/b\n└── ConfigMap/ns/config\n",
			wantCompact: "Kustomization/ns/a\n└── Kustomization/ns/b\n", gets: []client.ObjectKey{b},
		},
		{
			name: "missing child error",
			root: ks("ns", "a", b), gets: []client.ObjectKey{b}, wantNotFound: true,
		},
	}
	for _, tc := range cases {
		for _, compact := range []bool{false, true} {
			name := tc.name
			if compact {
				name += " compact"
			}
			t.Run(name, func(t *testing.T) {
				objects := []client.Object{tc.root}
				for _, obj := range tc.objects {
					objects = append(objects, obj)
				}
				kubeClient := &limitedTreeClient{
					Client:   fake.NewClientBuilder().WithScheme(utils.NewScheme()).WithObjects(objects...).Build(),
					limit:    8,
					limitErr: errors.New("inventory traversal exceeded lookup limit"),
				}
				kTree := tree.New(object.ObjMetadata{
					Namespace: tc.root.Namespace, Name: tc.root.Name,
					GroupKind: schema.GroupKind{Group: kustomizev1.GroupVersion.Group, Kind: kustomizev1.KustomizationKind},
				})
				err := treeKustomization(context.Background(), kTree, tc.root, kubeClient, compact)
				if tc.wantNotFound {
					if !apierrors.IsNotFound(err) || !strings.Contains(err.Error(), "failed to find object") {
						t.Fatalf("expected wrapped not-found error, got %v", err)
					}
				} else {
					if err != nil {
						t.Fatalf("unexpected traversal error after %d lookups: %v", len(kubeClient.gets), err)
					}
					want := tc.want
					if compact && tc.wantCompact != "" {
						want = tc.wantCompact
					}
					if diff := cmp.Diff(want, kTree.Print()); diff != "" {
						t.Errorf("tree mismatch (-want +got):\n%s", diff)
					}
				}
				if diff := cmp.Diff(tc.gets, kubeClient.gets); diff != "" {
					t.Errorf("lookup mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}
