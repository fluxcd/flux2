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
	"fmt"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/chartutil"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var debugHelmReleaseCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Debug a HelmRelease resource",
	Long: withPreviewNote(`The debug helmrelease command can be used to troubleshoot failing Helm release reconciliations.
WARNING: This command will print sensitive information if Kubernetes Secrets are referenced in the HelmRelease .spec.valuesFrom field.`),
	Example: `  # Print the status of a Helm release
  flux debug hr podinfo --show-status

  # Export the final values of a Helm release composed from referred ConfigMaps and Secrets
  flux debug hr podinfo --show-values > values.yaml`,
	RunE:              debugHelmReleaseCmdRun,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: resourceNamesCompletionFunc(helmv2.GroupVersion.WithKind(helmv2.HelmReleaseKind)),
}

type debugHelmReleaseFlags struct {
	showStatus bool
	showValues bool
}

var debugHelmReleaseArgs debugHelmReleaseFlags

func init() {
	debugHelmReleaseCmd.Flags().BoolVar(&debugHelmReleaseArgs.showStatus, "show-status", false, "print the status of the Helm release")
	debugHelmReleaseCmd.Flags().BoolVar(&debugHelmReleaseArgs.showValues, "show-values", false, "print the final values of the Helm release")
	debugCmd.AddCommand(debugHelmReleaseCmd)
}

func debugHelmReleaseCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if (!debugHelmReleaseArgs.showStatus && !debugHelmReleaseArgs.showValues) ||
		(debugHelmReleaseArgs.showStatus && debugHelmReleaseArgs.showValues) {
		return fmt.Errorf("either --show-status or --show-values must be set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	hr := &helmv2.HelmRelease{}
	hrName := types.NamespacedName{Namespace: *kubeconfigArgs.Namespace, Name: name}
	if err := kubeClient.Get(ctx, hrName, hr); err != nil {
		return err
	}

	if debugHelmReleaseArgs.showStatus {
		status, err := yaml.Marshal(hr.Status)
		if err != nil {
			return err
		}
		rootCmd.Println("# Status documentation: https://fluxcd.io/flux/components/helm/helmreleases/#helmrelease-status")
		rootCmd.Print(string(status))
		return nil
	}

	if debugHelmReleaseArgs.showValues {
		finalValues, err := chartutil.ChartValuesFromReferences(ctx,
			logr.Discard(),
			kubeClient,
			hr.GetNamespace(),
			hr.GetValues(),
			hr.Spec.ValuesFrom...)
		if err != nil {
			return err
		}

		values, err := yaml.Marshal(finalValues)
		if err != nil {
			return err
		}
		rootCmd.Print(string(values))
	}

	return nil
}
