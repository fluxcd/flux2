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

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var reconcileSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Reconcile a GitRepository source",
	Long:  `The reconcile source command triggers a reconciliation of a GitRepository resource and waits for it to finish.`,
	Example: `  # Trigger a git pull for an existing source
  flux reconcile source git podinfo
`,
	RunE: reconcileSourceGitCmdRun,
}

func init() {
	reconcileSourceCmd.AddCommand(reconcileSourceGitCmd)
}

func reconcileSourceGitCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("source name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	var repository sourcev1.GitRepository
	err = kubeClient.Get(ctx, namespacedName, &repository)
	if err != nil {
		return err
	}

	if repository.Spec.Suspend {
		return fmt.Errorf("resource is suspended")
	}

	logger.Actionf("annotating GitRepository source %s in %s namespace", name, namespace)
	if err := requestGitRepositoryReconciliation(ctx, kubeClient, namespacedName, &repository); err != nil {
		return err
	}
	logger.Successf("GitRepository source annotated")

	lastHandledReconcileAt := repository.Status.LastHandledReconcileAt
	logger.Waitingf("waiting for GitRepository source reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		gitRepositoryReconciliationHandled(ctx, kubeClient, namespacedName, &repository, lastHandledReconcileAt)); err != nil {
		return err
	}
	logger.Successf("GitRepository source reconciliation completed")

	if apimeta.IsStatusConditionFalse(repository.Status.Conditions, meta.ReadyCondition) {
		return fmt.Errorf("GitRepository source reconciliation failed")
	}
	logger.Successf("fetched revision %s", repository.Status.Artifact.Revision)
	return nil
}

func gitRepositoryReconciliationHandled(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, repository *sourcev1.GitRepository, lastHandledReconcileAt string) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, repository)
		if err != nil {
			return false, err
		}
		return repository.Status.LastHandledReconcileAt != lastHandledReconcileAt, nil
	}
}

func requestGitRepositoryReconciliation(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, repository *sourcev1.GitRepository) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		if err := kubeClient.Get(ctx, namespacedName, repository); err != nil {
			return err
		}
		if repository.Annotations == nil {
			repository.Annotations = map[string]string{
				meta.ReconcileAtAnnotation: time.Now().Format(time.RFC3339Nano),
			}
		} else {
			repository.Annotations[meta.ReconcileAtAnnotation] = time.Now().Format(time.RFC3339Nano)
		}
		return kubeClient.Update(ctx, repository)
	})
}
