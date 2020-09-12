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
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2alpha1"
	consts "github.com/fluxcd/pkg/runtime"
	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
)

var reconcileHrCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Reconcile a HelmRelease resource",
	Long: `
The reconcile kustomization command triggers a reconciliation of a HelmRelease resource and waits for it to finish.`,
	Example: `  # Trigger a HelmRelease apply outside of the reconciliation interval
  gotk reconcile hr podinfo

  # Trigger a reconciliation of the HelmRelease's source and apply changes
  gotk reconcile hr podinfo --with-source
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

	kubeClient, err := utils.kubeClient(kubeconfig)
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

	if syncHrWithSource {
		switch helmRelease.Spec.Chart.Spec.SourceRef.Kind {
		case sourcev1.HelmRepositoryKind:
			err = syncSourceHelmCmdRun(nil, []string{helmRelease.Spec.Chart.Spec.SourceRef.Name})
		case sourcev1.GitRepositoryKind:
			err = syncSourceGitCmdRun(nil, []string{helmRelease.Spec.Chart.Spec.SourceRef.Name})
		}
		if err != nil {
			return err
		}
	} else {
		logger.Actionf("annotating HelmRelease %s in %s namespace", name, namespace)
		if helmRelease.Annotations == nil {
			helmRelease.Annotations = map[string]string{
				consts.ReconcileAtAnnotation: time.Now().Format(time.RFC3339Nano),
			}
		} else {
			helmRelease.Annotations[consts.ReconcileAtAnnotation] = time.Now().Format(time.RFC3339Nano)
		}
		if err := kubeClient.Update(ctx, &helmRelease); err != nil {
			return err
		}
		logger.Successf("HelmRelease annotated")
	}

	logger.Waitingf("waiting for HelmRelease reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isHelmReleaseReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("HelmRelease reconciliation completed")

	err = kubeClient.Get(ctx, namespacedName, &helmRelease)
	if err != nil {
		return err
	}

	if helmRelease.Status.LastAppliedRevision != "" {
		logger.Successf("reconciled revision %s", helmRelease.Status.LastAppliedRevision)
	} else {
		return fmt.Errorf("HelmRelease reconciliation failed")
	}
	return nil
}

func isHelmReleaseReady(ctx context.Context, kubeClient client.Client, name, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		var helmRelease helmv2.HelmRelease
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := kubeClient.Get(ctx, namespacedName, &helmRelease)
		if err != nil {
			return false, err
		}

		for _, condition := range helmRelease.Status.Conditions {
			if condition.Type == helmv2.ReadyCondition {
				if condition.Status == corev1.ConditionTrue {
					return true, nil
				} else if condition.Status == corev1.ConditionFalse && helmRelease.Status.LastAttemptedRevision != "" {
					return false, fmt.Errorf(condition.Message)
				}
			}
		}
		return false, nil
	}
}
