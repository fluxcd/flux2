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

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/toolkit/internal/flags"
	"github.com/fluxcd/toolkit/internal/utils"

	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
  gotk create hr podinfo \
    --interval=10m \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --chart-version=">4.0.0"

  # Create a HelmRelease with a chart from a GitRepository source
  gotk create hr podinfo \
    --interval=10m \
    --source=GitRepository/podinfo \
    --chart=./charts/podinfo

  # Create a HelmRelease with a chart from a Bucket source
  gotk create hr podinfo \
    --interval=10m \
    --source=Bucket/podinfo \
    --chart=./charts/podinfo

  # Create a HelmRelease with values from a local YAML file
  gotk create hr podinfo \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --values=./my-values.yaml

  # Create a HelmRelease with a custom release name
  gotk create hr podinfo \
    --release-name=podinfo-dev
    --source=HelmRepository/podinfo \
    --chart=podinfo \

  # Create a HelmRelease targeting another namespace than the resource
  gotk create hr podinfo \
    --target-namespace=default \
    --source=HelmRepository/podinfo \
    --chart=podinfo

  # Create a HelmRelease definition on disk without applying it on the cluster
  gotk create hr podinfo \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --values=./values.yaml \
    --export > podinfo-release.yaml
`,
	RunE: createHelmReleaseCmdRun,
}

var (
	hrName            string
	hrSource          flags.HelmChartSource
	hrDependsOn       []string
	hrChart           string
	hrChartVersion    string
	hrTargetNamespace string
	hrValuesFile      string
)

func init() {
	createHelmReleaseCmd.Flags().StringVar(&hrName, "release-name", "", "name used for the Helm release, defaults to a composition of '[<target-namespace>-]<HelmRelease-name>'")
	createHelmReleaseCmd.Flags().Var(&hrSource, "source", hrSource.Description())
	createHelmReleaseCmd.Flags().StringVar(&hrChart, "chart", "", "Helm chart name or path")
	createHelmReleaseCmd.Flags().StringVar(&hrChartVersion, "chart-version", "", "Helm chart version, accepts a semver range (ignored for charts from GitRepository sources)")
	createHelmReleaseCmd.Flags().StringArrayVar(&hrDependsOn, "depends-on", nil, "HelmReleases that must be ready before this release can be installed, supported formats '<name>' and '<namespace>/<name>'")
	createHelmReleaseCmd.Flags().StringVar(&hrTargetNamespace, "target-namespace", "", "namespace to install this release, defaults to the HelmRelease namespace")
	createHelmReleaseCmd.Flags().StringVar(&hrValuesFile, "values", "", "local path to the values.yaml file")
	createCmd.AddCommand(createHelmReleaseCmd)
}

func createHelmReleaseCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("HelmRelease name is required")
	}
	name := args[0]

	if hrChart == "" {
		return fmt.Errorf("chart name or path is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	if !export {
		logger.Generatef("generating HelmRelease")
	}

	helmRelease := helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    sourceLabels,
		},
		Spec: helmv2.HelmReleaseSpec{
			ReleaseName: hrName,
			DependsOn:   utils.MakeDependsOn(hrDependsOn),
			Interval: metav1.Duration{
				Duration: interval,
			},
			TargetNamespace: hrTargetNamespace,
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   hrChart,
					Version: hrChartVersion,
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind: hrSource.Kind,
						Name: hrSource.Name,
					},
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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig)
	if err != nil {
		return err
	}

	logger.Actionf("applying HelmRelease")
	namespacedName, err := upsertHelmRelease(ctx, kubeClient, &helmRelease)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for HelmRelease reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
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

		return meta.HasReadyCondition(helmRelease.Status.Conditions), nil
	}
}
