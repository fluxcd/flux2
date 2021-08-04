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

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"

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

	// this is usually implemented by GOTK types, since it's used for meta.SetResourceCondition
	GetStatusConditions() *[]metav1.Condition

	lastHandledReconcileRequest() string // what was the last handled reconcile request?
	successMessage() string              // what do you want to tell people when successfully reconciled?
}

func (reconcile reconcileCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", reconcile.kind)
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: rootArgs.namespace,
		Name:      name,
	}

	err = kubeClient.Get(ctx, namespacedName, reconcile.object.asClientObject())
	if err != nil {
		return err
	}

	if reconcile.object.isSuspended() {
		return fmt.Errorf("resource is suspended")
	}

	logger.Actionf("annotating %s %s in %s namespace", reconcile.kind, name, rootArgs.namespace)
	if err := requestReconciliation(ctx, kubeClient, namespacedName, reconcile.object); err != nil {
		return err
	}
	logger.Successf("%s annotated", reconcile.kind)

	if reconcile.kind == v1beta1.AlertKind || reconcile.kind == v1beta1.ReceiverKind {
		if err = wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
			isReconcileReady(ctx, kubeClient, namespacedName, reconcile.object)); err != nil {
			return err
		}

		logger.Successf(reconcile.object.successMessage())
		return nil
	}

	lastHandledReconcileAt := reconcile.object.lastHandledReconcileRequest()
	logger.Waitingf("waiting for %s reconciliation", reconcile.kind)
	if err := wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
		reconciliationHandled(ctx, kubeClient, namespacedName, reconcile.object, lastHandledReconcileAt)); err != nil {
		return err
	}

	logger.Successf("%s reconciliation completed", reconcile.kind)

	if apimeta.IsStatusConditionFalse(*reconcile.object.GetStatusConditions(), meta.ReadyCondition) {
		return fmt.Errorf("%s reconciliation failed", reconcile.kind)
	}
	logger.Successf(reconcile.object.successMessage())
	return nil
}

func reconciliationHandled(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, obj reconcilable, lastHandledReconcileAt string) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, obj.asClientObject())
		if err != nil {
			return false, err
		}
		return obj.lastHandledReconcileRequest() != lastHandledReconcileAt, nil
	}
}

func requestReconciliation(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, obj reconcilable) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		if err := kubeClient.Get(ctx, namespacedName, obj.asClientObject()); err != nil {
			return err
		}
		patch := client.MergeFrom(obj.deepCopyClientObject())
		if ann := obj.GetAnnotations(); ann == nil {
			obj.SetAnnotations(map[string]string{
				meta.ReconcileRequestAnnotation: time.Now().Format(time.RFC3339Nano),
			})
		} else {
			ann[meta.ReconcileRequestAnnotation] = time.Now().Format(time.RFC3339Nano)
			obj.SetAnnotations(ann)
		}
		return kubeClient.Patch(ctx, obj.asClientObject(), patch)
	})
}

func isReconcileReady(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, obj reconcilable) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, obj.asClientObject())
		if err != nil {
			return false, err
		}

		if c := apimeta.FindStatusCondition(*obj.GetStatusConditions(), meta.ReadyCondition); c != nil {
			switch c.Status {
			case metav1.ConditionTrue:
				return true, nil
			case metav1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}
