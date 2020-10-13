/*
Copyright 2020 The Flux CD contributors.

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
	"context"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
)

var deleteAlertProviderCmd = &cobra.Command{
	Use:   "alert-provider [name]",
	Short: "Delete a Provider resource",
	Long:  "The delete alert-provider command removes the given Provider from the cluster.",
	Example: `  # Delete a Provider and the Kubernetes resources created by it
  gotk delete alert-provider slack
`,
	RunE: deleteAlertProviderCmdRun,
}

func init() {
	deleteCmd.AddCommand(deleteAlertProviderCmd)
}

func deleteAlertProviderCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("provider name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	var alertProvider notificationv1.Provider
	err = kubeClient.Get(ctx, namespacedName, &alertProvider)
	if err != nil {
		return err
	}

	if !deleteSilent {
		prompt := promptui.Prompt{
			Label:     "Are you sure you want to delete this Provider",
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	logger.Actionf("deleting provider %s in %s namespace", name, namespace)
	err = kubeClient.Delete(ctx, &alertProvider)
	if err != nil {
		return err
	}
	logger.Successf("provider deleted")

	return nil
}
