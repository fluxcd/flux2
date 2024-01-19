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

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sync"
)

var (
	ErrReconciledWithWarning = errors.New("reconciled with warning")
)

// Reconciler reconciles and reports the health of different
// components and kubernetes resources involved in the installation of Flux.
//
// It is recommended use the `ReconcilerWithSyncCheck` interface that also
// reports the health of the GitRepository.
type Reconciler interface {
	// ReconcileComponents reconciles the components by generating the
	// manifests with the provided values, committing them to Git and
	// pushing to remote if there are any changes, and applying them
	// to the cluster.
	ReconcileComponents(ctx context.Context, manifestsBase string, options install.Options, secretOpts sourcesecret.Options) error

	// ReconcileSourceSecret reconciles the source secret by generating
	// a new secret with the provided values if the secret does not
	// already exists on the cluster, or if any of the configuration
	// options changed.
	ReconcileSourceSecret(ctx context.Context, options sourcesecret.Options) error

	// ReconcileSyncConfig reconciles the sync configuration by generating
	// the sync manifests with the provided values, committing them to Git
	// and pushing to remote if there are any changes.
	ReconcileSyncConfig(ctx context.Context, options sync.Options) error

	// ReportKustomizationHealth reports about the health of the
	// Kustomization synchronizing the components.
	ReportKustomizationHealth(ctx context.Context, options sync.Options, pollInterval, timeout time.Duration) error

	// ReportComponentsHealth reports about the health for the components
	// and extra components in install.Options.
	ReportComponentsHealth(ctx context.Context, options install.Options, timeout time.Duration) error
}

type RepositoryReconciler interface {
	// ReconcileRepository reconciles an external Git repository.
	ReconcileRepository(ctx context.Context) error
}

// ReconcilerWithSyncCheck extends the Reconciler interface to also report the health of the GitReposiotry
// that syncs Flux on the cluster
type ReconcilerWithSyncCheck interface {
	Reconciler
	// ReportGitRepoHealth reports about the health of the GitRepository synchronizing the components.
	ReportGitRepoHealth(ctx context.Context, options sync.Options, pollInterval, timeout time.Duration) error
}

type PostGenerateSecretFunc func(ctx context.Context, secret corev1.Secret, options sourcesecret.Options) error

func Run(ctx context.Context, reconciler Reconciler, manifestsBase string,
	installOpts install.Options, secretOpts sourcesecret.Options, syncOpts sync.Options,
	pollInterval, timeout time.Duration) error {

	var err error
	if r, ok := reconciler.(RepositoryReconciler); ok {
		if err = r.ReconcileRepository(ctx); err != nil && !errors.Is(err, ErrReconciledWithWarning) {
			return err
		}
	}

	if err := reconciler.ReconcileComponents(ctx, manifestsBase, installOpts, secretOpts); err != nil {
		return err
	}
	if err := reconciler.ReconcileSourceSecret(ctx, secretOpts); err != nil {
		return err
	}
	if err := reconciler.ReconcileSyncConfig(ctx, syncOpts); err != nil {
		return err
	}

	var errs []error
	if r, ok := reconciler.(ReconcilerWithSyncCheck); ok {
		if err := r.ReportGitRepoHealth(ctx, syncOpts, pollInterval, timeout); err != nil {
			errs = append(errs, err)
		}
	}

	if err := reconciler.ReportKustomizationHealth(ctx, syncOpts, pollInterval, timeout); err != nil {
		errs = append(errs, err)
	}

	if err := reconciler.ReportComponentsHealth(ctx, installOpts, timeout); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		err = fmt.Errorf("bootstrap failed with %d health check failure(s): %w", len(errs), apierrors.NewAggregate(errs))
	}

	return err
}

func mustInstallManifests(ctx context.Context, kube client.Client, namespace string) bool {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      namespace,
	}
	var k kustomizev1.Kustomization
	if err := kube.Get(ctx, namespacedName, &k); err != nil {
		return true
	}
	return k.Status.LastAppliedRevision == ""
}

func secretExists(ctx context.Context, kube client.Client, objKey client.ObjectKey) (bool, error) {
	if err := kube.Get(ctx, objKey, &corev1.Secret{}); err != nil {
		if apierr.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func reconcileSecret(ctx context.Context, kube client.Client, secret corev1.Secret) error {
	objKey := client.ObjectKeyFromObject(&secret)
	var existing corev1.Secret
	err := kube.Get(ctx, objKey, &existing)
	if err != nil {
		if apierr.IsNotFound(err) {
			return kube.Create(ctx, &secret)
		}
		return err
	}
	existing.StringData = secret.StringData
	return kube.Update(ctx, &existing)
}

func kustomizationPathDiffers(ctx context.Context, kube client.Client, objKey client.ObjectKey, path string) (string, error) {
	var k kustomizev1.Kustomization
	if err := kube.Get(ctx, objKey, &k); err != nil {
		if apierr.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	normalizePath := func(p string) string {
		// remove the trailing '/' if the path is not './'
		if len(p) > 2 {
			p = strings.TrimSuffix(p, "/")
		}
		return fmt.Sprintf("./%s", strings.TrimPrefix(p, "./"))
	}
	if normalizePath(path) == normalizePath(k.Spec.Path) {
		return "", nil
	}
	return k.Spec.Path, nil
}

type objectWithConditions interface {
	client.Object
	GetConditions() []metav1.Condition
}

func objectReconciled(kube client.Client, objKey client.ObjectKey, clientObject objectWithConditions, expectRevision string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		// for some reason, TypeMeta gets unset after kube.Get so we want to store the GVK and set it after
		// ref https://github.com/kubernetes-sigs/controller-runtime/issues/1517#issuecomment-844703142
		gvk := clientObject.GetObjectKind().GroupVersionKind()
		if err := kube.Get(ctx, objKey, clientObject); err != nil {
			return false, err
		}
		clientObject.GetObjectKind().SetGroupVersionKind(gvk)

		kind := gvk.Kind
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(clientObject)
		if err != nil {
			return false, err
		}

		// Detect suspended object, as this would result in an endless wait
		if suspended, ok, _ := unstructured.NestedBool(obj, "spec", "suspend"); ok && suspended {
			return false, fmt.Errorf("%s '%s' is suspended", kind, objKey.String())
		}

		// Confirm the state we are observing is for the current generation
		if generation, ok, _ := unstructured.NestedInt64(obj, "status", "observedGeneration"); ok && generation != clientObject.GetGeneration() {
			return false, nil
		}

		// Confirm the resource is healthy
		if c := apimeta.FindStatusCondition(clientObject.GetConditions(), meta.ReadyCondition); c != nil {
			switch c.Status {
			case metav1.ConditionTrue:
				// Confirm the given revision has been attempted by the controller
				hasRev, err := hasRevision(kind, obj, expectRevision)
				if err != nil {
					return false, err
				}
				return hasRev, nil
			case metav1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}

// hasRevision checks that the reconciled revision (for Kustomization this is `.status.lastAttemptedRevision`
// and for Source APIs, it is stored in `.status.artifact.revision`) is the same as the expectedRev
func hasRevision(kind string, obj map[string]interface{}, expectedRev string) (bool, error) {
	var rev string
	switch kind {
	case sourcev1.GitRepositoryKind, sourcev1b2.OCIRepositoryKind, sourcev1b2.BucketKind, sourcev1b2.HelmChartKind:
		rev, _, _ = unstructured.NestedString(obj, "status", "artifact", "revision")
	case kustomizev1.KustomizationKind:
		rev, _, _ = unstructured.NestedString(obj, "status", "lastAttemptedRevision")
	default:
		return false, fmt.Errorf("cannot get status revision for kind: '%s'", kind)
	}
	return sourcev1b2.TransformLegacyRevision(rev) == expectedRev, nil
}

func retry(retries int, wait time.Duration, fn func() error) (err error) {
	for i := 0; ; i++ {
		err = fn()
		if err == nil {
			return
		}
		if i >= retries {
			break
		}
		time.Sleep(wait)
	}
	return err
}
