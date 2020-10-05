/*
Copyright 2020 The Flux CD contributors.

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
	"context"
	"os"

	"github.com/fluxcd/pkg/apis/meta"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var getSourceGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Get GitRepository source statuses",
	Long:  "The get sources git command prints the status of the GitRepository sources.",
	Example: `  # List all Git repositories and their status
  gotk get sources git
`,
	RunE: getSourceGitCmdRun,
}

func init() {
	getSourceCmd.AddCommand(getSourceGitCmd)
}

func getSourceGitCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var listOpts []client.ListOption
	if !allNamespaces {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}
	var list sourcev1.GitRepositoryList
	err = kubeClient.List(ctx, &list, listOpts...)
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no git sources found in %s namespace", namespace)
		return nil
	}

	header := []string{"Name", "Revision", "Ready", "Message"}
	if allNamespaces {
		header = append([]string{"Namespace"}, header...)
	}
	var rows [][]string
	for _, source := range list.Items {
		var row []string
		var revision string
		if source.GetArtifact() != nil {
			revision = source.GetArtifact().Revision
		}
		if c := meta.GetCondition(source.Status.Conditions, meta.ReadyCondition); c != nil {
			row = []string{
				source.GetName(),
				revision,
				string(c.Status),
				c.Message,
			}
		} else {
			row = []string{
				source.GetName(),
				revision,
				string(corev1.ConditionFalse),
				"waiting to be reconciled",
			}
		}
		if allNamespaces {
			row = append([]string{source.Namespace}, row...)
		}
		rows = append(rows, row)
	}
	utils.printTable(os.Stdout, header, rows)
	return nil
}
