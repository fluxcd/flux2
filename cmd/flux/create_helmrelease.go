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
	"encoding/json"
	"fmt"
	"os"

	"github.com/fluxcd/flux2/internal/flags"
	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/transform"

	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
)

var createHelmReleaseCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Create or update a HelmRelease resource",
	Long:    "The helmrelease create command generates a HelmRelease resource for a given HelmRepository source.",
	Example: `  # Create a HelmRelease with a chart from a HelmRepository source
  flux create hr podinfo \
    --interval=10m \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --chart-version=">4.0.0"

  # Create a HelmRelease with a chart from a GitRepository source
  flux create hr podinfo \
    --interval=10m \
    --source=GitRepository/podinfo \
    --chart=./charts/podinfo

  # Create a HelmRelease with a chart from a Bucket source
  flux create hr podinfo \
    --interval=10m \
    --source=Bucket/podinfo \
    --chart=./charts/podinfo

  # Create a HelmRelease with values from local YAML files
  flux create hr podinfo \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --values=./my-values1.yaml \
    --values=./my-values2.yaml

  # Create a HelmRelease with values from a Kubernetes secret
  kubectl -n app create secret generic my-secret-values \
	--from-file=values.yaml=/path/to/my-secret-values.yaml
  flux -n app create hr podinfo \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --values-from=Secret/my-secret-values

  # Create a HelmRelease with a custom release name
  flux create hr podinfo \
    --release-name=podinfo-dev
    --source=HelmRepository/podinfo \
    --chart=podinfo \

  # Create a HelmRelease targeting another namespace than the resource
  flux create hr podinfo \
    --target-namespace=test \
    --create-target-namespace=true \
    --source=HelmRepository/podinfo \
    --chart=podinfo

  # Create a HelmRelease using a source from a different namespace
  flux create hr podinfo \
    --namespace=default \
    --source=HelmRepository/podinfo.flux-system \
    --chart=podinfo

  # Create a HelmRelease definition on disk without applying it on the cluster
  flux create hr podinfo \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --values=./values.yaml \
    --export > podinfo-release.yaml`,
	RunE: createHelmReleaseCmdRun,
}

type helmReleaseFlags struct {
	name            string
	source          flags.HelmChartSource
	dependsOn       []string
	chart           string
	chartVersion    string
	targetNamespace string
	createNamespace bool
	valuesFiles     []string
	valuesFrom      flags.HelmReleaseValuesFrom
	saName          string
	crds            flags.CRDsPolicy
}

var helmReleaseArgs helmReleaseFlags

func init() {
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.name, "release-name", "", "name used for the Helm release, defaults to a composition of '[<target-namespace>-]<HelmRelease-name>'")
	createHelmReleaseCmd.Flags().Var(&helmReleaseArgs.source, "source", helmReleaseArgs.source.Description())
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.chart, "chart", "", "Helm chart name or path")
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.chartVersion, "chart-version", "", "Helm chart version, accepts a semver range (ignored for charts from GitRepository sources)")
	createHelmReleaseCmd.Flags().StringSliceVar(&helmReleaseArgs.dependsOn, "depends-on", nil, "HelmReleases that must be ready before this release can be installed, supported formats '<name>' and '<namespace>/<name>'")
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.targetNamespace, "target-namespace", "", "namespace to install this release, defaults to the HelmRelease namespace")
	createHelmReleaseCmd.Flags().BoolVar(&helmReleaseArgs.createNamespace, "create-target-namespace", false, "create the target namespace if it does not exist")
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.saName, "service-account", "", "the name of the service account to impersonate when reconciling this HelmRelease")
	createHelmReleaseCmd.Flags().StringSliceVar(&helmReleaseArgs.valuesFiles, "values", nil, "local path to values.yaml files, also accepts comma-separated values")
	createHelmReleaseCmd.Flags().Var(&helmReleaseArgs.valuesFrom, "values-from", helmReleaseArgs.valuesFrom.Description())
	createHelmReleaseCmd.Flags().Var(&helmReleaseArgs.crds, "crds", helmReleaseArgs.crds.Description())
	createCmd.AddCommand(createHelmReleaseCmd)
}

func createHelmReleaseCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if helmReleaseArgs.chart == "" {
		return fmt.Errorf("chart name or path is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	if !createArgs.export {
		logger.Generatef("generating HelmRelease")
	}

	helmRelease := helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *kubeconfigArgs.Namespace,
			Labels:    sourceLabels,
		},
		Spec: helmv2.HelmReleaseSpec{
			ReleaseName: helmReleaseArgs.name,
			DependsOn:   utils.MakeDependsOn(helmReleaseArgs.dependsOn),
			Interval: metav1.Duration{
				Duration: createArgs.interval,
			},
			TargetNamespace: helmReleaseArgs.targetNamespace,

			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   helmReleaseArgs.chart,
					Version: helmReleaseArgs.chartVersion,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind:      helmReleaseArgs.source.Kind,
						Name:      helmReleaseArgs.source.Name,
						Namespace: helmReleaseArgs.source.Namespace,
					},
				},
			},
			Suspend: false,
		},
	}

	if helmReleaseArgs.createNamespace {
		if helmRelease.Spec.Install == nil {
			helmRelease.Spec.Install = &helmv2.Install{}
		}

		helmRelease.Spec.Install.CreateNamespace = helmReleaseArgs.createNamespace
	}

	if helmReleaseArgs.saName != "" {
		helmRelease.Spec.ServiceAccountName = helmReleaseArgs.saName
	}

	if helmReleaseArgs.crds != "" {
		if helmRelease.Spec.Install == nil {
			helmRelease.Spec.Install = &helmv2.Install{}
		}

		helmRelease.Spec.Install.CRDs = helmv2.Create
		helmRelease.Spec.Upgrade = &helmv2.Upgrade{CRDs: helmv2.CRDsPolicy(helmReleaseArgs.crds.String())}
	}

	if len(helmReleaseArgs.valuesFiles) > 0 {
		valuesMap := make(map[string]interface{})
		for _, v := range helmReleaseArgs.valuesFiles {
			data, err := os.ReadFile(v)
			if err != nil {
				return fmt.Errorf("reading values from %s failed: %w", v, err)
			}

			jsonBytes, err := yaml.YAMLToJSON(data)
			if err != nil {
				return fmt.Errorf("converting values to JSON from %s failed: %w", v, err)
			}

			jsonMap := make(map[string]interface{})
			if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
				return fmt.Errorf("unmarshaling values from %s failed: %w", v, err)
			}

			valuesMap = transform.MergeMaps(valuesMap, jsonMap)
		}

		jsonRaw, err := json.Marshal(valuesMap)
		if err != nil {
			return fmt.Errorf("marshaling values failed: %w", err)
		}

		helmRelease.Spec.Values = &apiextensionsv1.JSON{Raw: jsonRaw}
	}

	if helmReleaseArgs.valuesFrom.String() != "" {
		helmRelease.Spec.ValuesFrom = []helmv2.ValuesReference{{
			Kind: helmReleaseArgs.valuesFrom.Kind,
			Name: helmReleaseArgs.valuesFrom.Name,
		}}
	}

	if createArgs.export {
		return printExport(exportHelmRelease(&helmRelease))
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs)
	if err != nil {
		return err
	}

	logger.Actionf("applying HelmRelease")
	namespacedName, err := upsertHelmRelease(ctx, kubeClient, &helmRelease)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for HelmRelease reconciliation")
	if err := wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
		isHelmReleaseReady(ctx, kubeClient, namespacedName, &helmRelease)); err != nil {
		return err
	}
	logger.Successf("HelmRelease %s is ready", name)

	logger.Successf("applied revision %s", helmRelease.Status.LastAppliedRevision)
	return nil
}

func upsertHelmRelease(ctx context.Context, kubeClient client.Client,
	helmRelease *helmv2.HelmRelease) (types.NamespacedName, error) {
	namespacedName := types.NamespacedName{
		Namespace: helmRelease.GetNamespace(),
		Name:      helmRelease.GetName(),
	}

	var existing helmv2.HelmRelease
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, helmRelease); err != nil {
				return namespacedName, err
			} else {
				logger.Successf("HelmRelease created")
				return namespacedName, nil
			}
		}
		return namespacedName, err
	}

	existing.Labels = helmRelease.Labels
	existing.Spec = helmRelease.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return namespacedName, err
	}
	helmRelease = &existing
	logger.Successf("HelmRelease updated")
	return namespacedName, nil
}

func isHelmReleaseReady(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, helmRelease *helmv2.HelmRelease) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, helmRelease)
		if err != nil {
			return false, err
		}

		// Confirm the state we are observing is for the current generation
		if helmRelease.Generation != helmRelease.Status.ObservedGeneration {
			return false, nil
		}

		return apimeta.IsStatusConditionTrue(helmRelease.Status.Conditions, meta.ReadyCondition), nil
	}
}
