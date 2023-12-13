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

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
	notificationv1b3 "github.com/fluxcd/notification-controller/api/v1beta3"
)

var getAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Get all resources and statuses",
	Long:  withPreviewNote("The get all command print the statuses of all resources."),
	Example: `  # List all resources in a namespace
  flux get all --namespace=flux-system

  # List all resources in all namespaces
  flux get all --all-namespaces`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := validateWatchOption(cmd, "all")
		if err != nil {
			return err
		}

		err = getSourceAllCmd.RunE(cmd, args)
		if err != nil {
			logError(err)
		}

		// all get command
		var allCmd = []getCommand{
			{
				apiType: helmReleaseType,
				list:    &helmReleaseListAdapter{&helmv2.HelmReleaseList{}},
			},
			{
				apiType: kustomizationType,
				list:    &kustomizationListAdapter{&kustomizev1.KustomizationList{}},
			},
			{
				apiType: receiverType,
				list:    receiverListAdapter{&notificationv1.ReceiverList{}},
			},
			{
				apiType: alertProviderType,
				list:    alertProviderListAdapter{&notificationv1b3.ProviderList{}},
			},
			{
				apiType: alertType,
				list:    &alertListAdapter{&notificationv1b3.AlertList{}},
			},
		}

		err = getImageAllCmd.RunE(cmd, args)
		if err != nil {
			logError(err)
		}

		for _, c := range allCmd {
			if err := c.run(cmd, args); err != nil {
				logError(err)
			}
		}

		return nil
	},
}

func logError(err error) {
	if !apimeta.IsNoMatchError(err) {
		logger.Failuref(err.Error())
	}
}

func init() {
	getCmd.AddCommand(getAllCmd)
}
