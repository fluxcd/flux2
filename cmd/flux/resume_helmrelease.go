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

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
)

var resumeHrCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Resume a suspended HelmRelease",
	Long: `The resume command marks a previously suspended HelmRelease resource for reconciliation and waits for it to
finish the apply.`,
	Example: `  # Resume reconciliation for an existing Helm release
  flux resume hr podinfo
`,
	RunE: resumeHrCmdRun,
}

func init() {
	resumeCmd.AddCommand(resumeHrCmd)
}

func resumeHrCmdRun(cmd *cobra.Command, args []string) error {
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

	logger.Actionf("resuming HelmRelease %s in %s namespace", name, namespace)
	helmRelease.Spec.Suspend = false
	if err := kubeClient.Update(ctx, &helmRelease); err != nil {
		return err
	}
	logger.Successf("HelmRelease resumed")

	logger.Waitingf("waiting for HelmRelease reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isHelmReleaseResumed(ctx, kubeClient, namespacedName, &helmRelease)); err != nil {
		return err
	}
	logger.Successf("HelmRelease reconciliation completed")

	logger.Successf("applied revision %s", helmRelease.Status.LastAppliedRevision)
	return nil
}

func isHelmReleaseResumed(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, helmRelease *helmv2.HelmRelease) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, helmRelease)
		if err != nil {
			return false, err
		}

		// Confirm the state we are observing is for the current generation
		if helmRelease.Generation != helmRelease.Status.ObservedGeneration {
			return false, err
		}

		if c := meta.GetCondition(helmRelease.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case corev1.ConditionTrue:
				return true, nil
			case corev1.ConditionFalse:
				if c.Reason == meta.SuspendedReason {
					return false, nil
				}
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}
