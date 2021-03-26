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

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/spf13/cobra"
)

var getAlertCmd = &cobra.Command{
	Use:     "alerts",
	Aliases: []string{"alert"},
	Short:   "Get Alert statuses",
	Long:    "The get alert command prints the statuses of the resources.",
	Example: `  # List all Alerts and their status
  flux get alerts`,
	RunE: getCommand{
		apiType: alertType,
		list:    &alertListAdapter{&notificationv1.AlertList{}},
	}.run,
}

func init() {
	getCmd.AddCommand(getAlertCmd)
}

func (s alertListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := s.Items[i]
	status, msg := statusAndMessage(item.Status.Conditions)
	return append(nameColumns(&item, includeNamespace, includeKind), status, msg, strings.Title(strconv.FormatBool(item.Spec.Suspend)))
}

func (s alertListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Ready", "Message", "Suspended"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}
