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
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/toolkit/internal/utils"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
)

var getHelmReleaseCmd = &cobra.Command{
	Use:     "helmreleases",
	Aliases: []string{"hr"},
	Short:   "Get HelmRelease statuses",
	Long:    "The get helmreleases command prints the statuses of the resources.",
	Example: `  # List all Helm releases and their status
  flux get helmreleases
`,
	RunE: getHelmReleaseCmdRun,
}

func init() {
	getCmd.AddCommand(getHelmReleaseCmd)
}

func getHelmReleaseCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var listOpts []client.ListOption
	if !allNamespaces {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}
	var list helmv2.HelmReleaseList
	err = kubeClient.List(ctx, &list, listOpts...)
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no releases found in %s namespace", namespace)
		return nil
	}

	header := []string{"Name", "Revision", "Suspended", "Ready", "Message"}
	if allNamespaces {
		header = append([]string{"Namespace"}, header...)
	}
	var rows [][]string
	for _, helmRelease := range list.Items {
		row := []string{}
		if c := meta.GetCondition(helmRelease.Status.Conditions, meta.ReadyCondition); c != nil {
			row = []string{
				helmRelease.GetName(),
				helmRelease.Status.LastAppliedRevision,
				strings.Title(strconv.FormatBool(helmRelease.Spec.Suspend)),
				string(c.Status),
				c.Message,
			}
		} else {
			row = []string{
				helmRelease.GetName(),
				helmRelease.Status.LastAppliedRevision,
				strings.Title(strconv.FormatBool(helmRelease.Spec.Suspend)),
				string(corev1.ConditionFalse),
				"waiting to be reconciled",
			}
		}
		if allNamespaces {
			row = append([]string{helmRelease.Namespace}, row...)
		}
		rows = append(rows, row)
	}
	utils.PrintTable(os.Stdout, header, rows)
	return nil
}
