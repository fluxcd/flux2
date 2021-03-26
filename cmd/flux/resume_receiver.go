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

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
)

var resumeReceiverCmd = &cobra.Command{
	Use:   "receiver [name]",
	Short: "Resume a suspended Receiver",
	Long: `The resume command marks a previously suspended Receiver resource for reconciliation and waits for it to
finish the apply.`,
	Example: `  # Resume reconciliation for an existing Receiver
  flux resume receiver main`,
	RunE: resumeReceiverCmdRun,
}

func init() {
	resumeCmd.AddCommand(resumeReceiverCmd)
}

func resumeReceiverCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Receiver name is required")
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
	var receiver notificationv1.Receiver
	err = kubeClient.Get(ctx, namespacedName, &receiver)
	if err != nil {
		return err
	}

	logger.Actionf("resuming Receiver %s in %s namespace", name, rootArgs.namespace)
	receiver.Spec.Suspend = false
	if err := kubeClient.Update(ctx, &receiver); err != nil {
		return err
	}
	logger.Successf("Receiver resumed")

	logger.Waitingf("waiting for Receiver reconciliation")
	if err := wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
		isReceiverResumed(ctx, kubeClient, namespacedName, &receiver)); err != nil {
		return err
	}

	logger.Successf("Receiver reconciliation completed")
	return nil
}

func isReceiverResumed(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, receiver *notificationv1.Receiver) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, receiver)
		if err != nil {
			return false, err
		}

		if c := apimeta.FindStatusCondition(receiver.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case metav1.ConditionTrue:
				return true, nil
			case metav1.ConditionFalse:
				if c.Reason == meta.SuspendedReason {
					return false, nil
				}
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}
