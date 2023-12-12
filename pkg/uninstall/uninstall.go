/*
Copyright 2022 The Flux authors

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

package uninstall

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
	autov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
	notificationv1b3 "github.com/fluxcd/notification-controller/api/v1beta3"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/pkg/log"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen"
)

// Components removes all Kubernetes components that are part of Flux excluding the CRDs and namespace.
func Components(ctx context.Context, logger log.Logger, kubeClient client.Client, namespace string, dryRun bool) error {
	var aggregateErr []error
	opts, dryRunStr := getDeleteOptions(dryRun)
	selector := client.MatchingLabels{manifestgen.PartOfLabelKey: manifestgen.PartOfLabelValue}
	{
		var list appsv1.DeploymentList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("Deployment/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("Deployment/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list corev1.ServiceList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("Service/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("Service/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list networkingv1.NetworkPolicyList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("NetworkPolicy/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("NetworkPolicy/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list corev1.ServiceAccountList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("ServiceAccount/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("ServiceAccount/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list rbacv1.ClusterRoleList
		if err := kubeClient.List(ctx, &list, selector); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("ClusterRole/%s deletion failed: %s", r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("ClusterRole/%s deleted %s", r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list rbacv1.ClusterRoleBindingList
		if err := kubeClient.List(ctx, &list, selector); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("ClusterRoleBinding/%s deletion failed: %s", r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("ClusterRoleBinding/%s deleted %s", r.Name, dryRunStr)
				}
			}
		}
	}

	return errors.Reduce(errors.Flatten(errors.NewAggregate(aggregateErr)))
}

// Finalizers removes all finalizes on Kubernetes components that have been added by a Flux controller.
func Finalizers(ctx context.Context, logger log.Logger, kubeClient client.Client, dryRun bool) error {
	var aggregateErr []error
	opts, dryRunStr := getUpdateOptions(dryRun)
	{
		var list sourcev1.GitRepositoryList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list sourcev1b2.OCIRepositoryList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list sourcev1b2.HelmRepositoryList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list sourcev1b2.HelmChartList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list sourcev1b2.BucketList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list kustomizev1.KustomizationList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list helmv2.HelmReleaseList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list notificationv1b3.AlertList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list notificationv1b3.ProviderList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list notificationv1.ReceiverList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list imagev1.ImagePolicyList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list imagev1.ImageRepositoryList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list autov1.ImageUpdateAutomationList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	return errors.Reduce(errors.Flatten(errors.NewAggregate(aggregateErr)))
}

// CustomResourceDefinitions removes all Kubernetes CRDs that are a part of Flux.
func CustomResourceDefinitions(ctx context.Context, logger log.Logger, kubeClient client.Client, dryRun bool) error {
	var aggregateErr []error
	opts, dryRunStr := getDeleteOptions(dryRun)
	selector := client.MatchingLabels{manifestgen.PartOfLabelKey: manifestgen.PartOfLabelValue}
	{
		var list apiextensionsv1.CustomResourceDefinitionList
		if err := kubeClient.List(ctx, &list, selector); err == nil {
			for i := range list.Items {
				r := list.Items[i]
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					logger.Failuref("CustomResourceDefinition/%s deletion failed: %s", r.Name, err.Error())
					aggregateErr = append(aggregateErr, err)
				} else {
					logger.Successf("CustomResourceDefinition/%s deleted %s", r.Name, dryRunStr)
				}
			}
		}
	}
	return errors.Reduce(errors.Flatten(errors.NewAggregate(aggregateErr)))
}

// Namespace removes the namespace Flux is installed in.
func Namespace(ctx context.Context, logger log.Logger, kubeClient client.Client, namespace string, dryRun bool) error {
	var aggregateErr []error
	opts, dryRunStr := getDeleteOptions(dryRun)
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if err := kubeClient.Delete(ctx, &ns, opts); err != nil {
		logger.Failuref("Namespace/%s deletion failed: %s", namespace, err.Error())
		aggregateErr = append(aggregateErr, err)
	} else {
		logger.Successf("Namespace/%s deleted %s", namespace, dryRunStr)
	}
	return errors.Reduce(errors.Flatten(errors.NewAggregate(aggregateErr)))
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
