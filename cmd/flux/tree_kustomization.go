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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fluxcd/flux2/internal/tree"
	"github.com/fluxcd/flux2/internal/utils"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var treeKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks", "kustomization"},
	Short:   "Print the resource inventory of a Kustomization",
	Long:    `The tree command prints the resource list reconciled by a Kustomization.'`,
	Example: `  # Print the resources managed by the root Kustomization
  flux tree kustomization flux-system

  # Print the Flux resources managed by the root Kustomization
  flux tree kustomization flux-system --compact`,
	RunE: treeKsCmdRun,
}

type TreeKsFlags struct {
	compact bool
	output  string
}

var treeKsArgs TreeKsFlags

func init() {
	treeKsCmd.Flags().BoolVar(&treeKsArgs.compact, "compact", false, "list Flux resources only.")
	treeKsCmd.Flags().StringVarP(&treeKsArgs.output, "output", "o", "",
		"the format in which the tree should be printed. can be 'json' or 'yaml'")
	treeCmd.AddCommand(treeKsCmd)
}

func treeKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("kustomization name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	k := &kustomizev1.Kustomization{}
	err = kubeClient.Get(ctx, client.ObjectKey{
		Namespace: rootArgs.namespace,
		Name:      name,
	}, k)
	if err != nil {
		return err
	}

	kMeta, err := object.CreateObjMetadata(k.Namespace, k.Name,
		schema.GroupKind{Group: kustomizev1.GroupVersion.Group, Kind: kustomizev1.KustomizationKind})
	if err != nil {
		return err
	}

	kTree := tree.New(kMeta)
	err = treeKustomization(ctx, kTree, k, kubeClient, treeKsArgs.compact)
	if err != nil {
		return err
	}

	switch treeKsArgs.output {
	case "json":
		data, err := json.MarshalIndent(kTree, "", "  ")
		if err != nil {
			return err
		}
		rootCmd.Println(string(data))
	case "yaml":
		data, err := yaml.Marshal(kTree)
		if err != nil {
			return err
		}
		rootCmd.Println(string(data))
	default:
		rootCmd.Println(kTree.Print())
	}

	return nil
}

func treeKustomization(ctx context.Context, tree tree.ObjMetadataTree, item *kustomizev1.Kustomization, kubeClient client.Client, compact bool) error {
	if item.Status.Inventory == nil || len(item.Status.Inventory.Entries) == 0 {
		return nil
	}

	for _, entry := range item.Status.Inventory.Entries {
		objMetadata, err := object.ParseObjMetadata(entry.ID)
		if err != nil {
			return err
		}

		if compact && !strings.Contains(objMetadata.GroupKind.Group, "toolkit.fluxcd.io") {
			continue
		}

		if objMetadata.GroupKind.Group == kustomizev1.GroupVersion.Group &&
			objMetadata.GroupKind.Kind == kustomizev1.KustomizationKind &&
			objMetadata.Namespace == item.Namespace &&
			objMetadata.Name == item.Name {
			continue
		}

		ks := tree.Add(objMetadata)
		if objMetadata.GroupKind.Group == kustomizev1.GroupVersion.Group &&
			objMetadata.GroupKind.Kind == kustomizev1.KustomizationKind {
			k := &kustomizev1.Kustomization{}
			err = kubeClient.Get(ctx, client.ObjectKey{
				Namespace: objMetadata.Namespace,
				Name:      objMetadata.Name,
			}, k)
			if err != nil {
				return fmt.Errorf("failed to find object: %w", err)
			}
			err := treeKustomization(ctx, ks, k, kubeClient, compact)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
