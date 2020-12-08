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
	"fmt"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/internal/utils"
)

var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Reconcile sources and resources",
	Long:  "The reconcile sub-commands trigger a reconciliation of sources and resources.",
}

func init() {
	rootCmd.AddCommand(reconcileCmd)
}

type reconcileCommand struct {
	humanKind string
	adapter   reconcilable
}

type reconcilable interface {
	adapter     // to be able to load from the cluster
	suspendable // to tell if it's suspended

	// these are implemented by anything embedding metav1.ObjectMeta
	GetAnnotations() map[string]string
	SetAnnotations(map[string]string)

	// this is usually implemented by GOTK types, since it's used for meta.SetResourceCondition
	GetStatusConditions() *[]metav1.Condition

	lastHandledReconcileRequest() string // what was the last handled reconcile request?
	successMessage() string              // what do you want to tell people when successfully reconciled?
}

func (reconcile reconcileCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", reconcile.humanKind)
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	err = kubeClient.Get(ctx, namespacedName, reconcile.adapter.asRuntimeObject())
	if err != nil {
		return err
	}

	if reconcile.adapter.isSuspended() {
		return fmt.Errorf("resource is suspended")
	}

	logger.Actionf("annotating %s %s in %s namespace", reconcile.humanKind, name, namespace)
	if err := requestReconciliation(ctx, kubeClient, namespacedName, reconcile.adapter); err != nil {
		return err
	}
	logger.Successf("%s annotated", reconcile.humanKind)

	lastHandledReconcileAt := reconcile.adapter.lastHandledReconcileRequest()
	logger.Waitingf("waiting for %s reconciliation", reconcile.humanKind)
	if err := wait.PollImmediate(pollInterval, timeout,
		reconciliationHandled(ctx, kubeClient, namespacedName, reconcile.adapter, lastHandledReconcileAt)); err != nil {
		return err
	}
	logger.Successf("%s reconciliation completed", reconcile.humanKind)

	if apimeta.IsStatusConditionFalse(*reconcile.adapter.GetStatusConditions(), meta.ReadyCondition) {
		return fmt.Errorf("%s reconciliation failed", reconcile.humanKind)
	}
	logger.Successf(reconcile.adapter.successMessage())
	return nil
}

func reconciliationHandled(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, obj reconcilable, lastHandledReconcileAt string) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, obj.asRuntimeObject())
		if err != nil {
			return false, err
		}
		return obj.lastHandledReconcileRequest() != lastHandledReconcileAt, nil
	}
}

func requestReconciliation(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, obj reconcilable) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		if err := kubeClient.Get(ctx, namespacedName, obj.asRuntimeObject()); err != nil {
			return err
		}
		if ann := obj.GetAnnotations(); ann == nil {
			obj.SetAnnotations(map[string]string{
				meta.ReconcileAtAnnotation: time.Now().Format(time.RFC3339Nano),
			})
		} else {
			ann[meta.ReconcileAtAnnotation] = time.Now().Format(time.RFC3339Nano)
			obj.SetAnnotations(ann)
		}
		return kubeClient.Update(ctx, obj.asRuntimeObject())
	})
}
