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

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
)

var getImagePolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Get ImagePolicy status",
	Long:  withPreviewNote("The get image policy command prints the status of ImagePolicy objects."),
	Example: `  # List all image policies and their status
  flux get image policy

 # List image policies from all namespaces
  flux get image policy --all-namespaces`,
	ValidArgsFunction: resourceNamesCompletionFunc(imagev1.GroupVersion.WithKind(imagev1.ImagePolicyKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: imagePolicyType,
			list:    &imagePolicyListAdapter{&imagev1.ImagePolicyList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*imagev1.ImagePolicy)
			if !ok {
				return nil, fmt.Errorf("Impossible to cast type %#v policy", obj)
			}

			sink := imagePolicyListAdapter{&imagev1.ImagePolicyList{
				Items: []imagev1.ImagePolicy{
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
	getImageCmd.AddCommand(getImagePolicyCmd)
}

func (s imagePolicyListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := s.Items[i]
	status, msg := statusAndMessage(item.Status.Conditions)
	return append(nameColumns(&item, includeNamespace, includeKind), item.Status.LatestImage, status, msg)
}

func (s imagePolicyListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Latest image", "Ready", "Message"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}

func (s imagePolicyListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := s.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
