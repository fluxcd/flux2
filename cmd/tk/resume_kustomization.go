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

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var resumeKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Resume a suspended Kustomization",
	Long: `The resume command marks a previously suspended Kustomization resource for reconciliation and waits for it to
finish the apply.`,
	RunE: resumeKsCmdRun,
}

func init() {
	resumeCmd.AddCommand(resumeKsCmd)
}

func resumeKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("kustomization name is required")
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
	var kustomization kustomizev1.Kustomization
	err = kubeClient.Get(ctx, namespacedName, &kustomization)
	if err != nil {
		return err
	}

	logger.Actionf("resuming kustomization %s in %s namespace", name, namespace)
	kustomization.Spec.Suspend = false
	if err := kubeClient.Update(ctx, &kustomization); err != nil {
		return err
	}
	logger.Successf("kustomization resumed")

	logger.Waitingf("waiting for kustomization sync")
	if err := wait.PollImmediate(pollInterval, timeout,
		isKustomizationResumed(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("kustomization sync completed")

	err = kubeClient.Get(ctx, namespacedName, &kustomization)
	if err != nil {
		return err
	}

	if kustomization.Status.LastAppliedRevision != "" {
		logger.Successf("applied revision %s", kustomization.Status.LastAppliedRevision)
	} else {
		return fmt.Errorf("kustomization sync failed")
	}

	return nil
}

func isKustomizationResumed(ctx context.Context, kubeClient client.Client, name, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		var kustomization kustomizev1.Kustomization
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := kubeClient.Get(ctx, namespacedName, &kustomization)
		if err != nil {
			return false, err
		}

		for _, condition := range kustomization.Status.Conditions {
			if condition.Type == sourcev1.ReadyCondition {
				if condition.Status == corev1.ConditionTrue {
					return true, nil
				} else if condition.Status == corev1.ConditionFalse {
					if condition.Reason == kustomizev1.SuspendedReason {
						return false, nil
					}

					return false, fmt.Errorf(condition.Message)
				}
			}
		}
		return false, nil
	}
}
