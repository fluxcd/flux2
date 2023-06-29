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
	"github.com/spf13/cobra"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var suspendSourceHelmChartCmd = &cobra.Command{
	Use:   "chart [name]",
	Short: "Suspend reconciliation of a HelmChart",
	Long:  `The suspend command disables the reconciliation of a HelmChart resource.`,
	Example: `  # Suspend reconciliation for an existing HelmChart
  flux suspend source chart podinfo

  # Suspend reconciliation for multiple HelmCharts
  flux suspend source chart podinfo-1 podinfo-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.HelmChartKind)),
	RunE: suspendCommand{
		apiType: helmChartType,
		object:  helmChartAdapter{&sourcev1.HelmChart{}},
		list:    helmChartListAdapter{&sourcev1.HelmChartList{}},
	}.run,
}

func init() {
	suspendSourceCmd.AddCommand(suspendSourceHelmChartCmd)
}

func (obj helmChartAdapter) isSuspended() bool {
	return obj.HelmChart.Spec.Suspend
}

func (obj helmChartAdapter) setSuspended() {
	obj.HelmChart.Spec.Suspend = true
}

func (a helmChartListAdapter) item(i int) suspendable {
	return &helmChartAdapter{&a.HelmChartList.Items[i]}
}
