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

	autov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
)

var getImageAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Get all image statuses",
	Long:  withPreviewNote("The get image sub-commands print the statuses of all image objects."),
	Example: `  # List all image objects in a namespace
  flux get images all --namespace=flux-system

  # List all image objects in all namespaces
  flux get images all --all-namespaces`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := validateWatchOption(cmd, "all")
		if err != nil {
			return err
		}

		var allImageCmd = []getCommand{
			{
				apiType: imageRepositoryType,
				list:    imageRepositoryListAdapter{&imagev1.ImageRepositoryList{}},
			},
			{
				apiType: imagePolicyType,
				list:    &imagePolicyListAdapter{&imagev1.ImagePolicyList{}},
			},
			{
				apiType: imageUpdateAutomationType,
				list:    &imageUpdateAutomationListAdapter{&autov1.ImageUpdateAutomationList{}},
			},
		}

		for _, c := range allImageCmd {
			if err := c.run(cmd, args); err != nil {
				logger.Failuref(err.Error())
			}
		}

		return nil
	},
}

func init() {
	getImageCmd.AddCommand(getImageAllCmd)
}
