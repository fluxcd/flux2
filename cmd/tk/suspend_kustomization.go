package main

import (
	"context"
	"fmt"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

var suspendKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Suspend kustomization",
	Long:    "The suspend command disables the reconciliation of a Kustomization resource.",
	RunE:    suspendKsCmdRun,
}

func init() {
	suspendCmd.AddCommand(suspendKsCmd)
}

func suspendKsCmdRun(cmd *cobra.Command, args []string) error {
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

	logAction("suspending kustomization %s in %s namespace", name, namespace)
	kustomization.Spec.Suspend = true
	if err := kubeClient.Update(ctx, &kustomization); err != nil {
		return err
	}
	logSuccess("kustomization suspended")

	return nil
}
