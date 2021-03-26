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

var resumeAlertCmd = &cobra.Command{
	Use:   "alert [name]",
	Short: "Resume a suspended Alert",
	Long: `The resume command marks a previously suspended Alert resource for reconciliation and waits for it to
finish the apply.`,
	Example: `  # Resume reconciliation for an existing Alert
  flux resume alert main`,
	RunE: resumeAlertCmdRun,
}

func init() {
	resumeCmd.AddCommand(resumeAlertCmd)
}

func resumeAlertCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Alert name is required")
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
	var alert notificationv1.Alert
	err = kubeClient.Get(ctx, namespacedName, &alert)
	if err != nil {
		return err
	}

	logger.Actionf("resuming Alert %s in %s namespace", name, rootArgs.namespace)
	alert.Spec.Suspend = false
	if err := kubeClient.Update(ctx, &alert); err != nil {
		return err
	}
	logger.Successf("Alert resumed")

	logger.Waitingf("waiting for Alert reconciliation")
	if err := wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
		isAlertResumed(ctx, kubeClient, namespacedName, &alert)); err != nil {
		return err
	}
	logger.Successf("Alert reconciliation completed")
	return nil
}

func isAlertResumed(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, alert *notificationv1.Alert) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, alert)
		if err != nil {
			return false, err
		}

		if c := apimeta.FindStatusCondition(alert.Status.Conditions, meta.ReadyCondition); c != nil {
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
