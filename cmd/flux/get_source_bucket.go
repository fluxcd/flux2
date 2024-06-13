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
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/apimachinery/pkg/runtime"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var getSourceBucketCmd = &cobra.Command{
	Use:   "bucket",
	Short: "Get Bucket source statuses",
	Long:  withPreviewNote("The get sources bucket command prints the status of the Bucket sources."),
	Example: `  # List all Buckets and their status
  flux get sources bucket

 # List buckets from all namespaces
  flux get sources helm --all-namespaces`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.BucketKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: bucketType,
			list:    &bucketListAdapter{&sourcev1.BucketList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*sourcev1.Bucket)
			if !ok {
				return nil, fmt.Errorf("Impossible to cast type %#v bucket", obj)
			}

			sink := &bucketListAdapter{&sourcev1.BucketList{
				Items: []sourcev1.Bucket{
					*o,
				}}}
			return sink, nil
		})

		if err != nil {
			return err
		}

		if err := get.run(cmd, args); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	getSourceCmd.AddCommand(getSourceBucketCmd)
}

func (a *bucketListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := a.Items[i]
	var revision string
	if item.GetArtifact() != nil {
		revision = item.GetArtifact().Revision
	}
	status, msg := statusAndMessage(item.Status.Conditions)
	revision = utils.TruncateHex(revision)
	msg = utils.TruncateHex(msg)
	return append(nameColumns(&item, includeNamespace, includeKind),
		revision, cases.Title(language.English).String(strconv.FormatBool(item.Spec.Suspend)), status, msg)
}

func (a bucketListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Revision", "Suspended", "Ready", "Message"}
	if includeNamespace {
		headers = append([]string{"Namespace"}, headers...)
	}
	return headers
}

func (a bucketListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := a.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
