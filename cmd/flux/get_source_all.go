/*
Copyright 2021 The Flux authors

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
	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"
)

var getSourceAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Get all source statuses",
	Long:  withPreviewNote("The get sources all command print the statuses of all sources."),
	Example: `  # List all sources in a namespace
  flux get sources all --namespace=flux-system

  # List all sources in all namespaces
  flux get sources all --all-namespaces`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := validateWatchOption(cmd, "all")
		if err != nil {
			return err
		}

		var allSourceCmd = []getCommand{
			{
				apiType: ociRepositoryType,
				list:    &ociRepositoryListAdapter{&sourcev1b2.OCIRepositoryList{}},
			},
			{
				apiType: bucketType,
				list:    &bucketListAdapter{&sourcev1b2.BucketList{}},
			},
			{
				apiType: gitRepositoryType,
				list:    &gitRepositoryListAdapter{&sourcev1.GitRepositoryList{}},
			},
			{
				apiType: helmRepositoryType,
				list:    &helmRepositoryListAdapter{&sourcev1b2.HelmRepositoryList{}},
			},
			{
				apiType: helmChartType,
				list:    &helmChartListAdapter{&sourcev1b2.HelmChartList{}},
			},
		}

		for _, c := range allSourceCmd {
			if err := c.run(cmd, args); err != nil {
				if !apimeta.IsNoMatchError(err) {
					logger.Failuref(err.Error())
				}
			}
		}

		return nil
	},
}

func init() {
	getSourceCmd.AddCommand(getSourceAllCmd)
}
