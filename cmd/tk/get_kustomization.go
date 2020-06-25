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

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var getKsCmd = &cobra.Command{
	Use:     "kustomizations",
	Aliases: []string{"ks"},
	Short:   "Get Kustomization source statuses",
	Long:    "The get kustomizations command prints the statuses of the resources.",
	RunE:    getKsCmdRun,
}

func init() {
	getCmd.AddCommand(getKsCmd)
}

func getKsCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var list kustomizev1.KustomizationList
	err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no kustomizations found in %s namespace", namespace)
		return nil
	}

	for _, kustomization := range list.Items {
		if kustomization.Spec.Suspend {
			logger.Successf("%s is suspended", kustomization.GetName())
			continue
		}
		isInitialized := false
		for _, condition := range kustomization.Status.Conditions {
			if condition.Type == kustomizev1.ReadyCondition {
				if condition.Status != corev1.ConditionFalse {
					if kustomization.Status.LastAppliedRevision != "" {
						logger.Successf("%s last applied revision %s", kustomization.GetName(), kustomization.Status.LastAppliedRevision)
					} else {
						logger.Successf("%s reconciling", kustomization.GetName())
					}
				} else {
					logger.Failuref("%s %s", kustomization.GetName(), condition.Message)
				}
				isInitialized = true
				break
			}
		}
		if !isInitialized {
			logger.Failuref("%s is not ready", kustomization.GetName())
		}
	}
	return nil
}
