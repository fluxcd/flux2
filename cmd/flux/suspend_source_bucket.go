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
	"github.com/spf13/cobra"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var suspendSourceBucketCmd = &cobra.Command{
	Use:   "bucket [name]",
	Short: "Suspend reconciliation of a Bucket",
	Long:  `The suspend command disables the reconciliation of a Bucket resource.`,
	Example: `  # Suspend reconciliation for an existing Bucket
  flux suspend source bucket podinfo

  # Suspend reconciliation for multiple Buckets
  flux suspend source bucket podinfo-1 podinfo-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.BucketKind)),
	RunE: suspendCommand{
		apiType: bucketType,
		object:  bucketAdapter{&sourcev1.Bucket{}},
		list:    bucketListAdapter{&sourcev1.BucketList{}},
	}.run,
}

func init() {
	suspendSourceCmd.AddCommand(suspendSourceBucketCmd)
}

func (obj bucketAdapter) isSuspended() bool {
	return obj.Bucket.Spec.Suspend
}

func (obj bucketAdapter) setSuspended() {
	obj.Bucket.Spec.Suspend = true
}

func (a bucketListAdapter) item(i int) suspendable {
	return &bucketAdapter{&a.BucketList.Items[i]}
}
