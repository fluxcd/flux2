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

	autov1 "github.com/fluxcd/image-automation-controller/api/v1alpha1"
)

var getImageUpdateCmd = &cobra.Command{
	Use:   "image-update",
	Short: "Get ImageUpdateAutomation statuses",
	Long:  "The get auto image-update command prints the status of ImageUpdateAutomation objects.",
	Example: `  # List all image update automation object and their status
  flux get auto image-update

 # List image update automations from all namespaces
  flux get auto image-update --all-namespaces
`,
	RunE: getCommand{
		list: &imageUpdateSummary{&autov1.ImageUpdateAutomationList{}},
	}.run,
}

func init() {
	getAutoCmd.AddCommand(getImageUpdateCmd)
}

type imageUpdateSummary struct {
	*autov1.ImageUpdateAutomationList
}

func (s imageUpdateSummary) Len() int {
	return len(s.Items)
}

func (s imageUpdateSummary) SummariseAt(i int, includeNamespace bool) []string {
	item := s.Items[i]
	status, msg := statusAndMessage(item.Status.Conditions)
	var lastRun string
	if item.Status.LastAutomationRunTime != nil {
		lastRun = item.Status.LastAutomationRunTime.Time.Format(time.RFC3339)
	}
	return append(nameColumns(&item, includeNamespace), status, msg, lastRun, strings.Title(strconv.FormatBool(item.Spec.Suspend)))
}

func (s imageUpdateSummary) Headers(includeNamespace bool) []string {
	headers := []string{"Name", "Ready", "Message", "Last run", "Suspended"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}

func (s imageUpdateSummary) AsClientObject() runtime.Object {
	return s.ImageUpdateAutomationList
}
