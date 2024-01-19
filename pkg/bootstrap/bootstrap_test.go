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

package bootstrap

import (
	"context"
	"testing"

	"github.com/fluxcd/pkg/apis/meta"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/fluxcd/flux2/v2/internal/utils"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"
)

func Test_hasRevision(t *testing.T) {
	var revision = "main@sha1:5bf3a8f9bb0aa5ae8afd6208f43757ab73fc033a"

	tests := []struct {
		name         string
		obj          objectWithConditions
		rev          string
		expectErr    bool
		expectedBool bool
	}{
		{
			name: "Kustomization revision",
			obj: &kustomizev1.Kustomization{
				TypeMeta: metav1.TypeMeta{
					Kind: kustomizev1.KustomizationKind,
				},
				Status: kustomizev1.KustomizationStatus{
					LastAttemptedRevision: "main@sha1:5bf3a8f9bb0aa5ae8afd6208f43757ab73fc033a",
				},
			},
			expectedBool: true,
		},
		{
			name: "GitRepository revision",
			obj: &sourcev1.GitRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       sourcev1.GitRepositoryKind,
					APIVersion: sourcev1.GroupVersion.String(),
				},
				Status: sourcev1.GitRepositoryStatus{
					Artifact: &sourcev1.Artifact{
						Revision: "main@sha1:5bf3a8f9bb0aa5ae8afd6208f43757ab73fc033a",
					},
				},
			},
			expectedBool: true,
		},
		{
			name: "GitRepository revision (wrong revision)",
			obj: &sourcev1.GitRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       sourcev1.GitRepositoryKind,
					APIVersion: sourcev1.GroupVersion.String(),
				},
				Status: sourcev1.GitRepositoryStatus{
					Artifact: &sourcev1.Artifact{
						Revision: "main@sha1:e7f3a8f9bb0aa5ae8afd6208f43757ab73fc043a",
					},
				},
			},
		},
		{
			name: "Kustomization revision (empty revision)",
			obj: &kustomizev1.Kustomization{
				TypeMeta: metav1.TypeMeta{
					Kind: kustomizev1.KustomizationKind,
				},
				Status: kustomizev1.KustomizationStatus{
					LastAttemptedRevision: "",
				},
			},
		},
		{
			name: "OCIRepository revision",
			obj: &sourcev1b2.OCIRepository{
				TypeMeta: metav1.TypeMeta{
					Kind: sourcev1b2.OCIRepositoryKind,
				},
				Status: sourcev1b2.OCIRepositoryStatus{
					Artifact: &sourcev1.Artifact{
						Revision: "main@sha1:5bf3a8f9bb0aa5ae8afd6208f43757ab73fc033a",
					},
				},
			},
			expectedBool: true,
		},
		{
			name: "Alert revision(Not supported)",
			obj: &notificationv1.Alert{
				TypeMeta: metav1.TypeMeta{
					Kind: notificationv1.AlertKind,
				},
				Status: notificationv1.AlertStatus{
					ObservedGeneration: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tt.obj)
			g.Expect(err).To(BeNil())
			got, err := hasRevision(tt.obj.GetObjectKind().GroupVersionKind().Kind, obj, revision)
			if tt.expectErr {
				g.Expect(err).To(HaveOccurred())
				return
			}
			g.Expect(got).To(Equal(tt.expectedBool))
		})
	}
}

func Test_objectReconciled(t *testing.T) {
	expectedRev := "main@sha1:5bf3a8f9bb0aa5ae8afd6208f43757ab73fc033a"

	type updateStatus struct {
		statusFn     func(o client.Object)
		expectedErr  bool
		expectedBool bool
	}
	tests := []struct {
		name     string
		obj      objectWithConditions
		statuses []updateStatus
	}{
		{
			name: "GitRepository with no status",
			obj: &sourcev1.GitRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       sourcev1.GitRepositoryKind,
					APIVersion: sourcev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "flux-system",
					Namespace: "flux-system",
				},
			},
			statuses: []updateStatus{
				{
					expectedErr:  false,
					expectedBool: false,
				},
			},
		},
		{
			name: "suspended Kustomization",
			obj: &kustomizev1.Kustomization{
				TypeMeta: metav1.TypeMeta{
					Kind:       kustomizev1.KustomizationKind,
					APIVersion: kustomizev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "flux-system",
					Namespace: "flux-system",
				},
				Spec: kustomizev1.KustomizationSpec{
					Suspend: true,
				},
			},
			statuses: []updateStatus{
				{
					expectedErr: true,
				},
			},
		},
		{
			name: "Kustomization - status with old generation",
			obj: &kustomizev1.Kustomization{
				TypeMeta: metav1.TypeMeta{
					Kind:       kustomizev1.KustomizationKind,
					APIVersion: kustomizev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "flux-system",
					Namespace:  "flux-system",
					Generation: 1,
				},
				Status: kustomizev1.KustomizationStatus{
					ObservedGeneration: -1,
				},
			},
			statuses: []updateStatus{
				{
					expectedErr:  false,
					expectedBool: false,
				},
			},
		},
		{
			name: "GitRepository - status with same generation but no conditions",
			obj: &sourcev1.GitRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       sourcev1.GitRepositoryKind,
					APIVersion: sourcev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "flux-system",
					Namespace:  "flux-system",
					Generation: 1,
				},
				Status: sourcev1.GitRepositoryStatus{
					ObservedGeneration: 1,
				},
			},
			statuses: []updateStatus{
				{
					expectedErr:  false,
					expectedBool: false,
				},
			},
		},
		{
			name: "GitRepository - status with conditions but no ready condition",
			obj: &sourcev1.GitRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       sourcev1.GitRepositoryKind,
					APIVersion: sourcev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "flux-system",
					Namespace:  "flux-system",
					Generation: 1,
				},
				Status: sourcev1.GitRepositoryStatus{
					ObservedGeneration: 1,
					Conditions: []metav1.Condition{
						{Type: meta.ReconcilingCondition, Status: metav1.ConditionTrue, ObservedGeneration: 1, Reason: "Progressing", Message: "Progressing"},
					},
				},
			},
			statuses: []updateStatus{
				{
					expectedErr:  false,
					expectedBool: false,
				},
			},
		},
		{
			name: "Kustomization - status with false ready condition",
			obj: &kustomizev1.Kustomization{
				TypeMeta: metav1.TypeMeta{
					Kind:       kustomizev1.KustomizationKind,
					APIVersion: kustomizev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "flux-system",
					Namespace:  "flux-system",
					Generation: 1,
				},
				Status: kustomizev1.KustomizationStatus{
					ObservedGeneration: 1,
					Conditions: []metav1.Condition{
						{Type: meta.ReadyCondition, Status: metav1.ConditionFalse, ObservedGeneration: 1, Reason: "Failing", Message: "Failed to clone"},
					},
				},
			},
			statuses: []updateStatus{
				{
					expectedErr:  true,
					expectedBool: false,
				},
			},
		},
		{
			name: "Kustomization - status with true ready condition but different revision",
			obj: &kustomizev1.Kustomization{
				TypeMeta: metav1.TypeMeta{
					Kind:       kustomizev1.KustomizationKind,
					APIVersion: kustomizev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "flux-system",
					Namespace:  "flux-system",
					Generation: 1,
				},
				Status: kustomizev1.KustomizationStatus{
					ObservedGeneration: 1,
					Conditions: []metav1.Condition{
						{Type: meta.ReadyCondition, Status: metav1.ConditionTrue, ObservedGeneration: 1, Reason: "Passing", Message: "Applied revision"},
					},
					LastAttemptedRevision: "main@sha1:e7f3a8f9bb0aa5ae8afd6208f43757ab73fc043a",
				},
			},
			statuses: []updateStatus{
				{
					expectedErr:  false,
					expectedBool: false,
				},
			},
		},
		{
			name: "GitRepository - status with true ready condition but different revision",
			obj: &sourcev1.GitRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       sourcev1.GitRepositoryKind,
					APIVersion: sourcev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "flux-system",
					Namespace:  "flux-system",
					Generation: 1,
				},
				Status: sourcev1.GitRepositoryStatus{
					ObservedGeneration: 1,
					Conditions: []metav1.Condition{
						{Type: meta.ReadyCondition, Status: metav1.ConditionTrue, ObservedGeneration: 1, Reason: "Readyyy", Message: "Cloned successfully"},
					},
					Artifact: &sourcev1.Artifact{
						Revision: "main@sha1:e7f3a8f9bb0aa5ae8afd6208f43757ab73fc043a",
					},
				},
			},
			statuses: []updateStatus{
				{
					expectedErr:  false,
					expectedBool: false,
				},
			},
		},
		{
			name: "GitRepository - ready with right revision",
			obj: &sourcev1.GitRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       sourcev1.GitRepositoryKind,
					APIVersion: sourcev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "flux-system",
					Namespace:  "flux-system",
					Generation: 1,
				},
				Status: sourcev1.GitRepositoryStatus{
					ObservedGeneration: 1,
					Conditions: []metav1.Condition{
						{Type: meta.ReadyCondition, Status: metav1.ConditionTrue, ObservedGeneration: 1, Reason: "Readyyy", Message: "Cloned successfully"},
					},
					Artifact: &sourcev1.Artifact{
						Revision: expectedRev,
					},
				},
			},
			statuses: []updateStatus{
				{
					expectedErr:  false,
					expectedBool: true,
				},
			},
		},
		{
			name: "GitRepository - sequence of status updates before ready",
			obj: &sourcev1.GitRepository{
				TypeMeta: metav1.TypeMeta{
					Kind:       sourcev1.GitRepositoryKind,
					APIVersion: sourcev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "flux-system",
					Namespace:  "flux-system",
					Generation: 1,
				},
			},
			statuses: []updateStatus{
				{
					// observed gen different
					statusFn: func(o client.Object) {
						gitRepo := o.(*sourcev1.GitRepository)
						gitRepo.Status = sourcev1.GitRepositoryStatus{
							ObservedGeneration: -1,
						}
					},
				},
				{
					// ready failing
					statusFn: func(o client.Object) {
						gitRepo := o.(*sourcev1.GitRepository)
						gitRepo.Status = sourcev1.GitRepositoryStatus{
							ObservedGeneration: 1,
							Conditions: []metav1.Condition{
								{Type: meta.ReadyCondition, Status: metav1.ConditionFalse, ObservedGeneration: 1, Reason: "Not Ready", Message: "Transient connection issue"},
							},
						}
					},
					expectedErr: true,
				},
				{
					// updated to a different revision
					statusFn: func(o client.Object) {
						gitRepo := o.(*sourcev1.GitRepository)
						gitRepo.Status = sourcev1.GitRepositoryStatus{
							ObservedGeneration: 1,
							Conditions: []metav1.Condition{
								{Type: meta.ReadyCondition, Status: metav1.ConditionTrue, ObservedGeneration: 1, Reason: "Readyyy", Message: "Cloned successfully"},
							},
							Artifact: &sourcev1.Artifact{
								Revision: "wrong rev",
							},
						}
					},
				},
				{
					// updated to the expected revision
					statusFn: func(o client.Object) {
						gitRepo := o.(*sourcev1.GitRepository)
						gitRepo.Status = sourcev1.GitRepositoryStatus{
							ObservedGeneration: 1,
							Conditions: []metav1.Condition{
								{Type: meta.ReadyCondition, Status: metav1.ConditionTrue, ObservedGeneration: 1, Reason: "Readyyy", Message: "Cloned successfully"},
							},
							Artifact: &sourcev1.Artifact{
								Revision: expectedRev,
							},
						}
					},
					expectedBool: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			builder := fake.NewClientBuilder().WithScheme(utils.NewScheme())
			builder.WithObjects(tt.obj)

			kubeClient := builder.Build()

			for _, updates := range tt.statuses {
				if updates.statusFn != nil {
					updates.statusFn(tt.obj)
					g.Expect(kubeClient.Update(context.TODO(), tt.obj)).To(Succeed())
				}

				waitFunc := objectReconciled(kubeClient, client.ObjectKeyFromObject(tt.obj), tt.obj, expectedRev)
				got, err := waitFunc(context.TODO())
				g.Expect(err != nil).To(Equal(updates.expectedErr))
				g.Expect(got).To(Equal(updates.expectedBool))
			}
		})
	}
}
