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

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta3"
)

var deleteAlertProviderCmd = &cobra.Command{
	Use:   "alert-provider [name]",
	Short: "Delete a Provider resource",
	Long:  withPreviewNote("The delete alert-provider command removes the given Provider from the cluster."),
	Example: `  # Delete a Provider and the Kubernetes resources created by it
  flux delete alert-provider slack`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.ProviderKind)),
	RunE: deleteCommand{
		apiType: alertProviderType,
		object:  universalAdapter{&notificationv1.Provider{}},
	}.run,
}

func init() {
	deleteCmd.AddCommand(deleteAlertProviderCmd)
}
