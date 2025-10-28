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

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
)

var suspendKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Suspend reconciliation of Kustomization",
	Long:    `The suspend command disables the reconciliation of a Kustomization resource.`,
	Example: `  # Suspend reconciliation for an existing Kustomization
  flux suspend ks podinfo

  # Suspend reconciliation for multiple Kustomizations
  flux suspend ks podinfo-1 podinfo-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE: suspendCommand{
		apiType: kustomizationType,
		object:  kustomizationAdapter{&kustomizev1.Kustomization{}},
		list:    &kustomizationListAdapter{&kustomizev1.KustomizationList{}},
	}.run,
}

func init() {
	suspendCmd.AddCommand(suspendKsCmd)
}

func (obj kustomizationAdapter) isSuspended() bool {
	return obj.Kustomization.Spec.Suspend
}

func (obj kustomizationAdapter) setSuspended(message string) {
	obj.Kustomization.Spec.Suspend = true
	if message != "" {
		obj.Kustomization.Annotations[SuspendMessageAnnotation] = message
	}
}

func (a kustomizationListAdapter) item(i int) suspendable {
	return &kustomizationAdapter{&a.KustomizationList.Items[i]}
}
