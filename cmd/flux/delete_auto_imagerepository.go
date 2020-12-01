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
	"context"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	"github.com/fluxcd/flux2/internal/utils"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var deleteImageRepositoryCmd = &cobra.Command{
	Use:   "image-repository [name]",
	Short: "Delete an ImageRepository object",
	Long:  "The delete auto image-repository command deletes the given ImageRepository from the cluster.",
	Example: `  # Delete an image repository
  flux delete auto image-repository alpine
`,
	RunE: deleteImageRepositoryRun,
}

func init() {
	deleteAutoCmd.AddCommand(deleteImageRepositoryCmd)
}

func deleteImageRepositoryRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("image repository name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	var repo imagev1.ImageRepository
	err = kubeClient.Get(ctx, namespacedName, &repo)
	if err != nil {
		return err
	}

	if !deleteSilent {
		prompt := promptui.Prompt{
			Label:     "Are you sure you want to delete this image repository",
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	logger.Actionf("deleting image repository %s in %s namespace", name, namespace)
	err = kubeClient.Delete(ctx, &repo)
	if err != nil {
		return err
	}
	logger.Successf("image repository deleted")

	return nil
}
