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
	"k8s.io/apimachinery/pkg/types"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"
)

var reconcileSourceHelmChartCmd = &cobra.Command{
	Use:   "chart [name]",
	Short: "Reconcile a HelmChart source",
	Long:  `The reconcile source command triggers a reconciliation of a HelmChart resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing source
  flux reconcile source chart podinfo
 
  # Trigger a reconciliation of the HelmCharts's source and apply changes
  flux reconcile helmchart podinfo --with-source`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1b2.GroupVersion.WithKind(sourcev1b2.HelmChartKind)),
	RunE: reconcileWithSourceCommand{
		apiType: helmChartType,
		object:  helmChartAdapter{&sourcev1b2.HelmChart{}},
	}.run,
}

func init() {
	reconcileSourceHelmChartCmd.Flags().BoolVar(&rhcArgs.syncHrWithSource, "with-source", false, "reconcile HelmChart source")
	reconcileSourceCmd.AddCommand(reconcileSourceHelmChartCmd)
}

func (obj helmChartAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

type reconcileHelmChartFlags struct {
	syncHrWithSource bool
}

var rhcArgs reconcileHelmChartFlags

func (obj helmChartAdapter) reconcileSource() bool {
	return rhcArgs.syncHrWithSource
}

func (obj helmChartAdapter) getSource() (reconcileSource, types.NamespacedName) {
	var cmd reconcileCommand
	switch obj.Spec.SourceRef.Kind {
	case sourcev1b2.HelmRepositoryKind:
		cmd = reconcileCommand{
			apiType: helmRepositoryType,
			object:  helmRepositoryAdapter{&sourcev1b2.HelmRepository{}},
		}
	case sourcev1.GitRepositoryKind:
		cmd = reconcileCommand{
			apiType: gitRepositoryType,
			object:  gitRepositoryAdapter{&sourcev1.GitRepository{}},
		}
	case sourcev1b2.BucketKind:
		cmd = reconcileCommand{
			apiType: bucketType,
			object:  bucketAdapter{&sourcev1b2.Bucket{}},
		}
	}

	return cmd, types.NamespacedName{
		Name:      obj.Spec.SourceRef.Name,
		Namespace: obj.Namespace,
	}
}

func (obj helmChartAdapter) isStatic() bool {
	return false
}
