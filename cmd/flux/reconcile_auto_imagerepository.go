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

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var reconcileImageRepositoryCmd = &cobra.Command{
	Use:   "image-repository [name]",
	Short: "Reconcile an ImageRepository",
	Long:  `The reconcile auto image-repository command triggers a reconciliation of an ImageRepository resource and waits for it to finish.`,
	Example: `  # Trigger an scan for an existing image repository
  flux reconcile auto image-repository alpine
`,
	RunE: reconcileImageRepositoryRun,
}

func init() {
	reconcileAutoCmd.AddCommand(reconcileImageRepositoryCmd)
}

func reconcileImageRepositoryRun(cmd *cobra.Command, args []string) error {
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
	var repo imagev1.ImageRepository
	err = kubeClient.Get(ctx, namespacedName, &repo)
	if err != nil {
		return err
	}

	if repo.Spec.Suspend {
		return fmt.Errorf("resource is suspended")
	}

	logger.Actionf("annotating ImageRepository %s in %s namespace", name, namespace)
	if err := requestImageRepositoryReconciliation(ctx, kubeClient, namespacedName, &repo); err != nil {
		return err
	}
	logger.Successf("ImageRepository annotated")

	lastHandledReconcileAt := repo.Status.LastHandledReconcileAt
	logger.Waitingf("waiting for ImageRepository reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		imageRepositoryReconciliationHandled(ctx, kubeClient, namespacedName, &repo, lastHandledReconcileAt)); err != nil {
		return err
	}
	logger.Successf("ImageRepository reconciliation completed")

	if apimeta.IsStatusConditionFalse(repo.Status.Conditions, meta.ReadyCondition) {
		return fmt.Errorf("ImageRepository reconciliation failed")
	}
	logger.Successf("scan fetched %d tags", repo.Status.LastScanResult.TagCount)
	return nil
}

func imageRepositoryReconciliationHandled(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, repo *imagev1.ImageRepository, lastHandledReconcileAt string) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, repo)
		if err != nil {
			return false, err
		}
		return repo.Status.LastHandledReconcileAt != lastHandledReconcileAt, nil
	}
}

func requestImageRepositoryReconciliation(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, repo *imagev1.ImageRepository) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		if err := kubeClient.Get(ctx, namespacedName, repo); err != nil {
			return err
		}
		if repo.Annotations == nil {
			repo.Annotations = map[string]string{
				meta.ReconcileAtAnnotation: time.Now().Format(time.RFC3339Nano),
			}
		} else {
			repo.Annotations[meta.ReconcileAtAnnotation] = time.Now().Format(time.RFC3339Nano)
		}
		return kubeClient.Update(ctx, repo)
	})
}
