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
	"time"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var getImageRepositoryCmd = &cobra.Command{
	Use:   "image-repository",
	Short: "Get ImageRepository statuses",
	Long:  "The get auto image-repository command prints the status of ImageRepository objects.",
	Example: `  # List all image repositories and their status
  flux get auto image-repository

 # List image repositories from all namespaces
  flux get auto image-repository --all-namespaces
`,
	RunE: getImageRepositoryRun,
}

func init() {
	getAutoCmd.AddCommand(getImageRepositoryCmd)
}

func getImageRepositoryRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return err
	}

	var listOpts []client.ListOption
	if !allNamespaces {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}
	var list imagev1.ImageRepositoryList
	err = kubeClient.List(ctx, &list, listOpts...)
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no imagerepository objects found in %s namespace", namespace)
		return nil
	}

	header := []string{"Name", "Ready", "Message", "Last scan", "Suspended"}
	if allNamespaces {
		header = append([]string{"Namespace"}, header...)
	}
	var rows [][]string
	for _, repo := range list.Items {
		var row []string
		var lastScan string
		if repo.Status.LastScanResult != nil {
			lastScan = repo.Status.LastScanResult.ScanTime.Time.Format(time.RFC3339)
		}
		if c := apimeta.FindStatusCondition(repo.Status.Conditions, meta.ReadyCondition); c != nil {
			row = []string{
				repo.GetName(),
				string(c.Status),
				c.Message,
				lastScan,
				strings.Title(strconv.FormatBool(repo.Spec.Suspend)),
			}
		} else {
			row = []string{
				repo.GetName(),
				string(metav1.ConditionFalse),
				"waiting to be reconciled",
				lastScan,
				strings.Title(strconv.FormatBool(repo.Spec.Suspend)),
			}
		}
		if allNamespaces {
			row = append([]string{repo.Namespace}, row...)
		}
		rows = append(rows, row)
	}
	utils.PrintTable(os.Stdout, header, rows)
	return nil
}
