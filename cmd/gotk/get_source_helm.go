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

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var getSourceHelmCmd = &cobra.Command{
	Use:   "helm",
	Short: "Get HelmRepository source statuses",
	Long:  "The get sources helm command prints the status of the HelmRepository sources.",
	Example: `  # List all Helm repositories and their status
  gotk get sources helm
`,
	RunE: getSourceHelmCmdRun,
}

func init() {
	getSourceCmd.AddCommand(getSourceHelmCmd)
}

func getSourceHelmCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var list sourcev1.HelmRepositoryList
	err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no sources found in %s namespace", namespace)
		return nil
	}

	// TODO(hidde): this should print a table, and should produce better output
	//  for items that have an artifact attached while they are in a reconciling
	//  'Unknown' state.
	for _, source := range list.Items {
		isInitialized := false
		if c := meta.GetCondition(source.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case corev1.ConditionTrue:
				logger.Successf("%s last fetched revision: %s", source.GetName(), source.GetArtifact().Revision)
			case corev1.ConditionUnknown:
				logger.Successf("%s reconciling", source.GetName())
			default:
				logger.Failuref("%s %s", source.GetName(), c.Message)
			}
			isInitialized = true
		}
		if !isInitialized {
			logger.Failuref("%s is not ready", source.GetName())
		}
	}
	return nil
}
