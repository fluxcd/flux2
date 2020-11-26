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
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var resumeSourceHelmChartCmd = &cobra.Command{
	Use:   "chart [name]",
	Short: "Resume a suspended HelmChart",
	Long:  `The resume command marks a previously suspended HelmChart resource for reconciliation and waits for it to finish.`,
	Example: `  # Resume reconciliation for an existing HelmChart
  flux resume source chart podinfo
`,
	RunE: resumeSourceHelmChartCmdRun,
}

func init() {
	resumeSourceCmd.AddCommand(resumeSourceHelmChartCmd)
}

func resumeSourceHelmChartCmdRun(cmd *cobra.Command, args []string) error {
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
	var repository sourcev1.HelmChart
	err = kubeClient.Get(ctx, namespacedName, &repository)
	if err != nil {
		return err
	}

	logger.Actionf("resuming source %s in %s namespace", name, namespace)
	repository.Spec.Suspend = false
	if err := kubeClient.Update(ctx, &repository); err != nil {
		return err
	}
	logger.Successf("source resumed")

	logger.Waitingf("waiting for HelmChart reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isHelmChartResumed(ctx, kubeClient, namespacedName, &repository)); err != nil {
		return err
	}
	logger.Successf("HelmChart reconciliation completed")

	logger.Successf("fetched revision %s", repository.Status.Artifact.Revision)
	return nil
}

func isHelmChartResumed(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, chart *sourcev1.HelmChart) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, chart)
		if err != nil {
			return false, err
		}

		// Confirm the state we are observing is for the current generation
		if chart.Generation != chart.Status.ObservedGeneration {
			return false, nil
		}

		if c := apimeta.FindStatusCondition(chart.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case metav1.ConditionTrue:
				return true, nil
			case metav1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}
