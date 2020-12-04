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
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var getImageRepositoryCmd = &cobra.Command{
	Use:   "image-repository",
	Short: "Get ImageRepository statuses",
	Long:  "The get auto image-repository command prints the status of ImageRepository objects.",
	Example: `  # List all image repositories and their status
  flux get auto image-repository

 # List image repositories from all namespaces
  flux get auto image-repository --all-namespaces
`,
	RunE: getCommand{
		list: imageRepositorySummary{&imagev1.ImageRepositoryList{}},
	}.run,
}

func init() {
	getAutoCmd.AddCommand(getImageRepositoryCmd)
}

type imageRepositorySummary struct {
	*imagev1.ImageRepositoryList
}

func (s imageRepositorySummary) Len() int {
	return len(s.Items)
}

func (s imageRepositorySummary) SummariseAt(i int, includeNamespace bool) []string {
	item := s.Items[i]
	status, msg := statusAndMessage(item.Status.Conditions)
	var lastScan string
	if item.Status.LastScanResult != nil {
		lastScan = item.Status.LastScanResult.ScanTime.Time.Format(time.RFC3339)
	}
	return append(nameColumns(&item, includeNamespace),
		status, msg, lastScan, strings.Title(strconv.FormatBool(item.Spec.Suspend)))
}

func (s imageRepositorySummary) Headers(includeNamespace bool) []string {
	headers := []string{"Name", "Ready", "Message", "Last scan", "Suspended"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}

func (s imageRepositorySummary) AsObject() runtime.Object {
	return s.ImageRepositoryList
}
