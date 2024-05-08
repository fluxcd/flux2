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
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/apimachinery/pkg/runtime"

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
)

var getImageRepositoryCmd = &cobra.Command{
	Use:   "repository",
	Short: "Get ImageRepository status",
	Long:  withPreviewNote("The get image repository command prints the status of ImageRepository objects."),
	Example: `  # List all image repositories and their status
  flux get image repository

 # List image repositories from all namespaces
  flux get image repository --all-namespaces`,
	ValidArgsFunction: resourceNamesCompletionFunc(imagev1.GroupVersion.WithKind(imagev1.ImageRepositoryKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: imageRepositoryType,
			list:    imageRepositoryListAdapter{&imagev1.ImageRepositoryList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*imagev1.ImageRepository)
			if !ok {
				return nil, fmt.Errorf("Impossible to cast type %#v repository", obj)
			}

			sink := imageRepositoryListAdapter{&imagev1.ImageRepositoryList{
				Items: []imagev1.ImageRepository{
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
	getImageCmd.AddCommand(getImageRepositoryCmd)
}

func (s imageRepositoryListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := s.Items[i]
	status, msg := statusAndMessage(item.Status.Conditions)
	var lastScan string
	if item.Status.LastScanResult != nil {
		lastScan = item.Status.LastScanResult.ScanTime.Time.Format(time.RFC3339)
	}
	return append(nameColumns(&item, includeNamespace, includeKind),
		lastScan, cases.Title(language.English).String(strconv.FormatBool(item.Spec.Suspend)), status, msg)
}

func (s imageRepositoryListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Last scan", "Suspended", "Ready", "Message"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}

func (s imageRepositoryListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := s.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
