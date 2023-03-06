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
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var reconcileHrCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Reconcile a HelmRelease resource",
	Long: `
The reconcile kustomization command triggers a reconciliation of a HelmRelease resource and waits for it to finish.`,
	Example: `  # Trigger a HelmRelease apply outside of the reconciliation interval
  flux reconcile hr podinfo

  # Trigger a reconciliation of the HelmRelease's source and apply changes
  flux reconcile hr podinfo --with-source`,
	ValidArgsFunction: resourceNamesCompletionFunc(helmv2.GroupVersion.WithKind(helmv2.HelmReleaseKind)),
	RunE:              reconcileSourceAndChart,
}

type reconcileHelmReleaseFlags struct {
	syncHrWithSource bool
}

var rhrArgs reconcileHelmReleaseFlags

func init() {
	reconcileHrCmd.Flags().BoolVar(&rhrArgs.syncHrWithSource, "with-source", false, "reconcile HelmRelease source")

	reconcileCmd.AddCommand(reconcileHrCmd)
}

func (obj helmReleaseAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

func (obj helmReleaseAdapter) reconcileSource() bool {
	return rhrArgs.syncHrWithSource
}

func (obj helmReleaseAdapter) getSource() (reconcileCommand, types.NamespacedName) {
	var cmd reconcileCommand
	switch obj.Spec.Chart.Spec.SourceRef.Kind {
	case sourcev1.HelmRepositoryKind:
		cmd = reconcileCommand{
			apiType: helmRepositoryType,
			object:  helmRepositoryAdapter{&sourcev1.HelmRepository{}},
		}
	case sourcev1.GitRepositoryKind:
		cmd = reconcileCommand{
			apiType: gitRepositoryType,
			object:  gitRepositoryAdapter{&sourcev1.GitRepository{}},
		}
	case sourcev1.BucketKind:
		cmd = reconcileCommand{
			apiType: bucketType,
			object:  bucketAdapter{&sourcev1.Bucket{}},
		}
	}

	return cmd, types.NamespacedName{
		Name:      obj.Spec.Chart.Spec.SourceRef.Name,
		Namespace: obj.Spec.Chart.Spec.SourceRef.Namespace,
	}
}

func (obj helmChartAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

func reconcileSourceAndChart(cmd *cobra.Command, args []string) error {
	hr := helmv2.HelmRelease{}
	reconcile := reconcileWithSourceCommand{
		apiType: helmReleaseType,
		object:  helmReleaseAdapter{&hr},
	}

	if err := reconcile.run(cmd, args); err != nil {
		return err
	}

	if reconcile.object.reconcileSource() {
		nsName := strings.Split(hr.Status.HelmChart, "/")
		if len(nsName) != 2 {
			return nil
		}

		nsCopy := *kubeconfigArgs.Namespace
		reconcileChart := reconcileCommand{
			apiType: helmChartType,
			object:  helmChartAdapter{&sourcev1.HelmChart{}},
		}

		if nsName[0] != "" {
			*kubeconfigArgs.Namespace = nsName[0]
		}

		err := reconcileChart.run(nil, []string{nsName[1]})
		*kubeconfigArgs.Namespace = nsCopy
		return err
	}

	return nil
}
