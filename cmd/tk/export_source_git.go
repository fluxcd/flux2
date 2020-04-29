package main

import (
	"context"
	"fmt"

	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var exportSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Export git source in YAML format",
	RunE:  exportSourceGitCmdRun,
}

func init() {
	exportSourceCmd.AddCommand(exportSourceGitCmd)
}

func exportSourceGitCmdRun(cmd *cobra.Command, args []string) error {
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
		var list sourcev1.GitRepositoryList
		err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
		if err != nil {
			return err
		}

		if len(list.Items) == 0 {
			logFailure("no source found in %s namespace", namespace)
			return nil
		}

		for _, repository := range list.Items {
			if err := exportGit(repository); err != nil {
				return err
			}
		}
	} else {
		name := args[0]
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		var repository sourcev1.GitRepository
		err = kubeClient.Get(ctx, namespacedName, &repository)
		if err != nil {
			return err
		}
		return exportGit(repository)
	}
	return nil
}

func exportGit(source sourcev1.GitRepository) error {
	gvk := sourcev1.GroupVersion.WithKind("GitRepository")
	export := sourcev1.GitRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      source.Name,
			Namespace: source.Namespace,
		},
		Spec: source.Spec,
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(string(data))
	return nil
}
