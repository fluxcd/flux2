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
	"k8s.io/apimachinery/pkg/types"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"
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
	RunE: reconcileWithSourceCommand{
		apiType: helmReleaseType,
		object:  helmReleaseAdapter{&helmv2.HelmRelease{}},
	}.run,
}

type reconcileHelmReleaseFlags struct {
	syncHrWithSource bool
	syncForce        bool
	syncReset        bool
}

var rhrArgs reconcileHelmReleaseFlags

func init() {
	reconcileHrCmd.Flags().BoolVar(&rhrArgs.syncHrWithSource, "with-source", false, "reconcile HelmRelease source")
	reconcileHrCmd.Flags().BoolVar(&rhrArgs.syncForce, "force", false, "force a one-off install or upgrade of the HelmRelease resource")
	reconcileHrCmd.Flags().BoolVar(&rhrArgs.syncReset, "reset", false, "reset the failure count for this HelmRelease resource")
	reconcileCmd.AddCommand(reconcileHrCmd)
}

func (obj helmReleaseAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

func (obj helmReleaseAdapter) reconcileSource() bool {
	return rhrArgs.syncHrWithSource
}

func (obj helmReleaseAdapter) getSource() (reconcileSource, types.NamespacedName) {
	cmd := reconcileWithSourceCommand{
		apiType: helmChartType,
		object:  helmChartAdapter{&sourcev1b2.HelmChart{}},
		force:   true,
	}

	ns := obj.Spec.Chart.Spec.SourceRef.Namespace
	if ns == "" {
		ns = obj.Namespace
	}

	return cmd, types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", obj.Namespace, obj.Name),
		Namespace: ns,
	}
}

func (obj helmReleaseAdapter) isStatic() bool {
	return false
}
