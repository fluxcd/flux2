/*
Copyright 2026 The Flux authors

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

	swapi "github.com/fluxcd/source-watcher/api/v2/v1beta1"
)

var suspendArtifactGeneratorCmd = &cobra.Command{
	Use:   "generator [name]",
	Short: "Suspend reconciliation of ArtifactGenerator",
	Long: `The suspend command disables the reconciliation of an ArtifactGenerator resource
by setting the source.extensions.fluxcd.io/reconcile annotation to disabled.`,
	Example: `  # Suspend reconciliation for an existing ArtifactGenerator
  flux suspend artifact generator podinfo

  # Suspend reconciliation using the kubectl-style TYPE/NAME form
  flux suspend ag/podinfo

  # Suspend reconciliation for multiple ArtifactGenerators
  flux suspend artifact generator frontend backend`,
	ValidArgsFunction: resourceNamesCompletionFunc(swapi.GroupVersion.WithKind(swapi.ArtifactGeneratorKind)),
	RunE: suspendCommand{
		apiType: artifactGeneratorType,
		object:  &artifactGeneratorAdapter{&swapi.ArtifactGenerator{}},
		list:    &artifactGeneratorListAdapter{&swapi.ArtifactGeneratorList{}},
	}.run,
}

func init() {
	suspendArtifactCmd.AddCommand(suspendArtifactGeneratorCmd)
}

func (obj artifactGeneratorAdapter) isSuspended() bool {
	return obj.ArtifactGenerator.IsDisabled()
}

func (obj artifactGeneratorAdapter) setSuspended() {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[swapi.ReconcileAnnotation] = swapi.DisabledValue
	obj.SetAnnotations(annotations)
}

func (a artifactGeneratorListAdapter) item(i int) suspendable {
	return &artifactGeneratorAdapter{&a.ArtifactGeneratorList.Items[i]}
}
