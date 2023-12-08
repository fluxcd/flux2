/*
Copyright 2022 The Flux authors

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

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var reconcileSourceOCIRepositoryCmd = &cobra.Command{
	Use:   "oci [name]",
	Short: "Reconcile an OCIRepository",
	Long:  `The reconcile source command triggers a reconciliation of an OCIRepository resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing source
  flux reconcile source oci podinfo`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.OCIRepositoryKind)),
	RunE: reconcileCommand{
		apiType: ociRepositoryType,
		object:  ociRepositoryAdapter{&sourcev1.OCIRepository{}},
	}.run,
}

func init() {
	reconcileSourceCmd.AddCommand(reconcileSourceOCIRepositoryCmd)
}

func (obj ociRepositoryAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

func (obj ociRepositoryAdapter) successMessage() string {
	return fmt.Sprintf("fetched revision %s", obj.Status.Artifact.Revision)
}

func (obj ociRepositoryAdapter) isStatic() bool {
	return false
}
