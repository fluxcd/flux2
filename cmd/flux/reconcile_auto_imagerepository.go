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

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var reconcileImageRepositoryCmd = &cobra.Command{
	Use:   "image-repository [name]",
	Short: "Reconcile an ImageRepository",
	Long:  `The reconcile auto image-repository command triggers a reconciliation of an ImageRepository resource and waits for it to finish.`,
	Example: `  # Trigger an scan for an existing image repository
  flux reconcile auto image-repository alpine
`,
	RunE: reconcileCommand{
		names:  imageRepositoryNames,
		object: imageRepositoryAdapter{&imagev1.ImageRepository{}},
	}.run,
}

func init() {
	reconcileAutoCmd.AddCommand(reconcileImageRepositoryCmd)
}

func (obj imageRepositoryAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

func (obj imageRepositoryAdapter) successMessage() string {
	return fmt.Sprintf("scan fetched %d tags", obj.Status.LastScanResult.TagCount)
}
