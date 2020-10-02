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
	"github.com/fluxcd/pkg/apis/meta"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var getKsCmd = &cobra.Command{
	Use:     "kustomizations",
	Aliases: []string{"ks"},
	Short:   "Get Kustomization statuses",
	Long:    "The get kustomizations command prints the statuses of the resources.",
	Example: `  # List all kustomizations and their status
  gotk get kustomizations
`,
	RunE: getKsCmdRun,
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
		if c := meta.GetCondition(kustomization.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case corev1.ConditionTrue:
				logger.Successf("%s last applied revision %s", kustomization.GetName(), kustomization.Status.LastAppliedRevision)
			case corev1.ConditionUnknown:
				logger.Successf("%s reconciling", kustomization.GetName())
			default:
				logger.Failuref("%s %s", kustomization.GetName(), c.Message)
			}
			isInitialized = true
		}
		if !isInitialized {
			logger.Failuref("%s is not ready", kustomization.GetName())
		}
	}
	return nil
}
