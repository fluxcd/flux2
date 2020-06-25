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

	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var getSourceGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Get GitRepository source statuses",
	Long:  "The get sources git command prints the status of the GitRepository sources.",
	RunE:  getSourceGitCmdRun,
}

func init() {
	getSourceCmd.AddCommand(getSourceGitCmd)
}

func getSourceGitCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var list sourcev1.GitRepositoryList
	err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no sources found in %s namespace", namespace)
		return nil
	}

	for _, source := range list.Items {
		isInitialized := false
		for _, condition := range source.Status.Conditions {
			if condition.Type == sourcev1.ReadyCondition {
				if condition.Status != corev1.ConditionFalse {
					logger.Successf("%s last fetched revision: %s", source.GetName(), source.Status.Artifact.Revision)
				} else {
					logger.Failuref("%s %s", source.GetName(), condition.Message)
				}
				isInitialized = true
				break
			}
		}
		if !isInitialized {
			logger.Failuref("%s is not ready", source.GetName())
		}
	}
	return nil
}
