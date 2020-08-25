/*
Copyright 2020 The Flux CD contributors.

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
	"io/ioutil"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	helmv2 "github.com/fluxcd/helm-controller/api/v2alpha1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
)

var createHelmReleaseCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Create or update a HelmRelease resource",
	Long:    "The helmrelease create command generates a HelmRelease resource for a given HelmRepository source.",
	Example: `  # Create a HelmRelease from a source
  gotk create hr podinfo \
    --interval=10m \
    --release-name=podinfo \
    --target-namespace=default \
    --source=podinfo \
    --chart-name=podinfo \
    --chart-version=">4.0.0"

  # Create a HelmRelease with values for a local YAML file
  gotk create hr podinfo \
    --target-namespace=default \
    --source=podinfo \
    --chart-name=podinfo \
    --chart-version=4.0.5 \
    --values=./my-values.yaml

  # Create a HelmRelease definition on disk without applying it on the cluster
  gotk create hr podinfo \
    --target-namespace=default \
    --source=podinfo \
    --chart-name=podinfo \
    --chart-version=4.0.5 \
    --values=./values.yaml \
    --export > podinfo-release.yaml
`,
	RunE: createHelmReleaseCmdRun,
}

var (
	hrName            string
	hrSource          string
	hrDependsOn       []string
	hrChartName       string
	hrChartVersion    string
	hrTargetNamespace string
	hrValuesFile      string
)

func init() {
	createHelmReleaseCmd.Flags().StringVar(&hrName, "release-name", "", "name used for the Helm release, defaults to a composition of '<target-namespace>-<hr-name>'")
	createHelmReleaseCmd.Flags().StringVar(&hrSource, "source", "", "HelmRepository name")
	createHelmReleaseCmd.Flags().StringVar(&hrChartName, "chart-name", "", "Helm chart name")
	createHelmReleaseCmd.Flags().StringVar(&hrChartVersion, "chart-version", "", "Helm chart version, accepts semver range")
	createHelmReleaseCmd.Flags().StringArrayVar(&hrDependsOn, "depends-on", nil, "HelmReleases that must be ready before this release can be installed")
	createHelmReleaseCmd.Flags().StringVar(&hrTargetNamespace, "target-namespace", "", "namespace to install this release, defaults to the HelmRelease namespace")
	createHelmReleaseCmd.Flags().StringVar(&hrValuesFile, "values", "", "local path to the values.yaml file")
	createCmd.AddCommand(createHelmReleaseCmd)
}

func createHelmReleaseCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("release name is required")
	}
	name := args[0]

	if hrSource == "" {
		return fmt.Errorf("source is required")
	}
	if hrChartName == "" {
		return fmt.Errorf("chart name is required")
	}
	if hrChartVersion == "" {
		return fmt.Errorf("chart version is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	if !export {
		logger.Generatef("generating release")
	}

	helmRelease := helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: helmv2.HelmReleaseSpec{
			ReleaseName: hrName,
			DependsOn:   hrDependsOn,
			Interval: metav1.Duration{
				Duration: interval,
			},
			TargetNamespace: hrTargetNamespace,
			Chart: helmv2.HelmChartTemplate{
				Name:    hrChartName,
				Version: hrChartVersion,
				SourceRef: helmv2.CrossNamespaceObjectReference{
					Kind: sourcev1.HelmRepositoryKind,
					Name: hrSource,
				},
			},
			Suspend: false,
		},
	}

	if hrValuesFile != "" {
		data, err := ioutil.ReadFile(hrValuesFile)
		if err != nil {
			return fmt.Errorf("reading values from %s failed: %w", hrValuesFile, err)
		}

		json, err := yaml.YAMLToJSON(data)
		if err != nil {
			return fmt.Errorf("converting values to JSON from %s failed: %w", hrValuesFile, err)
		}

		helmRelease.Spec.Values = &apiextensionsv1.JSON{Raw: json}
	}

	if export {
		return exportHelmRelease(helmRelease)
	}

	logger.Actionf("applying release")
	if err := upsertHelmRelease(ctx, kubeClient, helmRelease); err != nil {
		return err
	}

	logger.Waitingf("waiting for reconciliation")
	chartName := fmt.Sprintf("%s-%s", namespace, name)
	if err := wait.PollImmediate(pollInterval, timeout,
		isHelmChartReady(ctx, kubeClient, chartName, namespace)); err != nil {
		return err
	}
	if err := wait.PollImmediate(pollInterval, timeout,
		isHelmReleaseReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("release %s is ready", name)

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err = kubeClient.Get(ctx, namespacedName, &helmRelease)
	if err != nil {
		return fmt.Errorf("release failed: %w", err)
	}

	if helmRelease.Status.LastAppliedRevision != "" {
		logger.Successf("applied revision %s", helmRelease.Status.LastAppliedRevision)
	} else {
		return fmt.Errorf("reconciliation failed")
	}

	return nil
}

func upsertHelmRelease(ctx context.Context, kubeClient client.Client, helmRelease helmv2.HelmRelease) error {
	namespacedName := types.NamespacedName{
		Namespace: helmRelease.GetNamespace(),
		Name:      helmRelease.GetName(),
	}

	var existing helmv2.HelmRelease
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, &helmRelease); err != nil {
				return err
			} else {
				logger.Successf("release created")
				return nil
			}
		}
		return err
	}

	existing.Spec = helmRelease.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return err
	}

	logger.Successf("release updated")
	return nil
}

func isHelmChartReady(ctx context.Context, kubeClient client.Client, name, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		var helmChart sourcev1.HelmChart
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := kubeClient.Get(ctx, namespacedName, &helmChart)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		for _, condition := range helmChart.Status.Conditions {
			if condition.Type == helmv2.ReadyCondition {
				if condition.Status == corev1.ConditionTrue {
					return true, nil
				} else if condition.Status == corev1.ConditionFalse {
					return false, fmt.Errorf(condition.Message)
				}
			}
		}
		return false, nil
	}
}
