package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"
)

type reconcileWithSource interface {
	adapter
	reconcilable
	reconcileSource() bool
	getSource() (reconcileCommand, types.NamespacedName)
}

type reconcileWithSourceCommand struct {
	apiType
	object reconcileWithSource
}

func (reconcile reconcileWithSourceCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", reconcile.kind)
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: *kubeconfigArgs.Namespace,
		Name:      name,
	}

	err = kubeClient.Get(ctx, namespacedName, reconcile.object.asClientObject())
	if err != nil {
		return err
	}

	if reconcile.object.isSuspended() {
		return fmt.Errorf("resource is suspended")
	}

	if reconcile.object.reconcileSource() {
		reconcileCmd, nsName := reconcile.object.getSource()
		nsCopy := *kubeconfigArgs.Namespace
		if nsName.Namespace != "" {
			*kubeconfigArgs.Namespace = nsName.Namespace
		}

		err := reconcileCmd.run(nil, []string{nsName.Name})
		if err != nil {
			return err
		}
		*kubeconfigArgs.Namespace = nsCopy
	}

	lastHandledReconcileAt := reconcile.object.lastHandledReconcileRequest()
	logger.Actionf("annotating %s %s in %s namespace", reconcile.kind, name, *kubeconfigArgs.Namespace)
	if err := requestReconciliation(ctx, kubeClient, namespacedName,
		reconcile.groupVersion.WithKind(reconcile.kind)); err != nil {
		return err
	}
	logger.Successf("%s annotated", reconcile.kind)

	logger.Waitingf("waiting for %s reconciliation", reconcile.kind)
	if err := wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
		reconciliationHandled(ctx, kubeClient, namespacedName, reconcile.object, lastHandledReconcileAt)); err != nil {
		return err
	}

	readyCond := apimeta.FindStatusCondition(reconcilableConditions(reconcile.object), meta.ReadyCondition)
	if readyCond == nil {
		return fmt.Errorf("status can't be determined")
	}

	if readyCond.Status != metav1.ConditionTrue {
		return fmt.Errorf("%s reconciliation failed: %s", reconcile.kind, readyCond.Message)
	}
	logger.Successf(reconcile.object.successMessage())
	return nil
}
