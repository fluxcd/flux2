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
)

var suspendCmd = &cobra.Command{
	Use:   "suspend",
	Short: "Suspend resources",
	Long:  "The suspend sub-commands suspend the reconciliation of a resource.",
}

func init() {
	rootCmd.AddCommand(suspendCmd)
}

type suspendable interface {
	adapter
	isSuspended() bool
	setSuspended()
}

type suspendCommand struct {
	apiType
	object suspendable
}

func (suspend suspendCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", suspend.humanKind)
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: rootArgs.namespace,
		Name:      name,
	}
	err = kubeClient.Get(ctx, namespacedName, suspend.object.asClientObject())
	if err != nil {
		return err
	}

	logger.Actionf("suspending %s %s in %s namespace", suspend.humanKind, name, rootArgs.namespace)
	suspend.object.setSuspended()
	if err := kubeClient.Update(ctx, suspend.object.asClientObject()); err != nil {
		return err
	}
	logger.Successf("%s suspended", suspend.humanKind)

	return nil
}
