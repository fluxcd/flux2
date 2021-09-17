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
	"strings"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generates completion scripts for various shells",
	Long:  "The completion sub-command generates completion scripts for various shells",
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

func contextsCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	rawConfig, err := utils.ClientConfig(rootArgs.kubeconfig, rootArgs.kubecontext).RawConfig()
	if err != nil {
		return completionError(err)
	}

	var comps []string

	for name := range rawConfig.Contexts {
		if strings.HasPrefix(name, toComplete) {
			comps = append(comps, name)
		}
	}

	return comps, cobra.ShellCompDirectiveNoFileComp
}

func resourceNamesCompletionFunc(gvk schema.GroupVersionKind) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
		defer cancel()

		cfg, err := utils.KubeConfig(rootArgs.kubeconfig, rootArgs.kubecontext)
		if err != nil {
			return completionError(err)
		}

		dc, err := discovery.NewDiscoveryClientForConfig(cfg)
		if err != nil {
			return completionError(err)
		}
		mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return completionError(err)
		}

		client, err := dynamic.NewForConfig(cfg)
		if err != nil {
			return completionError(err)
		}

		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			dr = client.Resource(mapping.Resource).Namespace(rootArgs.namespace)
		} else {
			dr = client.Resource(mapping.Resource)
		}

		list, err := dr.List(ctx, metav1.ListOptions{})
		if err != nil {
			return completionError(err)
		}

		var comps []string

		for _, item := range list.Items {
			name := item.GetName()

			if strings.HasPrefix(name, toComplete) {
				comps = append(comps, name)
			}
		}

		return comps, cobra.ShellCompDirectiveNoFileComp
	}
}

func completionError(err error) ([]string, cobra.ShellCompDirective) {
	cobra.CompError(err.Error())
	return nil, cobra.ShellCompDirectiveError
}
