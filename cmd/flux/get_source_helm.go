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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var getSourceHelmCmd = &cobra.Command{
	Use:   "helm",
	Short: "Get HelmRepository source statuses",
	Long:  withPreviewNote("The get sources helm command prints the status of the HelmRepository sources."),
	Example: `  # List all Helm repositories and their status
  flux get sources helm

 # List Helm repositories from all namespaces
  flux get sources helm --all-namespaces`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.HelmRepositoryKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: helmRepositoryType,
			list:    &helmRepositoryListAdapter{&sourcev1.HelmRepositoryList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*sourcev1.HelmRepository)
			if !ok {
				return nil, fmt.Errorf("Impossible to cast type %#v helm", obj)
			}

			sink := &helmRepositoryListAdapter{&sourcev1.HelmRepositoryList{
				Items: []sourcev1.HelmRepository{
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
	getSourceCmd.AddCommand(getSourceHelmCmd)
}

func (a *helmRepositoryListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := a.Items[i]
	var revision string
	if item.GetArtifact() != nil {
		revision = item.GetArtifact().Revision
	}
	var status, msg string
	if item.Spec.Type == sourcev1.HelmRepositoryTypeOCI {
		status, msg = string(metav1.ConditionTrue), "Helm repository is Ready"
	} else {
		status, msg = statusAndMessage(item.Status.Conditions)
	}
	revision = utils.TruncateHex(revision)
	msg = utils.TruncateHex(msg)
	return append(nameColumns(&item, includeNamespace, includeKind),
		revision, cases.Title(language.English).String(strconv.FormatBool(item.Spec.Suspend)), status, msg)
}

func (a helmRepositoryListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Revision", "Suspended", "Ready", "Message"}
	if includeNamespace {
		headers = append([]string{"Namespace"}, headers...)
	}
	return headers
}

func (a helmRepositoryListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := a.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
