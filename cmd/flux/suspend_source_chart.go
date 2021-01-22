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
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

var suspendSourceHelmChartCmd = &cobra.Command{
	Use:   "chart [name]",
	Short: "Suspend reconciliation of a HelmChart",
	Long:  "The suspend command disables the reconciliation of a HelmChart resource.",
	Example: `  # Suspend reconciliation for an existing HelmChart
  flux suspend source chart podinfo
`,
	RunE: suspendSourceHelmChartCmdRun,
}

func init() {
	suspendSourceCmd.AddCommand(suspendSourceHelmChartCmd)
}

func suspendSourceHelmChartCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("source name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: rootArgs.namespace,
		Name:      name,
	}
	var chart sourcev1.HelmChart
	err = kubeClient.Get(ctx, namespacedName, &chart)
	if err != nil {
		return err
	}

	logger.Actionf("suspending source %s in %s namespace", name, rootArgs.namespace)
	chart.Spec.Suspend = true
	if err := kubeClient.Update(ctx, &chart); err != nil {
		return err
	}
	logger.Successf("source suspended")

	return nil
}
