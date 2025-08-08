/*
Copyright 2025 The Flux authors

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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Args:  cobra.NoArgs,
	Short: "Migrate the Flux custom resources to their latest API version",
	Long: `The migrate command must be run before a Flux minor version upgrade.
The command migrates the Flux custom resources stored in Kubernetes etcd to their latest API version,
ensuring the Flux components can continue to function correctly after the upgrade.
`,
	RunE: runMigrateCmd,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

func runMigrateCmd(cmd *cobra.Command, args []string) error {
	logger.Actionf("starting migration of custom resources")
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	cfg, err := utils.KubeConfig(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return fmt.Errorf("Kubernetes client initialization failed: %s", err.Error())
	}

	kubeClient, err := client.New(cfg, client.Options{Scheme: utils.NewScheme()})
	if err != nil {
		return err
	}

	migrator := NewMigrator(kubeClient, client.MatchingLabels{
		"app.kubernetes.io/part-of": "flux",
	})

	if err := migrator.Run(ctx); err != nil {
		return err
	}

	logger.Successf("custom resources migrated successfully")
	return nil
}

type Migrator struct {
	labelSelector client.MatchingLabels
	kubeClient    client.Client
}

// NewMigrator creates a new Migrator instance with the specified label selector.
func NewMigrator(kubeClient client.Client, labelSelector client.MatchingLabels) *Migrator {
	return &Migrator{
		labelSelector: labelSelector,
		kubeClient:    kubeClient,
	}
}

func (m *Migrator) Run(ctx context.Context) error {
	crdList := &apiextensionsv1.CustomResourceDefinitionList{}

	if err := m.kubeClient.List(ctx, crdList, m.labelSelector); err != nil {
		return fmt.Errorf("failed to list CRDs: %w", err)
	}

	for _, crd := range crdList.Items {
		if err := m.migrateCRD(ctx, crd.Name); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) migrateCRD(ctx context.Context, name string) error {
	crd := &apiextensionsv1.CustomResourceDefinition{}

	if err := m.kubeClient.Get(ctx, client.ObjectKey{Name: name}, crd); err != nil {
		return fmt.Errorf("failed to get CRD %s: %w", name, err)
	}

	// get the latest storage version for the CRD
	storageVersion := m.getStorageVersion(crd)
	if storageVersion == "" {
		return fmt.Errorf("no storage version found for CRD %s", name)
	}

	// migrate all the resources for the CRD
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return m.migrateCR(ctx, crd, storageVersion)
	})
	if err != nil {
		return fmt.Errorf("failed to migrate resources for CRD %s: %w", name, err)
	}

	// set the CRD status to contain only the latest storage version
	if len(crd.Status.StoredVersions) > 1 || crd.Status.StoredVersions[0] != storageVersion {
		crd.Status.StoredVersions = []string{storageVersion}
		if err := m.kubeClient.Status().Update(ctx, crd); err != nil {
			return fmt.Errorf("failed to update CRD %s status: %w", crd.Name, err)
		}
		logger.Successf("%s migrated to storage version %s", crd.Name, storageVersion)
	}
	return nil
}

// migrateCR migrates all CRs for the given CRD to the specified version by patching them with an empty patch.
func (m *Migrator) migrateCR(ctx context.Context, crd *apiextensionsv1.CustomResourceDefinition, version string) error {
	list := &unstructured.UnstructuredList{}

	apiVersion := crd.Spec.Group + "/" + version
	listKind := crd.Spec.Names.ListKind

	list.SetAPIVersion(apiVersion)
	list.SetKind(listKind)

	err := m.kubeClient.List(ctx, list, client.InNamespace(""))
	if err != nil {
		return fmt.Errorf("failed to list resources for CRD %s: %w", crd.Name, err)
	}

	if len(list.Items) == 0 {
		return nil
	}

	for _, item := range list.Items {
		// patch the resource with an empty patch to update the version
		if err := m.kubeClient.Patch(
			ctx,
			&item,
			client.RawPatch(client.Merge.Type(), []byte("{}")),
		); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf(" %s/%s/%s failed to migrate: %w",
				item.GetKind(), item.GetNamespace(), item.GetName(), err)
		}

		logger.Successf("%s/%s/%s migrated to version %s",
			item.GetKind(), item.GetNamespace(), item.GetName(), version)
	}

	return nil
}

// getStorageVersion retrieves the storage version of a CustomResourceDefinition.
func (m *Migrator) getStorageVersion(crd *apiextensionsv1.CustomResourceDefinition) string {
	var version string
	for _, v := range crd.Spec.Versions {
		if v.Storage {
			version = v.Name
			break
		}
	}

	return version
}
