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

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
)

var getHelmReleaseCmd = &cobra.Command{
	Use:     "helmreleases",
	Aliases: []string{"hr", "helmrelease"},
	Short:   "Get HelmRelease statuses",
	Long:    withPreviewNote("The get helmreleases command prints the statuses of the resources."),
	Example: `  # List all Helm releases and their status
  flux get helmreleases`,
	ValidArgsFunction: resourceNamesCompletionFunc(helmv2.GroupVersion.WithKind(helmv2.HelmReleaseKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: helmReleaseType,
			list:    &helmReleaseListAdapter{&helmv2.HelmReleaseList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*helmv2.HelmRelease)
			if !ok {
				return nil, fmt.Errorf("Impossible to cast type %#v helmrelease", obj)
			}

			sink := helmReleaseListAdapter{&helmv2.HelmReleaseList{
				Items: []helmv2.HelmRelease{
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
	getCmd.AddCommand(getHelmReleaseCmd)
}

func (a helmReleaseListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := a.Items[i]
	revision := item.Status.LastAppliedRevision
	status, msg := statusAndMessage(item.Status.Conditions)
	return append(nameColumns(&item, includeNamespace, includeKind),
		revision, cases.Title(language.English).String(strconv.FormatBool(item.Spec.Suspend)), status, msg)
}

func (a helmReleaseListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Revision", "Suspended", "Ready", "Message"}
	if includeNamespace {
		headers = append([]string{"Namespace"}, headers...)
	}
	return headers
}

func (a helmReleaseListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := a.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
