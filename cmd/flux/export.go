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
	"bytes"
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export resources in YAML format",
	Long:  `The export sub-commands export resources in YAML format.`,
}

type exportFlags struct {
	all bool
}

var exportArgs exportFlags

func init() {
	exportCmd.PersistentFlags().BoolVar(&exportArgs.all, "all", false, "select all resources")

	rootCmd.AddCommand(exportCmd)
}

// exportable represents a type that you can fetch from the Kubernetes
// API, then tidy up for serialising.
type exportable interface {
	adapter
	export() interface{}
}

// exportableList represents a type that has a list of values, each of
// which is exportable.
type exportableList interface {
	listAdapter
	exportItem(i int) interface{}
}

type exportCommand struct {
	object exportable
	list   exportableList
}

func (export exportCommand) run(cmd *cobra.Command, args []string) error {
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
		return printExport(export.object.export())
	}
	return nil
}

func printExport(export interface{}) error {
	data, err := yaml.Marshal(export)
	if err != nil {
		return err
	}
	rootCmd.Println("---")
	rootCmd.Println(resourceToString(data))
	return nil
}

func resourceToString(data []byte) string {
	data = bytes.Replace(data, []byte("  creationTimestamp: null\n"), []byte(""), 1)
	data = bytes.Replace(data, []byte("status: {}\n"), []byte(""), 1)
	data = bytes.TrimSpace(data)
	return string(data)
}
