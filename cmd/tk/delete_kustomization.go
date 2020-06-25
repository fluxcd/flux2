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

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

var deleteKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Delete a Kustomization resource",
	Long:    "The delete kustomization command deletes the given Kustomization from the cluster.",
	RunE:    deleteKsCmdRun,
}

func init() {
	deleteCmd.AddCommand(deleteKsCmd)
}

func deleteKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("kustomization name is required")
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

	var kustomization kustomizev1.Kustomization
	err = kubeClient.Get(ctx, namespacedName, &kustomization)
	if err != nil {
		return err
	}

	if !deleteSilent {
		if !kustomization.Spec.Suspend {
			logger.Waitingf("This action will remove the Kubernetes objects previously applied by the %s kustomization!", name)
		}
		prompt := promptui.Prompt{
			Label:     "Are you sure you want to delete this kustomization",
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	logger.Actionf("deleting kustomization %s in %s namespace", name, namespace)
	err = kubeClient.Delete(ctx, &kustomization)
	if err != nil {
		return err
	}
	logger.Successf("kustomization deleted")

	return nil
}
