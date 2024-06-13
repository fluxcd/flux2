/*
Copyright 2021 The Flux authors

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

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

// exportableWithSecret represents a type that you can fetch from the Kubernetes
// API, get a secretRef from the spec, then tidy up for serialising.
type exportableWithSecret interface {
	adapter
	exportable
	secret() *types.NamespacedName
}

// exportableWithSecretList represents a type that has a list of values, each of
// which is exportableWithSecret.
type exportableWithSecretList interface {
	listAdapter
	exportableList
	secretItem(i int) *types.NamespacedName
}

type exportWithSecretCommand struct {
	object exportableWithSecret
	list   exportableWithSecretList
}

func (export exportWithSecretCommand) run(cmd *cobra.Command, args []string) error {
	if !exportArgs.all && len(args) < 1 {
		return fmt.Errorf("name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	if exportArgs.all {
		err = kubeClient.List(ctx, export.list.asClientList(), client.InNamespace(*kubeconfigArgs.Namespace))
		if err != nil {
			return err
		}

		if export.list.len() == 0 {
			return fmt.Errorf("no objects found in %s namespace", *kubeconfigArgs.Namespace)
		}

		for i := 0; i < export.list.len(); i++ {
			if err = printExport(export.list.exportItem(i)); err != nil {
				return err
			}

			if exportSourceWithCred {
				if export.list.secretItem(i) != nil {
					namespacedName := *export.list.secretItem(i)
					return printSecretCredentials(ctx, kubeClient, namespacedName)
				}
			}
		}
	} else {
		name := args[0]
		namespacedName := types.NamespacedName{
			Namespace: *kubeconfigArgs.Namespace,
			Name:      name,
		}
		err = kubeClient.Get(ctx, namespacedName, export.object.asClientObject())
		if err != nil {
			return err
		}

		if err := printExport(export.object.export()); err != nil {
			return err
		}

		if exportSourceWithCred {
			if export.object.secret() != nil {
				namespacedName := *export.object.secret()
				return printSecretCredentials(ctx, kubeClient, namespacedName)
			}
		}

	}
	return nil
}

func printSecretCredentials(ctx context.Context, kubeClient client.Client, nsName types.NamespacedName) error {
	var cred corev1.Secret
	err := kubeClient.Get(ctx, nsName, &cred)
	if err != nil {
		return fmt.Errorf("failed to retrieve secret %s, error: %w", nsName.Name, err)
	}

	exported := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsName.Name,
			Namespace: nsName.Namespace,
		},
		Data: cred.Data,
		Type: cred.Type,
	}
	return printExport(exported)
}
