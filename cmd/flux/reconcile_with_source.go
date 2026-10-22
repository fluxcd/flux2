package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/fluxcd/pkg/apis/meta"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

type sourceReference struct {
	kind      string
	name      string
	namespace string
}

type reconcileWithSource interface {
	adapter
	reconcilable
	reconcileSource() bool
	getSource() (reconcileSource, sourceReference)
}

type reconcileSource interface {
	reconcileAsSource(name string, sources sourceResults) error
}

// sourceResults records the result of each source reconciliation within
// one command invocation, so that a source shared by several objects,
// directly or through a HelmChart, is reconciled once.
type sourceResults map[sourceReference]error

// reconcileAsSource reconciles name as the source of another object,
// returning a get error unwrapped so it is not reported as the parent's.
func (reconcile reconcileCommand) reconcileAsSource(name string, _ sourceResults) error {
	return unwrapGetError(reconcile.reconcile(name))
}

func (reconcile reconcileWithSourceCommand) reconcileAsSource(name string, sources sourceResults) error {
	return unwrapGetError(reconcile.reconcile(name, sources))
}

type reconcileWithSourceCommand struct {
	apiType
	object reconcileWithSource
	force  bool
}

func (reconcile reconcileWithSourceCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", reconcile.kind)
	}

	sources := make(sourceResults)
	return reconcileNames(reconcile.kind, args, func(name string) error {
		return reconcile.reconcile(name, sources)
	})
}

func (reconcile reconcileWithSourceCommand) reconcile(name string, sources sourceResults) error {
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

	if reconcile.object.isSuspended() {
		return fmt.Errorf("resource is suspended")
	}

	if reconcile.object.reconcileSource() || reconcile.force {
		reconcileCmd, srcRef := reconcile.object.getSource()
		if reconcileCmd == nil {
			return fmt.Errorf("cannot reconcile source of kind %s", srcRef.kind)
		}

		nsCopy := *kubeconfigArgs.Namespace
		if srcRef.namespace == "" {
			srcRef.namespace = nsCopy
		}

		err, done := sources[srcRef]
		if !done {
			*kubeconfigArgs.Namespace = srcRef.namespace
			err = reconcileCmd.reconcileAsSource(srcRef.name, sources)
			*kubeconfigArgs.Namespace = nsCopy
			sources[srcRef] = err
		}
		if err != nil {
			return err
		}
	}

	lastHandledReconcileAt := reconcile.object.lastHandledReconcileRequest()
	logger.Actionf("annotating %s %s in %s namespace", reconcile.kind, name, *kubeconfigArgs.Namespace)
	if err := requestReconciliation(ctx, kubeClient, namespacedName,
		reconcile.groupVersion.WithKind(reconcile.kind)); err != nil {
		return err
	}
	logger.Successf("%s annotated", reconcile.kind)

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
		return fmt.Errorf("%s reconciliation failed: %s", reconcile.kind, readyCond.Message)
	}
	logger.Successf("%s", reconcile.object.successMessage())
	return nil
}
