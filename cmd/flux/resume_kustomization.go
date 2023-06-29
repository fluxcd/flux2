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

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
)

var resumeKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Resume a suspended Kustomization",
	Long: `The resume command marks a previously suspended Kustomization resource for reconciliation and waits for it to
finish the apply.`,
	Example: `  # Resume reconciliation for an existing Kustomization
  flux resume ks podinfo

  # Resume reconciliation for multiple Kustomizations
  flux resume ks podinfo-1 podinfo-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE: resumeCommand{
		apiType: kustomizationType,
		list:    kustomizationListAdapter{&kustomizev1.KustomizationList{}},
	}.run,
}

func init() {
	resumeCmd.AddCommand(resumeKsCmd)
}

func (obj kustomizationAdapter) getObservedGeneration() int64 {
	return obj.Kustomization.Status.ObservedGeneration
}

func (obj kustomizationAdapter) setUnsuspended() {
	obj.Kustomization.Spec.Suspend = false
}

func (obj kustomizationAdapter) successMessage() string {
	return fmt.Sprintf("applied revision %s", obj.Status.LastAppliedRevision)
}

func (a kustomizationListAdapter) resumeItem(i int) resumable {
	return &kustomizationAdapter{&a.KustomizationList.Items[i]}
}
