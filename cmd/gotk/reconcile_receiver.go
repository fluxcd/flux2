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
	"github.com/fluxcd/pkg/apis/meta"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
)

var reconcileReceiverCmd = &cobra.Command{
	Use:   "receiver [name]",
	Short: "Reconcile a Receiver",
	Long:  `The reconcile receiver command triggers a reconciliation of a Receiver resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing receiver
  gotk reconcile receiver main
`,
	RunE: reconcileReceiverCmdRun,
}

func init() {
	reconcileCmd.AddCommand(reconcileReceiverCmd)
}

func reconcileReceiverCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("receiver name is required")
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

	logger.Actionf("annotating receiver %s in %s namespace", name, namespace)
	var receiver notificationv1.Receiver
	err = kubeClient.Get(ctx, namespacedName, &receiver)
	if err != nil {
		return err
	}

	if receiver.Annotations == nil {
		receiver.Annotations = map[string]string{
			meta.ReconcileAtAnnotation: time.Now().Format(time.RFC3339Nano),
		}
	} else {
		receiver.Annotations[meta.ReconcileAtAnnotation] = time.Now().Format(time.RFC3339Nano)
	}
	if err := kubeClient.Update(ctx, &receiver); err != nil {
		return err
	}
	logger.Successf("receiver annotated")

	logger.Waitingf("waiting for reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isReceiverReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("receiver reconciliation completed")

	return nil
}
