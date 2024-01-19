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

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/conditions"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var reconcileSourceHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Reconcile a HelmRepository source",
	Long:  `The reconcile source command triggers a reconciliation of a HelmRepository resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing source
  flux reconcile source helm podinfo`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.HelmRepositoryKind)),
	RunE: reconcileCommand{
		apiType: helmRepositoryType,
		object:  helmRepositoryAdapter{&sourcev1.HelmRepository{}},
	}.run,
}

func init() {
	reconcileSourceCmd.AddCommand(reconcileSourceHelmCmd)
}

func (obj helmRepositoryAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

func (obj helmRepositoryAdapter) successMessage() string {
	// HelmRepository of type OCI don't set an Artifact
	if obj.Spec.Type == sourcev1.HelmRepositoryTypeOCI {
		readyCondition := conditions.Get(obj.HelmRepository, meta.ReadyCondition)
		// This shouldn't happen, successMessage shouldn't be called if
		// object isn't ready
		if readyCondition == nil {
			return ""
		}
		return readyCondition.Message
	}
	return fmt.Sprintf("fetched revision %s", obj.Status.Artifact.Revision)
}

func (obj helmRepositoryAdapter) isStatic() bool {
	return obj.Spec.Type == sourcev1.HelmRepositoryTypeOCI
}
