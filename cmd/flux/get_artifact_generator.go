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
	"strconv"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/apimachinery/pkg/runtime"

	swapi "github.com/fluxcd/source-watcher/api/v2/v1beta1"
)

var getArtifactGeneratorCmd = &cobra.Command{
	Use:     "generators",
	Aliases: []string{"generator"},
	Short:   "Get artifact generator statuses",
	Long:    `The get artifact generator command prints the statuses of the resources.`,
	Example: `  # List all ArtifactGenerators and their status
  flux get artifact generators`,
	ValidArgsFunction: resourceNamesCompletionFunc(swapi.GroupVersion.WithKind(swapi.ArtifactGeneratorKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: receiverType,
			list:    artifactGeneratorListAdapter{&swapi.ArtifactGeneratorList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*swapi.ArtifactGenerator)
			if !ok {
				return nil, fmt.Errorf("impossible to cast type %#v generator", obj)
			}

			sink := artifactGeneratorListAdapter{&swapi.ArtifactGeneratorList{
				Items: []swapi.ArtifactGenerator{
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
	getArtifactCmd.AddCommand(getArtifactGeneratorCmd)
}

func (s artifactGeneratorListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := s.Items[i]
	status, msg := statusAndMessage(item.Status.Conditions)
	return append(nameColumns(&item, includeNamespace, includeKind),
		cases.Title(language.English).String(strconv.FormatBool(item.IsDisabled())), status, msg)
}

func (s artifactGeneratorListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Suspended", "Ready", "Message"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}

func (s artifactGeneratorListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := s.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
