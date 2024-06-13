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

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"
)

var reconcileKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Reconcile a Kustomization resource",
	Long: `
The reconcile kustomization command triggers a reconciliation of a Kustomization resource and waits for it to finish.`,
	Example: `  # Trigger a Kustomization apply outside of the reconciliation interval
  flux reconcile kustomization podinfo

  # Trigger a sync of the Kustomization's source and apply changes
  flux reconcile kustomization podinfo --with-source`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE: reconcileWithSourceCommand{
		apiType: kustomizationType,
		object:  kustomizationAdapter{&kustomizev1.Kustomization{}},
	}.run,
}

type reconcileKsFlags struct {
	syncKsWithSource bool
}

var rksArgs reconcileKsFlags

func init() {
	reconcileKsCmd.Flags().BoolVar(&rksArgs.syncKsWithSource, "with-source", false, "reconcile Kustomization source")

	reconcileCmd.AddCommand(reconcileKsCmd)
}

func (obj kustomizationAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

func (obj kustomizationAdapter) reconcileSource() bool {
	return rksArgs.syncKsWithSource
}

func (obj kustomizationAdapter) getSource() (reconcileSource, types.NamespacedName) {
	var cmd reconcileCommand
	switch obj.Spec.SourceRef.Kind {
	case sourcev1b2.OCIRepositoryKind:
		cmd = reconcileCommand{
			apiType: ociRepositoryType,
			object:  ociRepositoryAdapter{&sourcev1b2.OCIRepository{}},
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
		Namespace: obj.Spec.SourceRef.Namespace,
	}
}

func (obj kustomizationAdapter) isStatic() bool {
	return false
}
