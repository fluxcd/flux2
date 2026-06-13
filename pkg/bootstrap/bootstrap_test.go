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
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"io"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/git"
	gogit "github.com/fluxcd/pkg/git/gogit"
	"github.com/fluxcd/pkg/git/repository"
	"github.com/fluxcd/pkg/git/signature"
	extgogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	. "github.com/onsi/gomega"
	gossh "golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	"github.com/fluxcd/flux2/v2/internal/utils"
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
					Artifact: &meta.Artifact{
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
					Artifact: &meta.Artifact{
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
			obj: &sourcev1.OCIRepository{
				TypeMeta: metav1.TypeMeta{
					Kind: sourcev1.OCIRepositoryKind,
				},
				Status: sourcev1.OCIRepositoryStatus{
					Artifact: &meta.Artifact{
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
					Artifact: &meta.Artifact{
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
					Artifact: &meta.Artifact{
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
							Artifact: &meta.Artifact{
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
							Artifact: &meta.Artifact{
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
					cloneObj := tt.obj.DeepCopyObject().(client.Object)
					g.Expect(kubeClient.Update(context.TODO(), cloneObj)).To(Succeed())
				}

				waitFunc := objectReconciled(kubeClient, client.ObjectKeyFromObject(tt.obj), tt.obj, expectedRev)
				got, err := waitFunc(context.TODO())
				g.Expect(err != nil).To(Equal(updates.expectedErr), "unexpected error: %v, for: %v", err, tt.obj)
				g.Expect(got).To(Equal(updates.expectedBool))
			}
		})
	}
}

func TestPlainGitBootstrapper_resolveSigner(t *testing.T) {
	t.Run("no signing configured returns nil signer", func(t *testing.T) {
		g := NewWithT(t)
		b := &PlainGitBootstrapper{}
		signer, err := b.resolveSigner()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(signer).To(BeNil())
	})

	t.Run("GPG key ring returns an OpenPGP signer", func(t *testing.T) {
		g := NewWithT(t)
		entity, err := openpgp.NewEntity("Alice", "test", "alice@example.com", nil)
		g.Expect(err).ToNot(HaveOccurred())
		b := &PlainGitBootstrapper{gpgKeyRing: openpgp.EntityList{entity}}
		signer, err := b.resolveSigner()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(signer).ToNot(BeNil())
	})

	t.Run("SSH key returns an SSH signer", func(t *testing.T) {
		g := NewWithT(t)
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		g.Expect(err).ToNot(HaveOccurred())
		block, err := gossh.MarshalPrivateKey(priv, "test ed25519 key")
		g.Expect(err).ToNot(HaveOccurred())
		pemBytes := pem.EncodeToMemory(block)

		b := &PlainGitBootstrapper{sshSigningKey: pemBytes}
		signer, err := b.resolveSigner()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(signer).ToNot(BeNil())
	})

	t.Run("encrypted SSH key without password errors", func(t *testing.T) {
		g := NewWithT(t)
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		g.Expect(err).ToNot(HaveOccurred())
		block, err := gossh.MarshalPrivateKeyWithPassphrase(priv, "test ed25519 key", []byte("pw"))
		g.Expect(err).ToNot(HaveOccurred())
		pemBytes := pem.EncodeToMemory(block)

		b := &PlainGitBootstrapper{sshSigningKey: pemBytes}
		_, err = b.resolveSigner()
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("GPG path takes precedence over SSH path", func(t *testing.T) {
		g := NewWithT(t)
		entity, err := openpgp.NewEntity("Alice", "test", "alice@example.com", nil)
		g.Expect(err).ToNot(HaveOccurred())
		b := &PlainGitBootstrapper{
			gpgKeyRing:    openpgp.EntityList{entity},
			sshSigningKey: []byte("ignored"),
		}
		signer, err := b.resolveSigner()
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(signer).ToNot(BeNil())
	})
}

// TestPlainGitBootstrapper_sshSignerProducesVerifiableCommit is an
// end-to-end wiring test. resolveSigner already has unit tests for
// dispatch behaviour, but nothing in pkg/bootstrap exercises the full
// path from sshSigningKey → resolveSigner → repository.WithSigner →
// gogit.Client.Commit → gpgsig header on the resulting commit object.
// This test drives that path and then verifies the signature via
// signature.VerifySSHSignature, catching regressions that the existing
// dispatcher unit tests would miss.
func TestPlainGitBootstrapper_sshSignerProducesVerifiableCommit(t *testing.T) {
	g := NewWithT(t)

	// Generate an ed25519 keypair and marshal the private key to PEM.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	g.Expect(err).ToNot(HaveOccurred())
	pemBlock, err := gossh.MarshalPrivateKey(priv, "test ed25519 key")
	g.Expect(err).ToNot(HaveOccurred())
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Resolve a Signer via the same path the bootstrap commit code uses.
	b := &PlainGitBootstrapper{sshSigningKey: pemBytes}
	signer, err := b.resolveSigner()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(signer).ToNot(BeNil())

	// Initialise a gogit.Client against a fresh on-disk repo. Init sets
	// the internal repository pointer so that Commit can operate.
	tmp := t.TempDir()
	gogitClient, err := gogit.NewClient(tmp, nil)
	g.Expect(err).ToNot(HaveOccurred())
	// Use a file:// URL; Init only records the remote URL, it does not
	// actually connect, so any syntactically valid URL works here.
	g.Expect(gogitClient.Init(context.Background(), "file:///dev/null", git.DefaultBranch)).To(Succeed())

	// Drive a commit through the same gogit pipeline bootstrap uses.
	hash, err := gogitClient.Commit(
		git.Commit{
			Author:  git.Signature{Name: "Test", Email: "test@example.com"},
			Message: "ssh-signed test commit",
		},
		repository.WithFiles(map[string]io.Reader{
			"signed-file": strings.NewReader("hello sshsig"),
		}),
		repository.WithSigner(signer),
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Read the commit object back via a plain go-git open of the same path.
	repo, err := extgogit.PlainOpen(tmp)
	g.Expect(err).ToNot(HaveOccurred())
	commit, err := repo.CommitObject(plumbing.NewHash(hash))
	g.Expect(err).ToNot(HaveOccurred())

	// The commit must carry an SSH signature header.
	g.Expect(commit.PGPSignature).To(HavePrefix("-----BEGIN SSH SIGNATURE-----"))

	// Reconstruct the canonical payload (commit without signature) and
	// run the full cryptographic verification against the known public key.
	encoded := &plumbing.MemoryObject{}
	g.Expect(commit.EncodeWithoutSignature(encoded)).To(Succeed())
	payloadReader, err := encoded.Reader()
	g.Expect(err).ToNot(HaveOccurred())
	payload, err := io.ReadAll(payloadReader)
	g.Expect(err).ToNot(HaveOccurred())

	gosshPub, err := gossh.NewPublicKey(pub)
	g.Expect(err).ToNot(HaveOccurred())
	authorizedKey := gossh.MarshalAuthorizedKey(gosshPub)

	_, err = signature.VerifySSHSignature(commit.PGPSignature, payload, string(authorizedKey))
	g.Expect(err).ToNot(HaveOccurred())
}
