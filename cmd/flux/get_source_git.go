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

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/spf13/cobra"
)

var getSourceGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Get GitRepository source statuses",
	Long:  "The get sources git command prints the status of the GitRepository sources.",
	Example: `  # List all Git repositories and their status
  flux get sources git

 # List Git repositories from all namespaces
  flux get sources git --all-namespaces`,
	RunE: getCommand{
		apiType: gitRepositoryType,
		list:    &gitRepositoryListAdapter{&sourcev1.GitRepositoryList{}},
	}.run,
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
