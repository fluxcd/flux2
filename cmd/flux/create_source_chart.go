/*
Copyright 2024 The Flux authors

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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/internal/utils"
)

var createSourceChartCmd = &cobra.Command{
	Use:   "chart [name]",
	Short: "Create or update a HelmChart source",
	Long:  `The create source chart command generates a HelmChart resource and waits for the chart to be available.`,
	Example: `  # Create a source for a chart residing in a HelmRepository
  flux create source chart podinfo \
    --source=HelmRepository/podinfo \
    --chart=podinfo \
    --chart-version=6.x

  # Create a source for a chart residing in a Git repository
  flux create source chart podinfo \
    --source=GitRepository/podinfo \
    --chart=./charts/podinfo

  # Create a source for a chart residing in a S3 Bucket
  flux create source chart podinfo \
    --source=Bucket/podinfo \
    --chart=./charts/podinfo

  # Create a source for a chart from OCI and verify its signature
  flux create source chart podinfo \
    --source HelmRepository/podinfo \
    --chart podinfo \
    --chart-version=6.6.2 \
    --verify-provider=cosign \
    --verify-issuer=https://token.actions.githubusercontent.com \
    --verify-subject=https://github.com/stefanprodan/podinfo/.github/workflows/release.yml@refs/tags/6.6.2`,
	RunE: createSourceChartCmdRun,
}

type sourceChartFlags struct {
	chart             string
	chartVersion      string
	source            flags.LocalHelmChartSource
	reconcileStrategy string
	verifyProvider    flags.SourceOCIVerifyProvider
	verifySecretRef   string
	verifyOIDCIssuer  string
	verifySubject     string
}

var sourceChartArgs sourceChartFlags

func init() {
	createSourceChartCmd.Flags().StringVar(&sourceChartArgs.chart, "chart", "", "Helm chart name or path")
	createSourceChartCmd.Flags().StringVar(&sourceChartArgs.chartVersion, "chart-version", "", "Helm chart version, accepts a semver range (ignored for charts from GitRepository sources)")
	createSourceChartCmd.Flags().Var(&sourceChartArgs.source, "source", sourceChartArgs.source.Description())
	createSourceChartCmd.Flags().StringVar(&sourceChartArgs.reconcileStrategy, "reconcile-strategy", "ChartVersion", "the reconcile strategy for helm chart (accepted values: Revision and ChartRevision)")
	createSourceChartCmd.Flags().Var(&sourceChartArgs.verifyProvider, "verify-provider", sourceOCIRepositoryArgs.verifyProvider.Description())
	createSourceChartCmd.Flags().StringVar(&sourceChartArgs.verifySecretRef, "verify-secret-ref", "", "the name of a secret to use for signature verification")
	createSourceChartCmd.Flags().StringVar(&sourceChartArgs.verifySubject, "verify-subject", "", "regular expression to use for the OIDC subject during signature verification")
	createSourceChartCmd.Flags().StringVar(&sourceChartArgs.verifyOIDCIssuer, "verify-issuer", "", "regular expression to use for the OIDC issuer during signature verification")

	createSourceCmd.AddCommand(createSourceChartCmd)
}

func createSourceChartCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if sourceChartArgs.source.Kind == "" || sourceChartArgs.source.Name == "" {
		return fmt.Errorf("chart source is required")
	}

	if sourceChartArgs.chart == "" {
		return fmt.Errorf("chart name or path is required")
	}

	logger.Generatef("generating HelmChart source")

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	helmChart := &sourcev1.HelmChart{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *kubeconfigArgs.Namespace,
			Labels:    sourceLabels,
		},
		Spec: sourcev1.HelmChartSpec{
			Chart:   sourceChartArgs.chart,
			Version: sourceChartArgs.chartVersion,
			Interval: metav1.Duration{
				Duration: createArgs.interval,
			},
			ReconcileStrategy: sourceChartArgs.reconcileStrategy,
			SourceRef: sourcev1.LocalHelmChartSourceReference{
				Kind: sourceChartArgs.source.Kind,
				Name: sourceChartArgs.source.Name,
			},
		},
	}

	if provider := sourceChartArgs.verifyProvider.String(); provider != "" {
		helmChart.Spec.Verify = &sourcev1.OCIRepositoryVerification{
			Provider: provider,
		}
		if secretName := sourceChartArgs.verifySecretRef; secretName != "" {
			helmChart.Spec.Verify.SecretRef = &meta.LocalObjectReference{
				Name: secretName,
			}
		}
		verifyIssuer := sourceChartArgs.verifyOIDCIssuer
		verifySubject := sourceChartArgs.verifySubject
		if verifyIssuer != "" || verifySubject != "" {
			helmChart.Spec.Verify.MatchOIDCIdentity = []sourcev1.OIDCIdentityMatch{{
				Issuer:  verifyIssuer,
				Subject: verifySubject,
			}}
		}
	} else if sourceChartArgs.verifySecretRef != "" {
		return fmt.Errorf("a verification provider must be specified when a secret is specified")
	} else if sourceChartArgs.verifyOIDCIssuer != "" || sourceOCIRepositoryArgs.verifySubject != "" {
		return fmt.Errorf("a verification provider must be specified when OIDC issuer/subject is specified")
	}

	if createArgs.export {
		return printExport(exportHelmChart(helmChart))
	}

	logger.Actionf("applying HelmChart source")

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	namespacedName, err := upsertHelmChart(ctx, kubeClient, helmChart)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for HelmChart source reconciliation")
	readyConditionFunc := isObjectReadyConditionFunc(kubeClient, namespacedName, helmChart)
	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true, readyConditionFunc); err != nil {
		return err
	}
	logger.Successf("HelmChart source reconciliation completed")

	if helmChart.Status.Artifact == nil {
		return fmt.Errorf("HelmChart source reconciliation completed but no artifact was found")
	}
	logger.Successf("fetched revision: %s", helmChart.Status.Artifact.Revision)
	return nil
}

func upsertHelmChart(ctx context.Context, kubeClient client.Client,
	helmChart *sourcev1.HelmChart) (types.NamespacedName, error) {
	namespacedName := types.NamespacedName{
		Namespace: helmChart.GetNamespace(),
		Name:      helmChart.GetName(),
	}

	var existing sourcev1.HelmChart
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, helmChart); err != nil {
				return namespacedName, err
			} else {
				logger.Successf("source created")
				return namespacedName, nil
			}
		}
		return namespacedName, err
	}

	existing.Labels = helmChart.Labels
	existing.Spec = helmChart.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return namespacedName, err
	}
	helmChart = &existing
	logger.Successf("source updated")
	return namespacedName, nil
}
