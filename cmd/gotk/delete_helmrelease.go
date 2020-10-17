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

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/fluxcd/toolkit/internal/utils"
)

var deleteHelmReleaseCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Delete a HelmRelease resource",
	Long:    "The delete helmrelease command removes the given HelmRelease from the cluster.",
	Example: `  # Delete a Helm release and the Kubernetes resources created by it
  gotk delete hr podinfo
`,
	RunE: deleteHelmReleaseCmdRun,
}

func init() {
	deleteCmd.AddCommand(deleteHelmReleaseCmd)
}

func deleteHelmReleaseCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("release name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	var helmRelease helmv2.HelmRelease
	err = kubeClient.Get(ctx, namespacedName, &helmRelease)
	if err != nil {
		return err
	}

	if !deleteSilent {
		if !helmRelease.Spec.Suspend {
			logger.Waitingf("This action will remove the Kubernetes objects previously applied by the %s Helm release!", name)
		}
		prompt := promptui.Prompt{
			Label:     "Are you sure you want to delete this Helm release",
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	logger.Actionf("deleting release %s in %s namespace", name, namespace)
	err = kubeClient.Delete(ctx, &helmRelease)
	if err != nil {
		return err
	}
	logger.Successf("release deleted")

	return nil
}
