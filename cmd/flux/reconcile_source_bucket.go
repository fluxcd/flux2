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

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var reconcileSourceBucketCmd = &cobra.Command{
	Use:   "bucket [name]",
	Short: "Reconcile a Bucket source",
	Long:  `The reconcile source command triggers a reconciliation of a Bucket resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing source
  flux reconcile source bucket podinfo`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.BucketKind)),
	RunE: reconcileCommand{
		apiType: bucketType,
		object:  bucketAdapter{&sourcev1.Bucket{}},
	}.run,
}

func init() {
	reconcileSourceCmd.AddCommand(reconcileSourceBucketCmd)
}

func (obj bucketAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

func (obj bucketAdapter) successMessage() string {
	return fmt.Sprintf("fetched revision %s", obj.Status.Artifact.Revision)
}

func (obj bucketAdapter) isStatic() bool {
	return false
}
