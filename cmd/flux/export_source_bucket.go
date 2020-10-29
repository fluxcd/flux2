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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/internal/utils"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var exportSourceBucketCmd = &cobra.Command{
	Use:   "bucket [name]",
	Short: "Export Bucket sources in YAML format",
	Long:  "The export source git command exports on or all Bucket sources in YAML format.",
	Example: `  # Export all Bucket sources
  flux export source bucket --all > sources.yaml

  # Export a Bucket source including the static credentials
  flux export source bucket my-bucket --with-credentials > source.yaml
`,
	RunE: exportSourceBucketCmdRun,
}

func init() {
	exportSourceCmd.AddCommand(exportSourceBucketCmd)
}

func exportSourceBucketCmdRun(cmd *cobra.Command, args []string) error {
	if !exportAll && len(args) < 1 {
		return fmt.Errorf("name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig)
	if err != nil {
		return err
	}

	if exportAll {
		var list sourcev1.BucketList
		err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
		if err != nil {
			return err
		}

		if len(list.Items) == 0 {
			logger.Failuref("no source found in %s namespace", namespace)
			return nil
		}

		for _, bucket := range list.Items {
			if err := exportBucket(bucket); err != nil {
				return err
			}
			if exportSourceWithCred {
				if err := exportBucketCredentials(ctx, kubeClient, bucket); err != nil {
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
		var bucket sourcev1.Bucket
		err = kubeClient.Get(ctx, namespacedName, &bucket)
		if err != nil {
			return err
		}
		if err := exportBucket(bucket); err != nil {
			return err
		}
		if exportSourceWithCred {
			return exportBucketCredentials(ctx, kubeClient, bucket)
		}
	}
	return nil
}

func exportBucket(source sourcev1.Bucket) error {
	gvk := sourcev1.GroupVersion.WithKind(sourcev1.BucketKind)
	export := sourcev1.Bucket{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        source.Name,
			Namespace:   source.Namespace,
			Labels:      source.Labels,
			Annotations: source.Annotations,
		},
		Spec: source.Spec,
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(resourceToString(data))
	return nil
}

func exportBucketCredentials(ctx context.Context, kubeClient client.Client, source sourcev1.Bucket) error {
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
		fmt.Println(resourceToString(data))
	}
	return nil
}
