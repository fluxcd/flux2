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

var reconcileSourceHelmChartCmd = &cobra.Command{
	Use:   "chart [name]",
	Short: "Reconcile a HelmChart source",
	Long:  `The reconcile source command triggers a reconciliation of a HelmCHart resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing source
  flux reconcile source chart podinfo`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.HelmChartKind)),
	RunE: reconcileCommand{
		apiType: helmChartType,
		object:  helmChartAdapter{&sourcev1.HelmChart{}},
	}.run,
}

func init() {
	reconcileSourceCmd.AddCommand(reconcileSourceHelmChartCmd)
}

func (obj helmChartAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}
