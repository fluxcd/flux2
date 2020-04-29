package main

import (
	"context"
	"fmt"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
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

	if err := syncKsCmdRun(nil, []string{name}); err != nil {
		return err
	}

	return nil
}
