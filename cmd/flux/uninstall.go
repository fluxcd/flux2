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
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/flux2/pkg/manifestgen"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Flux and its custom resource definitions",
	Long:  "The uninstall command removes the Flux components and the toolkit.fluxcd.io resources from the cluster.",
	Example: `  # Uninstall Flux components, its custom resources and namespace
  flux uninstall --namespace=flux-system

  # Uninstall Flux but keep the namespace
  flux uninstall --namespace=infra --keep-namespace=true`,
	RunE: uninstallCmdRun,
}

type uninstallFlags struct {
	keepNamespace bool
	dryRun        bool
	silent        bool
}

var uninstallArgs uninstallFlags

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallArgs.keepNamespace, "keep-namespace", false,
		"skip namespace deletion")
	uninstallCmd.Flags().BoolVar(&uninstallArgs.dryRun, "dry-run", false,
		"only print the objects that would be deleted")
	uninstallCmd.Flags().BoolVarP(&uninstallArgs.silent, "silent", "s", false,
		"delete components without asking for confirmation")

	rootCmd.AddCommand(uninstallCmd)
}

func uninstallCmdRun(cmd *cobra.Command, args []string) error {
	if !uninstallArgs.dryRun && !uninstallArgs.silent {
		prompt := promptui.Prompt{
			Label:     "Are you sure you want to delete Flux and its custom resource definitions",
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs)
	if err != nil {
		return err
	}

	logger.Actionf("deleting components in %s namespace", *kubeconfigArgs.Namespace)
	uninstallComponents(ctx, kubeClient, *kubeconfigArgs.Namespace, uninstallArgs.dryRun)

	logger.Actionf("deleting toolkit.fluxcd.io finalizers in all namespaces")
	uninstallFinalizers(ctx, kubeClient, uninstallArgs.dryRun)

	logger.Actionf("deleting toolkit.fluxcd.io custom resource definitions")
	uninstallCustomResourceDefinitions(ctx, kubeClient, uninstallArgs.dryRun)

	if !uninstallArgs.keepNamespace {
		uninstallNamespace(ctx, kubeClient, *kubeconfigArgs.Namespace, uninstallArgs.dryRun)
	}

	logger.Successf("uninstall finished")
	return nil
}

func uninstallComponents(ctx context.Context, kubeClient client.Client, namespace string, dryRun bool) {
	opts, dryRunStr := getDeleteOptions(dryRun)
	selector := client.MatchingLabels{manifestgen.PartOfLabelKey: manifestgen.PartOfLabelValue}
	{
		var list appsv1.DeploymentList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("Deployment/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
				} else {
					logger.Successf("Deployment/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list corev1.ServiceList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("Service/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
				} else {
					logger.Successf("Service/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list networkingv1.NetworkPolicyList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("NetworkPolicy/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
				} else {
					logger.Successf("NetworkPolicy/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list corev1.ServiceAccountList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("ServiceAccount/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
				} else {
					logger.Successf("ServiceAccount/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list rbacv1.ClusterRoleList
		if err := kubeClient.List(ctx, &list, selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("ClusterRole/%s deletion failed: %s", r.Name, err.Error())
				} else {
					logger.Successf("ClusterRole/%s deleted %s", r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list rbacv1.ClusterRoleBindingList
		if err := kubeClient.List(ctx, &list, selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("ClusterRoleBinding/%s deletion failed: %s", r.Name, err.Error())
				} else {
					logger.Successf("ClusterRoleBinding/%s deleted %s", r.Name, dryRunStr)
				}
			}
		}
	}
}

func uninstallFinalizers(ctx context.Context, kubeClient client.Client, dryRun bool) {
	opts, dryRunStr := getUpdateOptions(dryRun)
	{
		var list sourcev1.GitRepositoryList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list sourcev1.HelmRepositoryList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list sourcev1.HelmChartList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list sourcev1.BucketList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list kustomizev1.KustomizationList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list helmv2.HelmReleaseList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
}

func uninstallCustomResourceDefinitions(ctx context.Context, kubeClient client.Client, dryRun bool) {
	opts, dryRunStr := getDeleteOptions(dryRun)
	selector := client.MatchingLabels{manifestgen.PartOfLabelKey: manifestgen.PartOfLabelValue}
	{
		var list apiextensionsv1.CustomResourceDefinitionList
		if err := kubeClient.List(ctx, &list, selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("CustomResourceDefinition/%s deletion failed: %s", r.Name, err.Error())
				} else {
					logger.Successf("CustomResourceDefinition/%s deleted %s", r.Name, dryRunStr)
				}
			}
		}
	}
}

func uninstallNamespace(ctx context.Context, kubeClient client.Client, namespace string, dryRun bool) {
	opts, dryRunStr := getDeleteOptions(dryRun)
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if err := kubeClient.Delete(ctx, &ns, opts); err != nil {
		logger.Failuref("Namespace/%s deletion failed: %s", namespace, err.Error())
	} else {
		logger.Successf("Namespace/%s deleted %s", namespace, dryRunStr)
	}
}

func getDeleteOptions(dryRun bool) (*client.DeleteOptions, string) {
	opts := &client.DeleteOptions{}
	var dryRunStr string
	if dryRun {
		client.DryRunAll.ApplyToDelete(opts)
		dryRunStr = "(dry run)"
	}

	return opts, dryRunStr
}

func getUpdateOptions(dryRun bool) (*client.UpdateOptions, string) {
	opts := &client.UpdateOptions{}
	var dryRunStr string
	if dryRun {
		client.DryRunAll.ApplyToUpdate(opts)
		dryRunStr = "(dry run)"
	}

	return opts, dryRunStr
}
