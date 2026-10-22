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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export resources in YAML format",
	Long:  `The export sub-commands export resources in YAML format.`,
	Example: `  # Export a Kustomization
  flux export kustomization podinfo > podinfo.yaml

  # Export a Kustomization using the kubectl-style TYPE/NAME form
  flux export ks/podinfo > podinfo.yaml

  # Export several Kustomizations
  flux export kustomization frontend backend > kustomizations.yaml

  # Export a Git repository by its kind, short name or fully qualified type
  flux export gitrepository/podinfo
  flux export gitrepo/podinfo
  flux export gitrepositories.source.toolkit.fluxcd.io podinfo

  # Export all OCI repositories using a qualified type
  flux export ocirepositories.v1.source.toolkit.fluxcd.io --all`,
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
		obj := export.object.asClientObject()
		return exportNames(kubeClient, obj, uniqueNames(args), func(name string) error {
			namespacedName := types.NamespacedName{
				Namespace: *kubeconfigArgs.Namespace,
				Name:      name,
			}
			resetObject(obj)
			if err := kubeClient.Get(ctx, namespacedName, obj); err != nil {
				return err
			}
			return printExport(export.object.export())
		})
	}
	return nil
}

// exportNames calls exportName for each name in order. With one name
// it returns the error from exportName as is. With several it reports
// each failure, continues with the next name and returns an error if
// any name failed.
func exportNames(kubeClient client.Client, obj client.Object, names []string, exportName func(name string) error) error {
	if len(names) == 1 {
		return exportName(names[0])
	}

	gvk, err := kubeClient.GroupVersionKindFor(obj)
	if err != nil {
		return err
	}

	failed := 0
	for _, name := range names {
		err := exportName(name)
		if err == nil {
			continue
		}
		failed++
		// Only an error from getting the object itself is a StatusError;
		// errors from fetching related objects are wrapped.
		if _, ok := err.(*apierrors.StatusError); ok && apierrors.IsNotFound(err) {
			logger.Failuref("%s %s not found in %s namespace", gvk.Kind, name, *kubeconfigArgs.Namespace)
		} else {
			logger.Failuref("failed to export %s %s in %s namespace: %s", gvk.Kind, name, *kubeconfigArgs.Namespace, err.Error())
		}
	}
	if failed > 0 {
		return fmt.Errorf("failed to export %d of %d %s objects", failed, len(names), gvk.Kind)
	}
	return nil
}

func printExport(export any) error {
	data, err := yaml.Marshal(export)
	if err != nil {
		return err
	}
	printlnStdout("---")
	printlnStdout(resourceToString(data))
	return nil
}

func resourceToString(data []byte) string {
	data = bytes.Replace(data, []byte("  creationTimestamp: null\n"), []byte(""), 1)
	data = bytes.Replace(data, []byte("status: {}\n"), []byte(""), 1)
	data = bytes.TrimSpace(data)
	return string(data)
}
