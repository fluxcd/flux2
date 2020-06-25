/*
Copyright 2020 The Flux CD contributors.

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

	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var exportSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Export GitRepository sources in YAML format",
	Long:  "The export source git command exports on or all GitRepository sources in YAML format.",
	Example: `  # Export all GitRepository sources
  export source git --all > sources.yaml

  # Export a GitRepository source including the SSH key pair or basic auth credentials
  export source git my-private-repo --with-credentials > source.yaml
`,
	RunE: exportSourceGitCmdRun,
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
			logger.Failuref("no source found in %s namespace", namespace)
			return nil
		}

		for _, repository := range list.Items {
			if err := exportGit(repository); err != nil {
				return err
			}
			if exportSourceWithCred {
				if err := exportGitCredentials(ctx, kubeClient, repository); err != nil {
					return err
				}
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
		if err := exportGit(repository); err != nil {
			return err
		}
		if exportSourceWithCred {
			return exportGitCredentials(ctx, kubeClient, repository)
		}
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

func exportGitCredentials(ctx context.Context, kubeClient client.Client, source sourcev1.GitRepository) error {
	if source.Spec.SecretRef != nil {
		namespacedName := types.NamespacedName{
			Namespace: source.Namespace,
			Name:      source.Spec.SecretRef.Name,
		}
		var cred corev1.Secret
		err := kubeClient.Get(ctx, namespacedName, &cred)
		if err != nil {
			return fmt.Errorf("failed to retrieve secret %s, error: %w", namespacedName.Name, err)
		}

		exported := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
			Data: cred.Data,
			Type: cred.Type,
		}

		data, err := yaml.Marshal(exported)
		if err != nil {
			return err
		}

		fmt.Println("---")
		fmt.Println(string(data))
	}
	return nil
}
