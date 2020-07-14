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

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2alpha1"
)

var resumeHrCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Resume a suspended HelmRelease",
	Long: `The resume command marks a previously suspended HelmRelease resource for reconciliation and waits for it to
finish the apply.`,
	RunE: resumeHrCmdRun,
}

func init() {
	resumeCmd.AddCommand(resumeHrCmd)
}

func resumeHrCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("HelmRelease name is required")
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
	var helmRelease helmv2.HelmRelease
	err = kubeClient.Get(ctx, namespacedName, &helmRelease)
	if err != nil {
		return err
	}

	logger.Actionf("resuming HelmRelease %s in %s namespace", name, namespace)
	helmRelease.Spec.Suspend = false
	if err := kubeClient.Update(ctx, &helmRelease); err != nil {
		return err
	}
	logger.Successf("HelmRelease resumed")

	logger.Waitingf("waiting for HelmRelease reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isHelmReleaseResumed(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("HelmRelease reconciliation completed")

	err = kubeClient.Get(ctx, namespacedName, &helmRelease)
	if err != nil {
		return err
	}

	if helmRelease.Status.LastAppliedRevision != "" {
		logger.Successf("applied revision %s", helmRelease.Status.LastAppliedRevision)
	} else {
		return fmt.Errorf("HelmRelease reconciliation failed")
	}

	return nil
}

func isHelmReleaseResumed(ctx context.Context, kubeClient client.Client, name, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		var helmRelease helmv2.HelmRelease
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := kubeClient.Get(ctx, namespacedName, &helmRelease)
		if err != nil {
			return false, err
		}

		for _, condition := range helmRelease.Status.Conditions {
			if condition.Type == helmv2.ReadyCondition {
				if condition.Status == corev1.ConditionTrue {
					return true, nil
				} else if condition.Status == corev1.ConditionFalse {
					if condition.Reason == helmv2.SuspendedReason {
						return false, nil
					}

					return false, fmt.Errorf(condition.Message)
				}
			}
		}
		return false, nil
	}
}
