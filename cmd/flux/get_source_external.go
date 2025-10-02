/*
Copyright 2025 The Flux authors

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

	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var getSourceExternalCmd = &cobra.Command{
	Use:   "external",
	Short: "Get ExternalArtifact source statuses",
	Long:  `The get sources external command prints the status of the ExternalArtifact sources.`,
	Example: `  # List all ExternalArtifacts and their status
  flux get sources external

  # List ExternalArtifacts from all namespaces
  flux get sources external --all-namespaces`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.ExternalArtifactKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: externalArtifactType,
			list:    &externalArtifactListAdapter{&sourcev1.ExternalArtifactList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*sourcev1.ExternalArtifact)
			if !ok {
				return nil, fmt.Errorf("impossible to cast type %#v to ExternalArtifact", obj)
			}

			sink := &externalArtifactListAdapter{&sourcev1.ExternalArtifactList{
				Items: []sourcev1.ExternalArtifact{
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
	getSourceCmd.AddCommand(getSourceExternalCmd)
}

func (a *externalArtifactListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := a.Items[i]
	var revision string
	if item.Status.Artifact != nil {
		revision = item.Status.Artifact.Revision
	}
	status, msg := statusAndMessage(item.Status.Conditions)
	revision = utils.TruncateHex(revision)
	msg = utils.TruncateHex(msg)

	var source string
	if item.Spec.SourceRef != nil {
		source = fmt.Sprintf("%s/%s/%s",
			item.Spec.SourceRef.Kind,
			item.Spec.SourceRef.Namespace,
			item.Spec.SourceRef.Name)
	}
	return append(nameColumns(&item, includeNamespace, includeKind),
		revision, source, status, msg)
}

func (a externalArtifactListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Revision", "Source", "Ready", "Message"}
	if includeNamespace {
		headers = append([]string{"Namespace"}, headers...)
	}
	return headers
}

func (a externalArtifactListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := a.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
