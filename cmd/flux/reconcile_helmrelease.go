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

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var reconcileHrCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Reconcile a HelmRelease resource",
	Long: `
The reconcile kustomization command triggers a reconciliation of a HelmRelease resource and waits for it to finish.`,
	Example: `  # Trigger a HelmRelease apply outside of the reconciliation interval
  flux reconcile hr podinfo

  # Trigger a reconciliation of the HelmRelease's source and apply changes
  flux reconcile hr podinfo --with-source
`,
	RunE: reconcileHrCmdRun,
}

var (
	syncHrWithSource bool
)

func init() {
	reconcileHrCmd.Flags().BoolVar(&syncHrWithSource, "with-source", false, "reconcile HelmRelease source")

	reconcileCmd.AddCommand(reconcileHrCmd)
}

func reconcileHrCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("HelmRelease name is required")
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

	var helmRelease helmv2.HelmRelease
	err = kubeClient.Get(ctx, namespacedName, &helmRelease)
	if err != nil {
		return err
	}

	if helmRelease.Spec.Suspend {
		return fmt.Errorf("resource is suspended")
	}

	if syncHrWithSource {
		switch helmRelease.Spec.Chart.Spec.SourceRef.Kind {
		case sourcev1.HelmRepositoryKind:
			err = reconcileSourceHelmCmdRun(nil, []string{helmRelease.Spec.Chart.Spec.SourceRef.Name})
		case sourcev1.GitRepositoryKind:
			err = reconcileSourceGitCmdRun(nil, []string{helmRelease.Spec.Chart.Spec.SourceRef.Name})
		case sourcev1.BucketKind:
			err = reconcileSourceBucketCmdRun(nil, []string{helmRelease.Spec.Chart.Spec.SourceRef.Name})
		}
		if err != nil {
			return err
		}
	}

	lastHandledReconcileAt := helmRelease.Status.LastHandledReconcileAt
	logger.Actionf("annotating HelmRelease %s in %s namespace", name, namespace)
	if err := requestHelmReleaseReconciliation(ctx, kubeClient, namespacedName, &helmRelease); err != nil {
		return err
	}
	logger.Successf("HelmRelease annotated")

	logger.Waitingf("waiting for HelmRelease reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		helmReleaseReconciliationHandled(ctx, kubeClient, namespacedName, &helmRelease, lastHandledReconcileAt),
	); err != nil {
		return err
	}
	logger.Successf("HelmRelease reconciliation completed")

	err = kubeClient.Get(ctx, namespacedName, &helmRelease)
	if err != nil {
		return err
	}
	if c := apimeta.FindStatusCondition(helmRelease.Status.Conditions, meta.ReadyCondition); c != nil {
		switch c.Status {
		case metav1.ConditionFalse:
			return fmt.Errorf("HelmRelease reconciliation failed: %s", c.Message)
		default:
			logger.Successf("reconciled revision %s", helmRelease.Status.LastAppliedRevision)
		}
	}
	return nil
}

func helmReleaseReconciliationHandled(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, helmRelease *helmv2.HelmRelease, lastHandledReconcileAt string) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, helmRelease)
		if err != nil {
			return false, err
		}
		return helmRelease.Status.LastHandledReconcileAt != lastHandledReconcileAt, nil
	}
}

func requestHelmReleaseReconciliation(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, helmRelease *helmv2.HelmRelease) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		if err := kubeClient.Get(ctx, namespacedName, helmRelease); err != nil {
			return err
		}
		if helmRelease.Annotations == nil {
			helmRelease.Annotations = map[string]string{
				meta.ReconcileRequestAnnotation: time.Now().Format(time.RFC3339Nano),
			}
		} else {
			helmRelease.Annotations[meta.ReconcileRequestAnnotation] = time.Now().Format(time.RFC3339Nano)
		}
		return kubeClient.Update(ctx, helmRelease)
	})
}
