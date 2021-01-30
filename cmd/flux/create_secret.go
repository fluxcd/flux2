/*
Copyright 2020 The Flux authors

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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var createSecretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Create or update Kubernetes secrets",
	Long:  "The create source sub-commands generate Kubernetes secrets specific to Flux.",
}

func init() {
	createCmd.AddCommand(createSecretCmd)
}

func makeSecret(name string) (corev1.Secret, error) {
	secretLabels, err := parseLabels()
	if err != nil {
		return corev1.Secret{}, err
	}

	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: rootArgs.namespace,
			Labels:    secretLabels,
		},
		StringData: map[string]string{},
		Data:       nil,
	}, nil
}

func upsertSecret(ctx context.Context, kubeClient client.Client, secret corev1.Secret) error {
	namespacedName := types.NamespacedName{
		Namespace: secret.GetNamespace(),
		Name:      secret.GetName(),
	}

	var existing corev1.Secret
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, &secret); err != nil {
				return err
			} else {
				return nil
			}
		}
		return err
	}

	existing.StringData = secret.StringData
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return err
	}
	return nil
}

func exportSecret(secret corev1.Secret) error {
	secret.TypeMeta = metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Secret",
	}

	data, err := yaml.Marshal(secret)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(resourceToString(data))
	return nil
}
