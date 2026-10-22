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

var reconcileArtifactGeneratorCmd = &cobra.Command{
	Use:   "generator [name]",
	Short: "Reconcile an ArtifactGenerator",
	Long:  `The reconcile artifact generator command triggers a reconciliation of an ArtifactGenerator resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing ArtifactGenerator
  flux reconcile artifact generator podinfo

  # Trigger a reconciliation using the kubectl-style TYPE/NAME form
  flux reconcile ag/podinfo

  # Trigger a reconciliation for multiple ArtifactGenerators
  flux reconcile artifact generator frontend backend`,
	ValidArgsFunction: resourceNamesCompletionFunc(swapi.GroupVersion.WithKind(swapi.ArtifactGeneratorKind)),
	RunE: reconcileCommand{
		apiType: artifactGeneratorType,
		object:  artifactGeneratorAdapter{&swapi.ArtifactGenerator{}},
	}.run,
}

func init() {
	reconcileArtifactCmd.AddCommand(reconcileArtifactGeneratorCmd)
}

func (obj artifactGeneratorAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

func (obj artifactGeneratorAdapter) isStatic() bool {
	return false
}
