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

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var suspendCmd = &cobra.Command{
	Use:   "suspend",
	Short: "Suspend resources",
	Long:  `The suspend sub-commands suspend the reconciliation of a resource.`,
}

type SuspendFlags struct {
	all bool
}

var suspendArgs SuspendFlags

func init() {
	suspendCmd.PersistentFlags().BoolVarP(&suspendArgs.all, "all", "", false,
		"suspend all resources in that namespace")
	rootCmd.AddCommand(suspendCmd)
}

type suspendable interface {
	adapter
	copyable
	isSuspended() bool
	setSuspended()
}

type suspendCommand struct {
	apiType
	list   listSuspendable
	object suspendable
}

type listSuspendable interface {
	listAdapter
	item(i int) suspendable
}

func (suspend suspendCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 && !suspendArgs.all {
		return fmt.Errorf("%s name is required", suspend.humanKind)
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	if len(args) < 1 && suspendArgs.all {
		listOpts := []client.ListOption{
			client.InNamespace(*kubeconfigArgs.Namespace),
		}

		if err := suspend.patch(ctx, kubeClient, listOpts); err != nil {
			return err
		}

		return nil
	}

	processed := make(map[string]struct{}, len(args))
	for _, arg := range args {
		if _, has := processed[arg]; has {
			continue // skip object that user might have provided more than once
		}
		processed[arg] = struct{}{}

		listOpts := []client.ListOption{
			client.InNamespace(*kubeconfigArgs.Namespace),
			client.MatchingFields{
				"metadata.name": arg,
			},
		}

		if err := suspend.patch(ctx, kubeClient, listOpts); err != nil {
			if err == ErrNoObjectsFound {
				logger.Failuref("%s %s not found in %s namespace", suspend.kind, arg, *kubeconfigArgs.Namespace)
			} else {
				logger.Failuref("failed suspending %s %s in %s namespace: %s", suspend.kind, arg, *kubeconfigArgs.Namespace, err.Error())
			}
		}
	}

	return nil
}

var ErrNoObjectsFound = errors.New("no objects found")

func (suspend suspendCommand) patch(ctx context.Context, kubeClient client.WithWatch, listOpts []client.ListOption) error {
	if err := kubeClient.List(ctx, suspend.list.asClientList(), listOpts...); err != nil {
		return err
	}

	if suspend.list.len() == 0 {
		return ErrNoObjectsFound
	}

	for i := 0; i < suspend.list.len(); i++ {
		logger.Actionf("suspending %s %s in %s namespace", suspend.humanKind, suspend.list.item(i).asClientObject().GetName(), *kubeconfigArgs.Namespace)

		obj := suspend.list.item(i)
		patch := client.MergeFrom(obj.deepCopyClientObject())
		obj.setSuspended()
		if err := kubeClient.Patch(ctx, obj.asClientObject(), patch); err != nil {
			return err
		}

		logger.Successf("%s suspended", suspend.humanKind)
	}

	return nil
}
