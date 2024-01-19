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

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
)

var deleteHelmReleaseCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Delete a HelmRelease resource",
	Long:    withPreviewNote("The delete helmrelease command removes the given HelmRelease from the cluster."),
	Example: `  # Delete a Helm release and the Kubernetes resources created by it
  flux delete hr podinfo`,
	ValidArgsFunction: resourceNamesCompletionFunc(helmv2.GroupVersion.WithKind(helmv2.HelmReleaseKind)),
	RunE: deleteCommand{
		apiType: helmReleaseType,
		object:  universalAdapter{&helmv2.HelmRelease{}},
	}.run,
}

func init() {
	deleteCmd.AddCommand(deleteHelmReleaseCmd)
}
