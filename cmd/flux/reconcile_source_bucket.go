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

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var reconcileSourceBucketCmd = &cobra.Command{
	Use:   "bucket [name]",
	Short: "Reconcile a Bucket source",
	Long:  `The reconcile source command triggers a reconciliation of a Bucket resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing source
  flux reconcile source bucket podinfo`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.BucketKind)),
	RunE: reconcileCommand{
		apiType: bucketType,
		object:  bucketAdapter{&sourcev1.Bucket{}},
	}.run,
}

func init() {
	reconcileSourceCmd.AddCommand(reconcileSourceBucketCmd)
}

func isBucketReady(ctx context.Context, kubeClient client.Client,
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

func (obj bucketAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}

func (obj bucketAdapter) successMessage() string {
	return fmt.Sprintf("fetched revision %s", obj.Status.Artifact.Revision)
}
