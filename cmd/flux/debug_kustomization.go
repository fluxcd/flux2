/*
Copyright 2024 The Flux authors

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
	"errors"
	"fmt"
	"sort"
	"strings"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/kustomize"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var debugKustomizationCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Debug a Flux Kustomization resource",
	Long: withPreviewNote(`The debug kustomization command can be used to troubleshoot failing Flux Kustomization reconciliations.
WARNING: This command will print sensitive information if Kubernetes Secrets are referenced in the Kustomization .spec.postBuild.substituteFrom field.`),
	Example: `  # Print the status of a Flux Kustomization
  flux debug ks podinfo --show-status

  # Export the final variables used for post-build substitutions composed from referred ConfigMaps and Secrets
  flux debug ks podinfo --show-vars > vars.env`,
	RunE:              debugKustomizationCmdRun,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
}

type debugKustomizationFlags struct {
	showStatus bool
	showVars   bool
}

var debugKustomizationArgs debugKustomizationFlags

func init() {
	debugKustomizationCmd.Flags().BoolVar(&debugKustomizationArgs.showStatus, "show-status", false, "print the status of the Flux Kustomization")
	debugKustomizationCmd.Flags().BoolVar(&debugKustomizationArgs.showVars, "show-vars", false, "print the final vars of the Flux Kustomization in dot env format")
	debugCmd.AddCommand(debugKustomizationCmd)
}

func debugKustomizationCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if (!debugKustomizationArgs.showStatus && !debugKustomizationArgs.showVars) ||
		(debugKustomizationArgs.showStatus && debugKustomizationArgs.showVars) {
		return fmt.Errorf("either --show-status or --show-vars must be set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	ks := &kustomizev1.Kustomization{}
	ksName := types.NamespacedName{Namespace: *kubeconfigArgs.Namespace, Name: name}
	if err := kubeClient.Get(ctx, ksName, ks); err != nil {
		return err
	}

	if debugKustomizationArgs.showStatus {
		status, err := yaml.Marshal(ks.Status)
		if err != nil {
			return err
		}
		rootCmd.Println("# Status documentation: https://fluxcd.io/flux/components/kustomize/kustomizations/#kustomization-status")
		rootCmd.Print(string(status))
		return nil
	}

	if debugKustomizationArgs.showVars {
		if ks.Spec.PostBuild == nil {
			return errors.New("no post build substitutions found")
		}

		ksObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ks)
		if err != nil {
			return err
		}

		finalVars, err := kustomize.LoadVariables(ctx, kubeClient, unstructured.Unstructured{Object: ksObj})
		if err != nil {
			return err
		}

		if len(ks.Spec.PostBuild.Substitute) > 0 {
			for k, v := range ks.Spec.PostBuild.Substitute {
				// Remove new lines from the values as they are not supported.
				// Replicates the controller behavior from
				// https://github.com/fluxcd/pkg/blob/main/kustomize/kustomize_varsub.go
				finalVars[k] = strings.ReplaceAll(v, "\n", "")
			}
		}

		keys := make([]string, 0, len(finalVars))
		for k := range finalVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			rootCmd.Println(k + "=" + finalVars[k])
		}
	}

	return nil
}
