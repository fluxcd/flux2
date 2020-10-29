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
	"time"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
)

var reconcileAlertCmd = &cobra.Command{
	Use:   "alert [name]",
	Short: "Reconcile an Alert",
	Long:  `The reconcile alert command triggers a reconciliation of an Alert resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing alert
  flux reconcile alert main
`,
	RunE: reconcileAlertCmdRun,
}

func init() {
	reconcileCmd.AddCommand(reconcileAlertCmd)
}

func reconcileAlertCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Alert name is required")
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

	logger.Actionf("annotating Alert %s in %s namespace", name, namespace)
	var alert notificationv1.Alert
	err = kubeClient.Get(ctx, namespacedName, &alert)
	if err != nil {
		return err
	}

	if alert.Annotations == nil {
		alert.Annotations = map[string]string{
			meta.ReconcileAtAnnotation: time.Now().Format(time.RFC3339Nano),
		}
	} else {
		alert.Annotations[meta.ReconcileAtAnnotation] = time.Now().Format(time.RFC3339Nano)
	}
	if err := kubeClient.Update(ctx, &alert); err != nil {
		return err
	}
	logger.Successf("Alert annotated")

	logger.Waitingf("waiting for reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isAlertReady(ctx, kubeClient, namespacedName, &alert)); err != nil {
		return err
	}
	logger.Successf("Alert reconciliation completed")
	return nil
}
