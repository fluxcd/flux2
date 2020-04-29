package main

import (
	"context"
	"fmt"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

var deleteKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Delete kustomization",
	RunE:    deleteKsCmdRun,
}

func init() {
	deleteCmd.AddCommand(deleteKsCmd)
}

func deleteKsCmdRun(cmd *cobra.Command, args []string) error {
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

	if !deleteSilent {
		warning := "This action will remove the Kubernetes objects previously applied by this kustomization. "
		if kustomization.Spec.Suspend {
			warning = ""
		}
		prompt := promptui.Prompt{
			Label: fmt.Sprintf(
				"%sAre you sure you want to delete the %s kustomization from the %s namespace",
				warning, name, namespace,
			),
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	logAction("deleting kustomization %s in %s namespace", name, namespace)
	err = kubeClient.Delete(ctx, &kustomization)
	if err != nil {
		return err
	}
	logSuccess("kustomization deleted")

	return nil
}
