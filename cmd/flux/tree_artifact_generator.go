/*
Copyright 2025 The Flux authors

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

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/cli-utils/pkg/object"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	swapi "github.com/fluxcd/source-watcher/api/v2/v1beta1"

	"github.com/fluxcd/flux2/v2/internal/tree"
	"github.com/fluxcd/flux2/v2/internal/utils"
)

var treeArtifactGeneratorCmd = &cobra.Command{
	Use:   "generator [name]",
	Short: "Print the inventory of an ArtifactGenerator",
	Long:  withPreviewNote(`The tree command prints the ExternalArtifact list managed by an ArtifactGenerator.'`),
	Example: `  # Print the ExternalArtifacts managed by an ArtifactGenerator
  flux tree artifact generator my-generator`,
	RunE:              treeArtifactGeneratorCmdRun,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: resourceNamesCompletionFunc(swapi.GroupVersion.WithKind(swapi.ArtifactGeneratorKind)),
}

type TreeArtifactGeneratorFlags struct {
	output string
}

var treeArtifactGeneratorArgs TreeArtifactGeneratorFlags

func init() {
	treeArtifactGeneratorCmd.Flags().StringVarP(&treeArtifactGeneratorArgs.output, "output", "o", "",
		"the format in which the tree should be printed. can be 'json' or 'yaml'")
	treeArtifactCmd.AddCommand(treeArtifactGeneratorCmd)
}

func treeArtifactGeneratorCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("generator name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	ag := &swapi.ArtifactGenerator{}
	err = kubeClient.Get(ctx, client.ObjectKey{
		Namespace: *kubeconfigArgs.Namespace,
		Name:      name,
	}, ag)
	if err != nil {
		return err
	}

	kTree := tree.New(object.ObjMetadata{
		Namespace: ag.Namespace,
		Name:      ag.Name,
		GroupKind: schema.GroupKind{Group: swapi.GroupVersion.Group, Kind: swapi.ArtifactGeneratorKind},
	})

	for _, ea := range ag.Status.Inventory {
		kTree.Add(object.ObjMetadata{
			Namespace: ea.Namespace,
			Name:      ea.Name,
			GroupKind: schema.GroupKind{Group: sourcev1.GroupVersion.Group, Kind: sourcev1.ExternalArtifactKind},
		})
	}

	switch treeArtifactGeneratorArgs.output {
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
