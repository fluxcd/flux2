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

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/fluxcd/flux2/internal/utils"
)

var reconcileAlertProviderCmd = &cobra.Command{
	Use:   "alert-provider [name]",
	Short: "Reconcile a Provider",
	Long:  `The reconcile alert-provider command triggers a reconciliation of a Provider resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing provider
  flux reconcile alert-provider slack`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.ProviderKind)),
	RunE:              reconcileAlertProviderCmdRun,
}

func init() {
	reconcileCmd.AddCommand(reconcileAlertProviderCmd)
}

func reconcileAlertProviderCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Provider name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: *kubeconfigArgs.Namespace,
		Name:      name,
	}

	logger.Actionf("annotating Provider %s in %s namespace", name, *kubeconfigArgs.Namespace)
	var alertProvider notificationv1.Provider
	err = kubeClient.Get(ctx, namespacedName, &alertProvider)
	if err != nil {
		return err
	}

	if alertProvider.Annotations == nil {
		alertProvider.Annotations = map[string]string{
			meta.ReconcileRequestAnnotation: time.Now().Format(time.RFC3339Nano),
		}
	} else {
		alertProvider.Annotations[meta.ReconcileRequestAnnotation] = time.Now().Format(time.RFC3339Nano)
	}
	if err := kubeClient.Update(ctx, &alertProvider); err != nil {
		return err
	}
	logger.Successf("Provider annotated")

	logger.Waitingf("waiting for reconciliation")
	if err := wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
		isAlertProviderReady(ctx, kubeClient, namespacedName, &alertProvider)); err != nil {
		return err
	}
	logger.Successf("Provider reconciliation completed")
	return nil
}
