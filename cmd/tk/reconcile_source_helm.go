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
	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

var reconcileSourceHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Reconcile a HelmRepository source",
	Long:  `The reconcile source command triggers a reconciliation of a HelmRepository resource and waits for it to finish.`,
	Example: `  # Trigger a helm repo update for an existing source
  tk reconcile source helm podinfo
`,
	RunE: syncSourceHelmCmdRun,
}

func init() {
	reconcileSourceCmd.AddCommand(reconcileSourceHelmCmd)
}

func syncSourceHelmCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("source name is required")
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

	logger.Actionf("annotating source %s in %s namespace", name, namespace)
	var helmRepository sourcev1.HelmRepository
	err = kubeClient.Get(ctx, namespacedName, &helmRepository)
	if err != nil {
		return err
	}

	if helmRepository.Annotations == nil {
		helmRepository.Annotations = map[string]string{
			sourcev1.ReconcileAtAnnotation: time.Now().Format(time.RFC3339Nano),
		}
	} else {
		helmRepository.Annotations[sourcev1.ReconcileAtAnnotation] = time.Now().Format(time.RFC3339Nano)
	}
	if err := kubeClient.Update(ctx, &helmRepository); err != nil {
		return err
	}
	logger.Successf("source annotated")

	logger.Waitingf("waiting for reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isGitRepositoryReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("helm reconciliation completed")

	err = kubeClient.Get(ctx, namespacedName, &helmRepository)
	if err != nil {
		return err
	}

	if helmRepository.Status.Artifact != nil {
		logger.Successf("fetched revision %s", helmRepository.Status.Artifact.Revision)
	} else {
		return fmt.Errorf("helm reconciliation failed, artifact not found")
	}
	return nil
}
