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

var resumeSourceBucketCmd = &cobra.Command{
	Use:   "bucket [name]",
	Short: "Resume a suspended Bucket",
	Long:  `The resume command marks a previously suspended Bucket resource for reconciliation and waits for it to finish.`,
	Example: `  # Resume reconciliation for an existing Bucket
  flux resume source bucket podinfo
`,
	RunE: resumeSourceBucketCmdRun,
}

func init() {
	resumeSourceCmd.AddCommand(resumeSourceBucketCmd)
}

func resumeSourceBucketCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("source name is required")
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
	var bucket sourcev1.Bucket
	err = kubeClient.Get(ctx, namespacedName, &bucket)
	if err != nil {
		return err
	}

	logger.Actionf("resuming source %s in %s namespace", name, rootArgs.namespace)
	bucket.Spec.Suspend = false
	if err := kubeClient.Update(ctx, &bucket); err != nil {
		return err
	}
	logger.Successf("source resumed")

	logger.Waitingf("waiting for Bucket reconciliation")
	if err := wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
		isBucketResumed(ctx, kubeClient, namespacedName, &bucket)); err != nil {
		return err
	}
	logger.Successf("Bucket reconciliation completed")

	logger.Successf("fetched revision %s", bucket.Status.Artifact.Revision)
	return nil
}

func isBucketResumed(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, bucket *sourcev1.Bucket) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, bucket)
		if err != nil {
			return false, err
		}

		// Confirm the state we are observing is for the current generation
		if bucket.Generation != bucket.Status.ObservedGeneration {
			return false, nil
		}

		if c := apimeta.FindStatusCondition(bucket.Status.Conditions, meta.ReadyCondition); c != nil {
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
