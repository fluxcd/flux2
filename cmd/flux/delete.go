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
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete sources and resources",
	Long:  "The delete sub-commands delete sources and resources.",
}

var (
	deleteSilent bool
)

func init() {
	deleteCmd.PersistentFlags().BoolVarP(&deleteSilent, "silent", "s", false,
		"delete resource without asking for confirmation")

	rootCmd.AddCommand(deleteCmd)
}

type deleteCommand struct {
	apiType
	object adapter // for getting the value, and later deleting it
}

func (del deleteCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", del.humanKind)
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

	err = kubeClient.Get(ctx, namespacedName, del.object.asRuntimeObject())
	if err != nil {
		return err
	}

	if !deleteSilent {
		prompt := promptui.Prompt{
			Label:     "Are you sure you want to delete this " + del.humanKind,
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	logger.Actionf("deleting %s %s in %s namespace", del.humanKind, name, namespace)
	err = kubeClient.Delete(ctx, del.object.asRuntimeObject())
	if err != nil {
		return err
	}
	logger.Successf("%s deleted", del.humanKind)

	return nil
}
