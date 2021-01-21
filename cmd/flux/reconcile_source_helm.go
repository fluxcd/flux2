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

	"github.com/fluxcd/pkg/apis/meta"
	"k8s.io/client-go/util/retry"

	"github.com/fluxcd/flux2/internal/utils"

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var reconcileSourceHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Reconcile a HelmRepository source",
	Long:  `The reconcile source command triggers a reconciliation of a HelmRepository resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing source
  flux reconcile source helm podinfo
`,
	RunE: reconcileSourceHelmCmdRun,
}

func init() {
	reconcileSourceCmd.AddCommand(reconcileSourceHelmCmd)
}

func reconcileSourceHelmCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("HelmRepository source name is required")
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
	var repository sourcev1.HelmRepository
	err = kubeClient.Get(ctx, namespacedName, &repository)
	if err != nil {
		return err
	}

	if repository.Spec.Suspend {
		return fmt.Errorf("resource is suspended")
	}

	logger.Actionf("annotating HelmRepository source %s in %s namespace", name, rootArgs.namespace)
	if err := requestHelmRepositoryReconciliation(ctx, kubeClient, namespacedName, &repository); err != nil {
		return err
	}
	logger.Successf("HelmRepository source annotated")

	lastHandledReconcileAt := repository.Status.LastHandledReconcileAt
	logger.Waitingf("waiting for HelmRepository source reconciliation")
	if err := wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
		helmRepositoryReconciliationHandled(ctx, kubeClient, namespacedName, &repository, lastHandledReconcileAt)); err != nil {
		return err
	}
	logger.Successf("HelmRepository source reconciliation completed")

	if apimeta.IsStatusConditionFalse(repository.Status.Conditions, meta.ReadyCondition) {
		return fmt.Errorf("HelmRepository source reconciliation failed")
	}
	logger.Successf("fetched revision %s", repository.Status.Artifact.Revision)
	return nil
}

func helmRepositoryReconciliationHandled(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, repository *sourcev1.HelmRepository, lastHandledReconcileAt string) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, repository)
		if err != nil {
			return false, err
		}
		return repository.Status.LastHandledReconcileAt != lastHandledReconcileAt, nil
	}
}

func requestHelmRepositoryReconciliation(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, repository *sourcev1.HelmRepository) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		if err := kubeClient.Get(ctx, namespacedName, repository); err != nil {
			return err
		}
		if repository.Annotations == nil {
			repository.Annotations = map[string]string{
				meta.ReconcileRequestAnnotation: time.Now().Format(time.RFC3339Nano),
			}
		} else {
			repository.Annotations[meta.ReconcileRequestAnnotation] = time.Now().Format(time.RFC3339Nano)
		}
		return kubeClient.Update(ctx, repository)
	})
}
