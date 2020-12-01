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

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	"github.com/fluxcd/flux2/internal/utils"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var suspendImageRepositoryCmd = &cobra.Command{
	Use:   "image-repository [name]",
	Short: "Suspend reconciliation of an ImageRepository",
	Long:  "The suspend command disables the reconciliation of a ImageRepository resource.",
	Example: `  # Suspend reconciliation for an existing ImageRepository
  flux suspend auto image-repository alpine
`,
	RunE: suspendImageRepositoryRun,
}

func init() {
	suspendAutoCmd.AddCommand(suspendImageRepositoryCmd)
}

func suspendImageRepositoryRun(cmd *cobra.Command, args []string) error {
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
	var repository imagev1.ImageRepository
	err = kubeClient.Get(ctx, namespacedName, &repository)
	if err != nil {
		return err
	}

	logger.Actionf("suspending image repository %s in %s namespace", name, namespace)
	repository.Spec.Suspend = true
	if err := kubeClient.Update(ctx, &repository); err != nil {
		return err
	}
	logger.Successf("image repository suspended")

	return nil
}
