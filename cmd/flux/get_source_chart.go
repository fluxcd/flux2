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
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var getSourceHelmChartCmd = &cobra.Command{
	Use:   "chart",
	Short: "Get HelmChart statuses",
	Long:  "The get sources chart command prints the status of the HelmCharts.",
	Example: `  # List all Helm charts and their status
  flux get sources chart

 # List Helm charts from all namespaces
  flux get sources chart --all-namespaces`,
	RunE: getCommand{
		apiType: helmChartType,
		list:    &helmChartListAdapter{&sourcev1.HelmChartList{}},
	}.run,
}

func init() {
	getSourceCmd.AddCommand(getSourceHelmChartCmd)
}

func (a *helmChartListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := a.Items[i]
	var revision string
	if item.GetArtifact() != nil {
		revision = item.GetArtifact().Revision
	}
	status, msg := statusAndMessage(item.Status.Conditions)
	return append(nameColumns(&item, includeNamespace, includeKind),
		status, msg, revision, strings.Title(strconv.FormatBool(item.Spec.Suspend)))
}

func (a helmChartListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Ready", "Message", "Revision", "Suspended"}
	if includeNamespace {
		headers = append([]string{"Namespace"}, headers...)
	}
	return headers
}

func (a helmChartListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := a.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
