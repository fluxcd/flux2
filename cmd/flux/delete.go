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
	"errors"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete sources and resources",
	Long:  `The delete sub-commands delete sources and resources.`,
	Example: `  # Delete a Kustomization
  flux delete kustomization podinfo

  # Delete a Kustomization using the kubectl-style TYPE/NAME form
  flux delete ks/podinfo

  # Delete several Kustomizations
  flux delete kustomization frontend backend

  # Delete a Git repository by its fully qualified type without asking for confirmation
  flux delete gitrepositories.source.toolkit.fluxcd.io/podinfo --silent`,
}

type deleteFlags struct {
	silent bool
}

var deleteArgs deleteFlags

func init() {
	deleteCmd.PersistentFlags().BoolVarP(&deleteArgs.silent, "silent", "s", false,
		"delete resource without asking for confirmation")

	rootCmd.AddCommand(deleteCmd)
}

type deleteCommand struct {
	apiType
	object adapter // for getting the value, and later deleting it
}

// errDeleteAborted is returned when the user declines a delete prompt.
var errDeleteAborted = errors.New("aborting")

func (del deleteCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", del.humanKind)
	}
	names := uniqueNames(args)

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	if len(names) == 1 {
		return del.delete(kubeClient, names[0])
	}

	failed := 0
	for _, name := range names {
		err := del.delete(kubeClient, name)
		switch {
		case err == nil:
		case errors.Is(err, errDeleteAborted):
			return err
		case apierrors.IsNotFound(err):
			failed++
			logger.Failuref("%s %s not found in %s namespace", del.humanKind, name, *kubeconfigArgs.Namespace)
		default:
			failed++
			logger.Failuref("failed to delete %s %s in %s namespace: %s", del.humanKind, name, *kubeconfigArgs.Namespace, err.Error())
		}
	}
	if failed > 0 {
		return fmt.Errorf("failed to delete %d of %d %s objects", failed, len(names), del.humanKind)
	}
	return nil
}

// delete deletes the named object after asking for confirmation,
// unless --silent is set.
func (del deleteCommand) delete(kubeClient client.Client, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	namespacedName := types.NamespacedName{
		Namespace: *kubeconfigArgs.Namespace,
		Name:      name,
	}

	obj := del.object.asClientObject()
	resetObject(obj)
	if err := kubeClient.Get(ctx, namespacedName, obj); err != nil {
		return err
	}

	if !deleteArgs.silent {
		prompt := promptui.Prompt{
			Label:     "Are you sure you want to delete this " + del.humanKind,
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return errDeleteAborted
		}
	}

	logger.Actionf("deleting %s %s in %s namespace", del.humanKind, name, *kubeconfigArgs.Namespace)
	if err := kubeClient.Delete(ctx, obj); err != nil {
		return err
	}
	logger.Successf("%s deleted", del.humanKind)

	return nil
}
