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

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
)

var getHelmReleaseCmd = &cobra.Command{
	Use:     "helmreleases",
	Aliases: []string{"hr"},
	Short:   "Get HelmRelease statuses",
	Long:    "The get helmreleases command prints the statuses of the resources.",
	Example: `  # List all Helm releases and their status
  gotk get helmreleases
`,
	RunE: getHelmReleaseCmdRun,
}

func init() {
	getCmd.AddCommand(getHelmReleaseCmd)
}

func getHelmReleaseCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var list helmv2.HelmReleaseList
	err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no releases found in %s namespace", namespace)
		return nil
	}

	for _, helmRelease := range list.Items {
		if helmRelease.Spec.Suspend {
			logger.Successf("%s is suspended", helmRelease.GetName())
			continue
		}
		isInitialized := false
		if c := meta.GetCondition(helmRelease.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case corev1.ConditionTrue:
				logger.Successf("%s last applied revision %s", helmRelease.GetName(), helmRelease.Status.LastAppliedRevision)
			case corev1.ConditionUnknown:
				logger.Successf("%s reconciling", helmRelease.GetName())
			default:
				logger.Failuref("%s %s", helmRelease.GetName(), c.Message)
			}
			isInitialized = true
			break
		}
		if !isInitialized {
			logger.Failuref("%s is not ready", helmRelease.GetName())
		}
	}
	return nil
}
