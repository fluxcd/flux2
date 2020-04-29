package main

import (
	"context"
	"fmt"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var exportKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Export kustomization in YAML format",
	RunE:    exportKsCmdRun,
}

func init() {
	exportCmd.AddCommand(exportKsCmd)
}

func exportKsCmdRun(cmd *cobra.Command, args []string) error {
	if !exportAll && len(args) < 1 {
		return fmt.Errorf("kustomization name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	if exportAll {
		var list kustomizev1.KustomizationList
		err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
		if err != nil {
			return err
		}

		if len(list.Items) == 0 {
			logFailure("no kustomizations found in %s namespace", namespace)
			return nil
		}

		for _, kustomization := range list.Items {
			if err := exportKs(kustomization); err != nil {
				return err
			}
		}
	} else {
		name := args[0]
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		var kustomization kustomizev1.Kustomization
		err = kubeClient.Get(ctx, namespacedName, &kustomization)
		if err != nil {
			return err
		}
		return exportKs(kustomization)
	}
	return nil
}

func exportKs(kustomization kustomizev1.Kustomization) error {
	gvk := kustomizev1.GroupVersion.WithKind("Kustomization")
	export := kustomizev1.Kustomization{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kustomization.Name,
			Namespace: kustomization.Namespace,
		},
		Spec: kustomization.Spec,
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(string(data))
	return nil
}
