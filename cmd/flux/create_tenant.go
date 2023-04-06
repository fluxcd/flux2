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
	"bytes"
	"context"
	"fmt"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var createTenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "Create or update a tenant",
	Long: withPreviewNote(`The create tenant command generates namespaces, service accounts and role bindings to limit the
reconcilers scope to the tenant namespaces.`),
	Example: `  # Create a tenant with access to a namespace 
  flux create tenant dev-team \
    --with-namespace=frontend \
    --label=environment=dev

  # Generate tenant namespaces and role bindings in YAML format
  flux create tenant dev-team \
    --with-namespace=frontend \
    --with-namespace=backend \
	--export > dev-team.yaml`,
	RunE: createTenantCmdRun,
}

const (
	tenantLabel = "toolkit.fluxcd.io/tenant"
)

type tenantFlags struct {
	namespaces  []string
	clusterRole string
}

var tenantArgs tenantFlags

func init() {
	createTenantCmd.Flags().StringSliceVar(&tenantArgs.namespaces, "with-namespace", nil, "namespace belonging to this tenant")
	createTenantCmd.Flags().StringVar(&tenantArgs.clusterRole, "cluster-role", "cluster-admin", "cluster role of the tenant role binding")
	createCmd.AddCommand(createTenantCmd)
}

func createTenantCmdRun(cmd *cobra.Command, args []string) error {
	tenant := args[0]
	if err := validation.IsQualifiedName(tenant); len(err) > 0 {
		return fmt.Errorf("invalid tenant name '%s': %v", tenant, err)
	}

	if tenantArgs.clusterRole == "" {
		return fmt.Errorf("cluster-role is required")
	}

	if tenantArgs.namespaces == nil {
		return fmt.Errorf("with-namespace is required")
	}

	var namespaces []corev1.Namespace
	var accounts []corev1.ServiceAccount
	var roleBindings []rbacv1.RoleBinding

	for _, ns := range tenantArgs.namespaces {
		if err := validation.IsQualifiedName(ns); len(err) > 0 {
			return fmt.Errorf("invalid namespace '%s': %v", ns, err)
		}

		objLabels, err := parseLabels()
		if err != nil {
			return err
		}

		objLabels[tenantLabel] = tenant

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   ns,
				Labels: objLabels,
			},
		}
		namespaces = append(namespaces, namespace)

		account := corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tenant,
				Namespace: ns,
				Labels:    objLabels,
			},
		}

		accounts = append(accounts, account)

		roleBinding := rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-reconciler", tenant),
				Namespace: ns,
				Labels:    objLabels,
			},
			Subjects: []rbacv1.Subject{
				{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "User",
					Name:     fmt.Sprintf("gotk:%s:reconciler", ns),
				},
				{
					Kind:      "ServiceAccount",
					Name:      tenant,
					Namespace: ns,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     tenantArgs.clusterRole,
			},
		}
		roleBindings = append(roleBindings, roleBinding)
	}

	if createArgs.export {
		for i := range tenantArgs.namespaces {
			if err := exportTenant(namespaces[i], accounts[i], roleBindings[i]); err != nil {
				return err
			}
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	for i := range tenantArgs.namespaces {
		logger.Actionf("applying namespace %s", namespaces[i].Name)
		if err := upsertNamespace(ctx, kubeClient, namespaces[i]); err != nil {
			return err
		}

		logger.Actionf("applying service account %s", accounts[i].Name)
		if err := upsertServiceAccount(ctx, kubeClient, accounts[i]); err != nil {
			return err
		}

		logger.Actionf("applying role binding %s", roleBindings[i].Name)
		if err := upsertRoleBinding(ctx, kubeClient, roleBindings[i]); err != nil {
			return err
		}
	}

	logger.Successf("tenant setup completed")
	return nil
}

func upsertNamespace(ctx context.Context, kubeClient client.Client, namespace corev1.Namespace) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace.GetNamespace(),
		Name:      namespace.GetName(),
	}

	var existing corev1.Namespace
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, &namespace); err != nil {
				return err
			} else {
				return nil
			}
		}
		return err
	}

	if !equality.Semantic.DeepDerivative(namespace.Labels, existing.Labels) {
		existing.Labels = namespace.Labels
		if err := kubeClient.Update(ctx, &existing); err != nil {
			return err
		}
	}

	return nil
}

func upsertServiceAccount(ctx context.Context, kubeClient client.Client, account corev1.ServiceAccount) error {
	namespacedName := types.NamespacedName{
		Namespace: account.GetNamespace(),
		Name:      account.GetName(),
	}

	var existing corev1.ServiceAccount
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, &account); err != nil {
				return err
			} else {
				return nil
			}
		}
		return err
	}

	if !equality.Semantic.DeepDerivative(account.Labels, existing.Labels) {
		existing.Labels = account.Labels
		if err := kubeClient.Update(ctx, &existing); err != nil {
			return err
		}
	}

	return nil
}

func upsertRoleBinding(ctx context.Context, kubeClient client.Client, roleBinding rbacv1.RoleBinding) error {
	namespacedName := types.NamespacedName{
		Namespace: roleBinding.GetNamespace(),
		Name:      roleBinding.GetName(),
	}

	var existing rbacv1.RoleBinding
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, &roleBinding); err != nil {
				return err
			} else {
				return nil
			}
		}
		return err
	}

	if !equality.Semantic.DeepDerivative(roleBinding.Subjects, existing.Subjects) ||
		!equality.Semantic.DeepDerivative(roleBinding.RoleRef, existing.RoleRef) ||
		!equality.Semantic.DeepDerivative(roleBinding.Labels, existing.Labels) {
		if err := kubeClient.Delete(ctx, &existing); err != nil {
			return err
		}
		if err := kubeClient.Create(ctx, &roleBinding); err != nil {
			return err
		}
	}

	return nil
}

func exportTenant(namespace corev1.Namespace, account corev1.ServiceAccount, roleBinding rbacv1.RoleBinding) error {
	namespace.TypeMeta = metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Namespace",
	}
	data, err := yaml.Marshal(namespace)
	if err != nil {
		return err
	}

	fmt.Println("---")
	data = bytes.Replace(data, []byte("spec: {}\n"), []byte(""), 1)
	fmt.Println(resourceToString(data))

	account.TypeMeta = metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "ServiceAccount",
	}
	data, err = yaml.Marshal(account)
	if err != nil {
		return err
	}

	fmt.Println("---")
	data = bytes.Replace(data, []byte("spec: {}\n"), []byte(""), 1)
	fmt.Println(resourceToString(data))

	roleBinding.TypeMeta = metav1.TypeMeta{
		APIVersion: "rbac.authorization.k8s.io/v1",
		Kind:       "RoleBinding",
	}
	data, err = yaml.Marshal(roleBinding)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(resourceToString(data))

	return nil
}
