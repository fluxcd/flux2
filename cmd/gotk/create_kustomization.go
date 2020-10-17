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
	"strings"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/fluxcd/toolkit/internal/utils"
)

var createKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Create or update a Kustomization resource",
	Long:    "The kustomization source create command generates a Kustomize resource for a given source.",
	Example: `  # Create a Kustomization resource from a source at a given path
  gotk create kustomization contour \
    --source=contour \
    --path="./examples/contour/" \
    --prune=true \
    --interval=10m \
    --validation=client \
    --health-check="Deployment/contour.projectcontour" \
    --health-check="DaemonSet/envoy.projectcontour" \
    --health-check-timeout=3m

  # Create a Kustomization resource that depends on the previous one
  gotk create kustomization webapp \
    --depends-on=contour \
    --source=webapp \
    --path="./deploy/overlays/dev" \
    --prune=true \
    --interval=5m \
    --validation=client

  # Create a Kustomization resource that references a Bucket
  gotk create kustomization secrets \
    --source=Bucket/secrets \
    --prune=true \
    --interval=5m
`,
	RunE: createKsCmdRun,
}

var (
	ksSource             string
	ksPath               string
	ksPrune              bool
	ksDependsOn          []string
	ksValidation         string
	ksHealthCheck        []string
	ksHealthTimeout      time.Duration
	ksSAName             string
	ksSANamespace        string
	ksDecryptionProvider string
	ksDecryptionSecret   string
)

func init() {
	createKsCmd.Flags().StringVar(&ksSource, "source", "",
		"source that contains the Kubernetes manifests in the format '[<kind>/]<name>', where kind can be GitRepository or Bucket, if kind is not specified it defaults to GitRepository")
	createKsCmd.Flags().StringVar(&ksPath, "path", "./", "path to the directory containing the Kustomization file")
	createKsCmd.Flags().BoolVar(&ksPrune, "prune", false, "enable garbage collection")
	createKsCmd.Flags().StringArrayVar(&ksHealthCheck, "health-check", nil, "workload to be included in the health assessment, in the format '<kind>/<name>.<namespace>'")
	createKsCmd.Flags().DurationVar(&ksHealthTimeout, "health-check-timeout", 2*time.Minute, "timeout of health checking operations")
	createKsCmd.Flags().StringVar(&ksValidation, "validation", "", "validate the manifests before applying them on the cluster, can be 'client' or 'server'")
	createKsCmd.Flags().StringArrayVar(&ksDependsOn, "depends-on", nil, "Kustomization that must be ready before this Kustomization can be applied, supported formats '<name>' and '<namespace>/<name>'")
	createKsCmd.Flags().StringVar(&ksSAName, "sa-name", "", "service account name")
	createKsCmd.Flags().StringVar(&ksSANamespace, "sa-namespace", "", "service account namespace")
	createKsCmd.Flags().StringVar(&ksDecryptionProvider, "decryption-provider", "", "enables secrets decryption, provider can be 'sops'")
	createKsCmd.Flags().StringVar(&ksDecryptionSecret, "decryption-secret", "", "set the Kubernetes secret name that contains the OpenPGP private keys used for sops decryption")
	createCmd.AddCommand(createKsCmd)
}

func createKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Kustomization name is required")
	}
	name := args[0]

	if ksSource == "" {
		return fmt.Errorf("source is required")
	}

	sourceKind, sourceName := utils.ParseObjectKindName(ksSource)
	if sourceKind == "" {
		sourceKind = sourcev1.GitRepositoryKind
	}
	if !utils.ContainsItemString(supportedKustomizationSourceKinds, sourceKind) {
		return fmt.Errorf("source kind %s is not supported, can be %v",
			sourceKind, supportedKustomizationSourceKinds)
	}

	if ksPath == "" {
		return fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(ksPath, "./") {
		return fmt.Errorf("path must begin with ./")
	}

	if !export {
		logger.Generatef("generating Kustomization")
	}

	ksLabels, err := parseLabels()
	if err != nil {
		return err
	}

	kustomization := kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    ksLabels,
		},
		Spec: kustomizev1.KustomizationSpec{
			DependsOn: utils.MakeDependsOn(ksDependsOn),
			Interval: metav1.Duration{
				Duration: interval,
			},
			Path:  ksPath,
			Prune: ksPrune,
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: sourceKind,
				Name: sourceName,
			},
			Suspend:    false,
			Validation: ksValidation,
		},
	}

	if len(ksHealthCheck) > 0 {
		healthChecks := make([]kustomizev1.CrossNamespaceObjectReference, 0)
		for _, w := range ksHealthCheck {
			kindObj := strings.Split(w, "/")
			if len(kindObj) != 2 {
				return fmt.Errorf("invalid health check '%s' must be in the format 'kind/name.namespace' %v", w, kindObj)
			}
			kind := kindObj[0]

			//TODO: (stefan) extend this list with all the kstatus builtin kinds
			kinds := map[string]bool{
				"Deployment":           true,
				"DaemonSet":            true,
				"StatefulSet":          true,
				helmv2.HelmReleaseKind: true,
			}
			if !kinds[kind] {
				return fmt.Errorf("invalid health check kind '%s' can be HelmRelease, Deployment, DaemonSet or StatefulSet", kind)
			}
			nameNs := strings.Split(kindObj[1], ".")
			if len(nameNs) != 2 {
				return fmt.Errorf("invalid health check '%s' must be in the format 'kind/name.namespace'", w)
			}

			check := kustomizev1.CrossNamespaceObjectReference{
				Kind:      kind,
				Name:      nameNs[0],
				Namespace: nameNs[1],
			}

			if kind == helmv2.HelmReleaseKind {
				check.APIVersion = helmv2.GroupVersion.String()
			}
			healthChecks = append(healthChecks, check)
		}
		kustomization.Spec.HealthChecks = healthChecks
		kustomization.Spec.Timeout = &metav1.Duration{
			Duration: ksHealthTimeout,
		}
	}

	if ksSAName != "" && ksSANamespace != "" {
		kustomization.Spec.ServiceAccount = &kustomizev1.ServiceAccount{
			Name:      ksSAName,
			Namespace: ksSANamespace,
		}
	}

	if ksDecryptionProvider != "" {
		if !utils.ContainsItemString(supportedDecryptionProviders, ksDecryptionProvider) {
			return fmt.Errorf("decryption provider %s is not supported, can be %v",
				ksDecryptionProvider, supportedDecryptionProviders)
		}

		kustomization.Spec.Decryption = &kustomizev1.Decryption{
			Provider: ksDecryptionProvider,
		}

		if ksDecryptionSecret != "" {
			kustomization.Spec.Decryption.SecretRef = &corev1.LocalObjectReference{Name: ksDecryptionSecret}
		}
	}

	if export {
		return exportKs(kustomization)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig)
	if err != nil {
		return err
	}

	logger.Actionf("applying Kustomization")
	namespacedName, err := upsertKustomization(ctx, kubeClient, &kustomization)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for Kustomization reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isKustomizationReady(ctx, kubeClient, namespacedName, &kustomization)); err != nil {
		return err
	}
	logger.Successf("Kustomization %s is ready", name)

	logger.Successf("applied revision %s", kustomization.Status.LastAppliedRevision)
	return nil
}

func upsertKustomization(ctx context.Context, kubeClient client.Client,
	kustomization *kustomizev1.Kustomization) (types.NamespacedName, error) {
	namespacedName := types.NamespacedName{
		Namespace: kustomization.GetNamespace(),
		Name:      kustomization.GetName(),
	}

	var existing kustomizev1.Kustomization
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, kustomization); err != nil {
				return namespacedName, err
			} else {
				logger.Successf("Kustomization created")
				return namespacedName, nil
			}
		}
		return namespacedName, err
	}

	existing.Labels = kustomization.Labels
	existing.Spec = kustomization.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return namespacedName, err
	}
	kustomization = &existing
	logger.Successf("Kustomization updated")
	return namespacedName, nil
}

func isKustomizationReady(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, kustomization *kustomizev1.Kustomization) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, kustomization)
		if err != nil {
			return false, err
		}

		// Confirm the state we are observing is for the current generation
		if kustomization.Generation != kustomization.Status.ObservedGeneration {
			return false, nil
		}

		if c := meta.GetCondition(kustomization.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case corev1.ConditionTrue:
				return true, nil
			case corev1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}
