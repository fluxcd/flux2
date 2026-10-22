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
	"time"

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"

	"github.com/fluxcd/pkg/apis/meta"
	swapi "github.com/fluxcd/source-watcher/api/v2/v1beta1"
)

var resumeArtifactGeneratorCmd = &cobra.Command{
	Use:   "generator [name]",
	Short: "Resume reconciliation of ArtifactGenerator",
	Long: `The resume command removes the disabled reconcile annotation from an ArtifactGenerator resource,
requests a reconciliation and waits for it to finish.`,
	Example: `  # Resume reconciliation for an existing ArtifactGenerator
  flux resume artifact generator podinfo

  # Resume reconciliation using the kubectl-style TYPE/NAME form
  flux resume ag/podinfo

  # Resume reconciliation for multiple ArtifactGenerators
  flux resume artifact generator frontend backend`,
	ValidArgsFunction: resourceNamesCompletionFunc(swapi.GroupVersion.WithKind(swapi.ArtifactGeneratorKind)),
	RunE: resumeCommand{
		apiType: artifactGeneratorType,
		list:    artifactGeneratorListAdapter{&swapi.ArtifactGeneratorList{}},
	}.run,
}

func init() {
	resumeArtifactCmd.AddCommand(resumeArtifactGeneratorCmd)
}

// getObservedGeneration returns the observed generation of the Ready
// condition, as ArtifactGeneratorStatus has no observedGeneration field.
func (obj artifactGeneratorAdapter) getObservedGeneration() int64 {
	if c := apimeta.FindStatusCondition(obj.Status.Conditions, meta.ReadyCondition); c != nil {
		return c.ObservedGeneration
	}
	return 0
}

// setUnsuspended removes the disabled reconcile annotation and requests a
// reconciliation, as removing the annotation does not trigger one.
func (obj artifactGeneratorAdapter) setUnsuspended() {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}
	delete(annotations, swapi.ReconcileAnnotation)
	annotations[meta.ReconcileRequestAnnotation] = time.Now().Format(time.RFC3339Nano)
	obj.SetAnnotations(annotations)
}

func (obj artifactGeneratorAdapter) reconcileRequestedAt() string {
	return obj.GetAnnotations()[meta.ReconcileRequestAnnotation]
}

func (obj artifactGeneratorAdapter) successMessage() string {
	return "ArtifactGenerator reconciliation completed"
}

func (a artifactGeneratorListAdapter) resumeItem(i int) resumable {
	return &artifactGeneratorAdapter{&a.ArtifactGeneratorList.Items[i]}
}
