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

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
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

	logger.Actionf("annotating HelmRepository source %s in %s namespace", name, namespace)
	var helmRepository sourcev1.HelmRepository
	err = kubeClient.Get(ctx, namespacedName, &helmRepository)
	if err != nil {
		return err
	}

	if helmRepository.Annotations == nil {
		helmRepository.Annotations = map[string]string{
			meta.ReconcileAtAnnotation: time.Now().Format(time.RFC3339Nano),
		}
	} else {
		helmRepository.Annotations[meta.ReconcileAtAnnotation] = time.Now().Format(time.RFC3339Nano)
	}
	if err := kubeClient.Update(ctx, &helmRepository); err != nil {
		return err
	}
	logger.Successf("HelmRepository source annotated")

	logger.Waitingf("waiting for HelmRepository source reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isHelmRepositoryReady(ctx, kubeClient, namespacedName, &helmRepository)); err != nil {
		return err
	}
	logger.Successf("HelmRepository source reconciliation completed")

	if helmRepository.Status.Artifact == nil {
		return fmt.Errorf("HelmRepository source reconciliation completed but no artifact was found")
	}
	logger.Successf("fetched revision %s", helmRepository.Status.Artifact.Revision)
	return nil
}

func isHelmRepositoryReady(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, helmRepository *sourcev1.HelmRepository) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, helmRepository)
		if err != nil {
			return false, err
		}

		// Confirm the state we are observing is for the current generation
		if helmRepository.Generation != helmRepository.Status.ObservedGeneration {
			return false, nil
		}

		if c := meta.GetCondition(helmRepository.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case corev1.ConditionTrue:
				return true, nil
			case corev1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}
