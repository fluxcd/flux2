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
	"strings"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/transform"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/internal/utils"

	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
)

var createHelmReleaseCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Create or update a HelmRelease resource",
	Long:    withPreviewNote(`The helmrelease create command generates a HelmRelease resource for a given HelmRepository source.`),
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
    --release-name=podinfo-dev \
    --source=HelmRepository/podinfo \
    --chart=podinfo

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
	name                string
	source              flags.HelmChartSource
	dependsOn           []string
	chart               string
	chartVersion        string
	targetNamespace     string
	createNamespace     bool
	valuesFiles         []string
	valuesFrom          []string
	saName              string
	crds                flags.CRDsPolicy
	reconcileStrategy   string
	chartInterval       time.Duration
	kubeConfigSecretRef string
}

var helmReleaseArgs helmReleaseFlags

var supportedHelmReleaseValuesFromKinds = []string{"Secret", "ConfigMap"}

func init() {
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.name, "release-name", "", "name used for the Helm release, defaults to a composition of '[<target-namespace>-]<HelmRelease-name>'")
	createHelmReleaseCmd.Flags().Var(&helmReleaseArgs.source, "source", helmReleaseArgs.source.Description())
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.chart, "chart", "", "Helm chart name or path")
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.chartVersion, "chart-version", "", "Helm chart version, accepts a semver range (ignored for charts from GitRepository sources)")
	createHelmReleaseCmd.Flags().StringSliceVar(&helmReleaseArgs.dependsOn, "depends-on", nil, "HelmReleases that must be ready before this release can be installed, supported formats '<name>' and '<namespace>/<name>'")
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.targetNamespace, "target-namespace", "", "namespace to install this release, defaults to the HelmRelease namespace")
	createHelmReleaseCmd.Flags().BoolVar(&helmReleaseArgs.createNamespace, "create-target-namespace", false, "create the target namespace if it does not exist")
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.saName, "service-account", "", "the name of the service account to impersonate when reconciling this HelmRelease")
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.reconcileStrategy, "reconcile-strategy", "ChartVersion", "the reconcile strategy for helm chart created by the helm release(accepted values: Revision and ChartRevision)")
	createHelmReleaseCmd.Flags().DurationVarP(&helmReleaseArgs.chartInterval, "chart-interval", "", 0, "the interval of which to check for new chart versions")
	createHelmReleaseCmd.Flags().StringSliceVar(&helmReleaseArgs.valuesFiles, "values", nil, "local path to values.yaml files, also accepts comma-separated values")
	createHelmReleaseCmd.Flags().StringSliceVar(&helmReleaseArgs.valuesFrom, "values-from", nil, "a Kubernetes object reference that contains the values.yaml data key in the format '<kind>/<name>', where kind must be one of: (Secret,ConfigMap)")
	createHelmReleaseCmd.Flags().Var(&helmReleaseArgs.crds, "crds", helmReleaseArgs.crds.Description())
	createHelmReleaseCmd.Flags().StringVar(&helmReleaseArgs.kubeConfigSecretRef, "kubeconfig-secret-ref", "", "the name of the Kubernetes Secret that contains a key with the kubeconfig file for connecting to a remote cluster")
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

	if !validateStrategy(helmReleaseArgs.reconcileStrategy) {
		return fmt.Errorf("'%s' is an invalid reconcile strategy(valid: Revision, ChartVersion)",
			helmReleaseArgs.reconcileStrategy)
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
					ReconcileStrategy: helmReleaseArgs.reconcileStrategy,
				},
			},
			Suspend: false,
		},
	}

	if helmReleaseArgs.kubeConfigSecretRef != "" {
		helmRelease.Spec.KubeConfig = &meta.KubeConfigReference{
			SecretRef: meta.SecretKeyReference{
				Name: helmReleaseArgs.kubeConfigSecretRef,
			},
		}
	}

	if helmReleaseArgs.chartInterval != 0 {
		helmRelease.Spec.Chart.Spec.Interval = &metav1.Duration{
			Duration: helmReleaseArgs.chartInterval,
		}
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

	if len(helmReleaseArgs.valuesFrom) != 0 {
		values := []helmv2.ValuesReference{}
		for _, value := range helmReleaseArgs.valuesFrom {
			sourceKind, sourceName := utils.ParseObjectKindName(value)
			if sourceKind == "" {
				return fmt.Errorf("invalid Kubernetes object reference '%s', must be in format <kind>/<name>", value)
			}
			cleanSourceKind, ok := utils.ContainsEqualFoldItemString(supportedHelmReleaseValuesFromKinds, sourceKind)
			if !ok {
				return fmt.Errorf("reference kind '%s' is not supported, must be one of: %s",
					sourceKind, strings.Join(supportedHelmReleaseValuesFromKinds, ", "))
			}

			values = append(values, helmv2.ValuesReference{
				Name: sourceName,
				Kind: cleanSourceKind,
			})
		}
		helmRelease.Spec.ValuesFrom = values
	}

	if createArgs.export {
		return printExport(exportHelmRelease(&helmRelease))
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	logger.Actionf("applying HelmRelease")
	namespacedName, err := upsertHelmRelease(ctx, kubeClient, &helmRelease)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for HelmRelease reconciliation")
	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true,
		isObjectReadyConditionFunc(kubeClient, namespacedName, &helmRelease)); err != nil {
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

func validateStrategy(input string) bool {
	allowedStrategy := []string{"Revision", "ChartVersion"}

	for _, strategy := range allowedStrategy {
		if strategy == input {
			return true
		}
	}

	return false
}
