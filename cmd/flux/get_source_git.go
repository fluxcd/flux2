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
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var getSourceGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Get GitRepository source statuses",
	Long:  "The get sources git command prints the status of the GitRepository sources.",
	Example: `  # List all Git repositories and their status
  flux get sources git

 # List Git repositories from all namespaces
  flux get sources git --all-namespaces`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.GitRepositoryKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: gitRepositoryType,
			list:    &gitRepositoryListAdapter{&sourcev1.GitRepositoryList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*sourcev1.GitRepository)
			if !ok {
				return nil, fmt.Errorf("Impossible to cast type %#v git", obj)
			}

			sink := &gitRepositoryListAdapter{&sourcev1.GitRepositoryList{
				Items: []sourcev1.GitRepository{
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
	getSourceCmd.AddCommand(getSourceGitCmd)
}

func (a *gitRepositoryListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := a.Items[i]
	var revision string
	if item.GetArtifact() != nil {
		revision = item.GetArtifact().Revision
	}
	status, msg := statusAndMessage(item.Status.Conditions)
	if status == string(metav1.ConditionTrue) {
		revision = shortenCommitSha(revision)
		msg = shortenCommitSha(msg)
	}
	return append(nameColumns(&item, includeNamespace, includeKind),
		status, msg, revision, strings.Title(strconv.FormatBool(item.Spec.Suspend)))
}

func (a gitRepositoryListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Ready", "Message", "Revision", "Suspended"}
	if includeNamespace {
		headers = append([]string{"Namespace"}, headers...)
	}
	return headers
}

func (a gitRepositoryListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := a.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
