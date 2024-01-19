/*
Copyright 2023 The Flux authors

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
	"fmt"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/cli-utils/pkg/kstatus/status"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
	autov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
	notificationv1b3 "github.com/fluxcd/notification-controller/api/v1beta3"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/printers"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Stats of Flux reconciles",
	Long: withPreviewNote(`The stats command prints a report of Flux custom resources present on a cluster,
including their reconcile status and the amount of cumulative storage used for each source type`),
	Example: `  # Print the stats report for a namespace
  flux stats --namespace default

  #  Print the stats report for the whole cluster
  flux stats -A`,
	RunE: runStatsCmd,
}

type StatsFlags struct {
	allNamespaces bool
}

var statsArgs StatsFlags

func init() {
	statsCmd.PersistentFlags().BoolVarP(&statsArgs.allNamespaces, "all-namespaces", "A", false,
		"list the statistics for objects across all namespaces")
	rootCmd.AddCommand(statsCmd)
}

func runStatsCmd(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	types := []metav1.GroupVersionKind{
		{
			Kind:    sourcev1.GitRepositoryKind,
			Version: sourcev1.GroupVersion.Version,
			Group:   sourcev1.GroupVersion.Group,
		},
		{
			Kind:    sourcev1b2.OCIRepositoryKind,
			Version: sourcev1b2.GroupVersion.Version,
			Group:   sourcev1b2.GroupVersion.Group,
		},
		{
			Kind:    sourcev1b2.HelmRepositoryKind,
			Version: sourcev1b2.GroupVersion.Version,
			Group:   sourcev1b2.GroupVersion.Group,
		},
		{
			Kind:    sourcev1b2.HelmChartKind,
			Version: sourcev1b2.GroupVersion.Version,
			Group:   sourcev1b2.GroupVersion.Group,
		},
		{
			Kind:    sourcev1b2.BucketKind,
			Version: sourcev1b2.GroupVersion.Version,
			Group:   sourcev1b2.GroupVersion.Group,
		},
		{
			Kind:    kustomizev1.KustomizationKind,
			Version: kustomizev1.GroupVersion.Version,
			Group:   kustomizev1.GroupVersion.Group,
		},
		{
			Kind:    helmv2.HelmReleaseKind,
			Version: helmv2.GroupVersion.Version,
			Group:   helmv2.GroupVersion.Group,
		},
		{
			Kind:    notificationv1b3.AlertKind,
			Version: notificationv1b3.GroupVersion.Version,
			Group:   notificationv1b3.GroupVersion.Group,
		},
		{
			Kind:    notificationv1b3.ProviderKind,
			Version: notificationv1b3.GroupVersion.Version,
			Group:   notificationv1b3.GroupVersion.Group,
		},
		{
			Kind:    notificationv1.ReceiverKind,
			Version: notificationv1.GroupVersion.Version,
			Group:   notificationv1.GroupVersion.Group,
		},
		{
			Kind:    autov1.ImageUpdateAutomationKind,
			Version: autov1.GroupVersion.Version,
			Group:   autov1.GroupVersion.Group,
		},
		{
			Kind:    imagev1.ImagePolicyKind,
			Version: imagev1.GroupVersion.Version,
			Group:   imagev1.GroupVersion.Group,
		},
		{
			Kind:    imagev1.ImageRepositoryKind,
			Version: imagev1.GroupVersion.Version,
			Group:   imagev1.GroupVersion.Group,
		},
	}

	header := []string{"Reconcilers", "Running", "Failing", "Suspended", "Storage"}
	var rows [][]string

	for _, t := range types {
		var total int
		var suspended int
		var failing int
		var totalSize int64

		list := unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"apiVersion": t.Group + "/" + t.Version,
				"kind":       t.Kind,
			},
		}

		scope := client.InNamespace("")
		if !statsArgs.allNamespaces {
			scope = client.InNamespace(*kubeconfigArgs.Namespace)
		}

		if err := kubeClient.List(ctx, &list, scope); err == nil {
			total = len(list.Items)

			for _, item := range list.Items {
				if s, _, _ := unstructured.NestedBool(item.Object, "spec", "suspend"); s {
					suspended++
				}

				if obj, err := status.GetObjectWithConditions(item.Object); err == nil {
					for _, cond := range obj.Status.Conditions {
						if cond.Type == "Ready" && cond.Status == corev1.ConditionFalse {
							failing++
						}
					}
				}

				if size, found, _ := unstructured.NestedInt64(item.Object, "status", "artifact", "size"); found {
					totalSize += size
				}
			}
		}

		rows = append(rows, []string{
			t.Kind,
			formatInt(total - suspended),
			formatInt(failing),
			formatInt(suspended),
			formatSize(totalSize),
		})
	}

	err = printers.TablePrinter(header).Print(cmd.OutOrStdout(), rows)
	if err != nil {
		return err
	}

	return nil
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}

func formatSize(b int64) string {
	if b == 0 {
		return "-"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}
