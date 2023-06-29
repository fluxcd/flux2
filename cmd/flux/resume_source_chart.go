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
	"fmt"

	"github.com/spf13/cobra"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var resumeSourceHelmChartCmd = &cobra.Command{
	Use:   "chart [name]",
	Short: "Resume a suspended HelmChart",
	Long:  `The resume command marks a previously suspended HelmChart resource for reconciliation and waits for it to finish.`,
	Example: `  # Resume reconciliation for an existing HelmChart
  flux resume source chart podinfo

  # Resume reconciliation for multiple HelmCharts
  flux resume source chart podinfo-1 podinfo-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.HelmChartKind)),
	RunE: resumeCommand{
		apiType: helmChartType,
		list:    &helmChartListAdapter{&sourcev1.HelmChartList{}},
	}.run,
}

func init() {
	resumeSourceCmd.AddCommand(resumeSourceHelmChartCmd)
}

func (obj helmChartAdapter) getObservedGeneration() int64 {
	return obj.HelmChart.Status.ObservedGeneration
}

func (obj helmChartAdapter) setUnsuspended() {
	obj.HelmChart.Spec.Suspend = false
}

func (obj helmChartAdapter) successMessage() string {
	return fmt.Sprintf("fetched revision %s", obj.Status.Artifact.Revision)
}

func (a helmChartListAdapter) resumeItem(i int) resumable {
	return &helmChartAdapter{&a.HelmChartList.Items[i]}
}
