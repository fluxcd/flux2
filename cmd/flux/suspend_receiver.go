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

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/toolkit/internal/utils"
)

var suspendReceiverCmd = &cobra.Command{
	Use:   "receiver [name]",
	Short: "Suspend reconciliation of Receiver",
	Long:  "The suspend command disables the reconciliation of a Receiver resource.",
	Example: `  # Suspend reconciliation for an existing Receiver
  flux suspend receiver main
`,
	RunE: suspendReceiverCmdRun,
}

func init() {
	suspendCmd.AddCommand(suspendReceiverCmd)
}

func suspendReceiverCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Receiver name is required")
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
	var receiver notificationv1.Receiver
	err = kubeClient.Get(ctx, namespacedName, &receiver)
	if err != nil {
		return err
	}

	logger.Actionf("suspending Receiver %s in %s namespace", name, namespace)
	receiver.Spec.Suspend = true
	if err := kubeClient.Update(ctx, &receiver); err != nil {
		return err
	}
	logger.Successf("Receiver suspended")

	return nil
}
