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
	"k8s.io/apimachinery/pkg/runtime"

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var getImagePolicyCmd = &cobra.Command{
	Use:   "image-policy",
	Short: "Get ImagePolicy statuses",
	Long:  "The get auto image-policy command prints the status of ImagePolicy objects.",
	Example: `  # List all image policies and their status
  flux get auto image-policy

 # List image policies from all namespaces
  flux get auto image-policy --all-namespaces
`,
	RunE: getCommand{
		list: &imagePolicySummary{&imagev1.ImagePolicyList{}},
	}.run,
}

func init() {
	getAutoCmd.AddCommand(getImagePolicyCmd)
}

type imagePolicySummary struct {
	*imagev1.ImagePolicyList
}

func (s imagePolicySummary) Len() int {
	return len(s.Items)
}

func (s imagePolicySummary) SummariseAt(i int, includeNamespace bool) []string {
	item := s.Items[i]
	status, msg := statusAndMessage(item.Status.Conditions)
	return append(nameColumns(&item, includeNamespace), status, msg, item.Status.LatestImage)
}

func (s imagePolicySummary) Headers(includeNamespace bool) []string {
	headers := []string{"Name", "Ready", "Message", "Latest image"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}

func (s imagePolicySummary) AsObject() runtime.Object {
	return s.ImagePolicyList
}
