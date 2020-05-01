package main

import (
	"context"
	"fmt"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var resumeKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Resume kustomization",
	Long:    "The resume command marks a previously suspended Kustomization resource for reconciliation and waits for it to finish the apply.",
	RunE:    resumeKsCmdRun,
}

func init() {
	resumeCmd.AddCommand(resumeKsCmd)
}

func resumeKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("kustomization name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	var kustomization kustomizev1.Kustomization
	err = kubeClient.Get(ctx, namespacedName, &kustomization)
	if err != nil {
		return err
	}

	logAction("resuming kustomization %s in %s namespace", name, namespace)
	kustomization.Spec.Suspend = false
	if err := kubeClient.Update(ctx, &kustomization); err != nil {
		return err
	}
	logSuccess("kustomization resumed")

	logWaiting("waiting for kustomization sync")
	if err := wait.PollImmediate(pollInterval, timeout,
		isKustomizationReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logSuccess("kustomization sync completed")

	err = kubeClient.Get(ctx, namespacedName, &kustomization)
	if err != nil {
		return err
	}

	if kustomization.Status.LastAppliedRevision != "" {
		logSuccess("applied revision %s", kustomization.Status.LastAppliedRevision)
	} else {
		return fmt.Errorf("kustomization sync failed")
	}

	return nil
}
