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

var resumeArtifactGeneratorCmd = &cobra.Command{
	Use:   "generator [name]",
	Short: "Resume reconciliation of ArtifactGenerator",
	Long:  `The resume command enables the reconciliation of an ArtifactGenerator resource.`,
	Example: `  # Resume reconciliation for an existing ArtifactGenerator
  flux resume artifact generator main

  # Resume reconciliation for multiple ArtifactGenerators
  flux resume artifact generator main-1 main-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(swapi.GroupVersion.WithKind(swapi.ArtifactGeneratorKind)),
	RunE: resumeCommand{
		apiType: artifactGeneratorType,
		list:    artifactGeneratorListAdapter{&swapi.ArtifactGeneratorList{}},
	}.run,
}

func init() {
	resumeArtifactCmd.AddCommand(resumeArtifactGeneratorCmd)
}

func (obj artifactGeneratorAdapter) getObservedGeneration() int64 {
	// ArtifactGeneratorStatus has no observedGeneration field.
	return 0
}

func (obj artifactGeneratorAdapter) setUnsuspended() {
	annotations := obj.GetAnnotations()
	delete(annotations, swapi.ReconcileAnnotation)
	obj.SetAnnotations(annotations)
}

func (obj artifactGeneratorAdapter) successMessage() string {
	return "ArtifactGenerator reconciliation completed"
}

func (a artifactGeneratorListAdapter) resumeItem(i int) resumable {
	return &artifactGeneratorAdapter{&a.ArtifactGeneratorList.Items[i]}
}
