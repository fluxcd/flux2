package main

import (
	"context"
	"fmt"
	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

type reconcileWithSource interface {
	adapter
	reconcilable
	reconcileSource() bool
	getSource() (reconcileCommand, string)
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

	if reconcile.object.reconcileSource() {
		nsCopy := rootArgs.namespace
		objectNs := reconcile.object.asClientObject().GetNamespace()
		if objectNs != "" {
			rootArgs.namespace = reconcile.object.asClientObject().GetNamespace()
		}

		reconcileCmd, sourceName := reconcile.object.getSource()
		err := reconcileCmd.run(nil, []string{sourceName})
		if err != nil {
			return err
		}
		rootArgs.namespace = nsCopy
	}

	logger.Actionf("annotating %s %s in %s namespace", reconcile.kind, name, rootArgs.namespace)
	if err := requestReconciliation(ctx, kubeClient, namespacedName, reconcile.object); err != nil {
		return err
	}
	logger.Successf("%s annotated", reconcile.kind)

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
