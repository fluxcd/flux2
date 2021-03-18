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

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/spf13/cobra"
)

var getHelmReleaseCmd = &cobra.Command{
	Use:     "helmreleases",
	Aliases: []string{"hr", "helmrelease"},
	Short:   "Get HelmRelease statuses",
	Long:    "The get helmreleases command prints the statuses of the resources.",
	Example: `  # List all Helm releases and their status
  flux get helmreleases
`,
	RunE: getCommand{
		apiType: helmReleaseType,
		list:    &helmReleaseListAdapter{&helmv2.HelmReleaseList{}},
	}.run,
}

func init() {
	getCmd.AddCommand(getHelmReleaseCmd)
}

func (a helmReleaseListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := a.Items[i]
	revision := item.Status.LastAppliedRevision
	status, msg := statusAndMessage(item.Status.Conditions)
	return append(nameColumns(&item, includeNamespace, includeKind),
		status, msg, revision, strings.Title(strconv.FormatBool(item.Spec.Suspend)))
}

func (a helmReleaseListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Ready", "Message", "Revision", "Suspended"}
	if includeNamespace {
		headers = append([]string{"Namespace"}, headers...)
	}
	return headers
}
