/*
Copyright 2020 The Flux authors

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
	"fmt"
	"time"

	kstatus "github.com/fluxcd/cli-utils/pkg/kstatus/status"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Reconcile sources and resources",
	Long:  `The reconcile sub-commands trigger a reconciliation of sources and resources.`,
	Example: `  # Trigger a reconciliation of a Kustomization
  flux reconcile kustomization podinfo

  # Trigger a reconciliation using the kubectl-style TYPE/NAME form
  flux reconcile ks/podinfo

  # Trigger a reconciliation of multiple Kustomizations
  flux reconcile kustomization frontend backend

  # Trigger a reconciliation of a Git repository by its kind, short name or fully qualified type
  flux reconcile gitrepository/podinfo
  flux reconcile gitrepo/podinfo
  flux reconcile gitrepositories.source.toolkit.fluxcd.io/podinfo`,
}

func init() {
	rootCmd.AddCommand(reconcileCmd)
}

type reconcileCommand struct {
	apiType
	object reconcilable
}

type reconcilable interface {
	adapter     // to be able to load from the cluster
	copyable    // to be able to calculate patches
	suspendable // to tell if it's suspended

	// these are implemented by anything embedding metav1.ObjectMeta
	GetAnnotations() map[string]string
	SetAnnotations(map[string]string)

	isStatic() bool                      // is it a static object that does not have a reconciler?
	lastHandledReconcileRequest() string // what was the last handled reconcile request?
	successMessage() string              // what do you want to tell people when successfully reconciled?
}

func reconcilableConditions(object reconcilable) []metav1.Condition {
	if s, ok := object.(meta.ObjectWithConditions); ok {
		return s.GetConditions()
	}

	if s, ok := object.(oldConditions); ok {
		return *s.GetStatusConditions()
	}

	return []metav1.Condition{}
}

func (reconcile reconcileCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", reconcile.kind)
	}
	return reconcileNames(reconcile.kind, args, reconcile.reconcile)
}

func (reconcile reconcileCommand) reconcile(name string) error {
	resetObject(reconcile.object.asClientObject())

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: *kubeconfigArgs.Namespace,
		Name:      name,
	}

	err = kubeClient.Get(ctx, namespacedName, reconcile.object.asClientObject())
	if err != nil {
		return objectGetError{err}
	}

	if reconcile.object.isStatic() {
		logger.Successf("reconciliation not supported by the object")
		return nil
	}

	if reconcile.object.isSuspended() {
		return fmt.Errorf("resource is suspended")
	}

	logger.Actionf("annotating %s %s in %s namespace", reconcile.kind, name, *kubeconfigArgs.Namespace)
	if err := requestReconciliation(ctx, kubeClient, namespacedName,
		reconcile.groupVersion.WithKind(reconcile.kind)); err != nil {
		return err
	}
	logger.Successf("%s annotated", reconcile.kind)

	lastHandledReconcileAt := reconcile.object.lastHandledReconcileRequest()
	logger.Waitingf("waiting for %s reconciliation", reconcile.kind)
	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true,
		reconciliationHandled(kubeClient, namespacedName, reconcile.object, lastHandledReconcileAt)); err != nil {
		return err
	}
	readyCond := apimeta.FindStatusCondition(reconcilableConditions(reconcile.object), meta.ReadyCondition)
	if readyCond == nil {
		return fmt.Errorf("status can't be determined")
	}

	if readyCond.Status != metav1.ConditionTrue {
		return fmt.Errorf("%s reconciliation failed: '%s'", reconcile.kind, readyCond.Message)
	}
	logger.Successf("%s", reconcile.object.successMessage())
	return nil
}

// reconcileNames calls fn for each name in order, skipping duplicates.
// With a single name the error from fn is returned as is. With several
// names, failures are logged and an aggregated error is returned at the end.
func reconcileNames(kind string, args []string, fn func(name string) error) error {
	names := uniqueNames(args)

	if len(names) == 1 {
		return unwrapGetError(fn(names[0]))
	}

	failed := 0
	for _, name := range names {
		if err := fn(name); err != nil {
			failed++
			var getErr objectGetError
			if errors.As(err, &getErr) && apierrors.IsNotFound(getErr.err) {
				logger.Failuref("%s %s not found in %s namespace", kind, name, *kubeconfigArgs.Namespace)
			} else {
				logger.Failuref("failed to reconcile %s %s in %s namespace: %s", kind, name, *kubeconfigArgs.Namespace, err.Error())
			}
		}
	}
	if failed > 0 {
		return fmt.Errorf("failed to reconcile %d of %d %s objects", failed, len(names), kind)
	}
	return nil
}

// objectGetError marks an error returned when fetching the named object,
// as opposed to errors from its source or from waiting.
type objectGetError struct{ err error }

func (e objectGetError) Error() string { return e.err.Error() }
func (e objectGetError) Unwrap() error { return e.err }

// unwrapGetError returns the error wrapped by objectGetError, or err as is.
func unwrapGetError(err error) error {
	var getErr objectGetError
	if errors.As(err, &getErr) {
		return getErr.err
	}
	return err
}

func reconciliationHandled(kubeClient client.Client, namespacedName types.NamespacedName, obj reconcilable, lastHandledReconcileAt string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, obj.asClientObject())
		if err != nil {
			return false, err
		}

		if obj.lastHandledReconcileRequest() == lastHandledReconcileAt {
			return false, nil
		}
		if r, ok := obj.(reconcileRequester); ok {
			if current, err := isRequesterCurrent(r); !current || err != nil {
				return false, err
			}
		}

		result, err := kstatusCompute(obj.asClientObject())
		if err != nil {
			return false, err
		}

		switch result.Status {
		case kstatus.CurrentStatus:
			return true, nil
		case kstatus.InProgressStatus:
			return false, nil
		default:
			return false, fmt.Errorf("%s", result.Message)
		}
	}
}

func requestReconciliation(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, gvk schema.GroupVersionKind) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		object := &metav1.PartialObjectMetadata{}
		object.SetGroupVersionKind(gvk)
		object.SetName(namespacedName.Name)
		object.SetNamespace(namespacedName.Namespace)
		if err := kubeClient.Get(ctx, namespacedName, object); err != nil {
			return err
		}
		patch := client.MergeFrom(object.DeepCopy())

		// Add a timestamp annotation to trigger a reconciliation.
		ts := time.Now().Format(time.RFC3339Nano)
		annotations := object.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string, 1)
		}
		annotations[meta.ReconcileRequestAnnotation] = ts

		// HelmRelease specific annotations to force or reset a release.
		if gvk.Kind == helmv2.HelmReleaseKind {
			if rhrArgs.syncForce {
				annotations[helmv2.ForceRequestAnnotation] = ts
			}
			if rhrArgs.syncReset {
				annotations[helmv2.ResetRequestAnnotation] = ts
			}
		}

		object.SetAnnotations(annotations)
		return kubeClient.Patch(ctx, object, patch)
	})
}
