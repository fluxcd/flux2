/*
Copyright 2022 The Flux authors

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

var getSourceOCIRepositoryCmd = &cobra.Command{
	Use:   "oci",
	Short: "Get OCIRepository status",
	Long:  withPreviewNote("The get sources oci command prints the status of the OCIRepository sources."),
	Example: `  # List all OCIRepositories and their status
  flux get sources oci

  # List OCIRepositories from all namespaces
  flux get sources oci --all-namespaces`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.OCIRepositoryKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: ociRepositoryType,
			list:    &ociRepositoryListAdapter{&sourcev1.OCIRepositoryList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*sourcev1.OCIRepository)
			if !ok {
				return nil, fmt.Errorf("impossible to cast type %#v to OCIRepository", obj)
			}

			sink := &ociRepositoryListAdapter{&sourcev1.OCIRepositoryList{
				Items: []sourcev1.OCIRepository{
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
	getSourceCmd.AddCommand(getSourceOCIRepositoryCmd)
}

func (a *ociRepositoryListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
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

func (a ociRepositoryListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Revision", "Suspended", "Ready", "Message"}
	if includeNamespace {
		headers = append([]string{"Namespace"}, headers...)
	}
	return headers
}

func (a ociRepositoryListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := a.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
