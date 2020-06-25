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

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
)

var createKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Create or update a Kustomization resource",
	Long:    "The kustomization source create command generates a Kustomize resource for a given GitRepository source.",
	Example: `  # Create a Kustomization resource from a source at a given path
  create kustomization contour \
    --source=contour \
    --path="./examples/contour/" \
    --prune=true \
    --interval=10m \
    --validate=client \
    --health-check="Deployment/contour.projectcontour" \
    --health-check="DaemonSet/envoy.projectcontour" \
    --health-check-timeout=3m

  # Create a Kustomization resource that depends on the previous one
  create kustomization webapp \
    --depends-on=contour \
    --source=webapp \
    --path="./deploy/overlays/dev" \
    --prune=true \
    --interval=5m \
    --validate=client

  # Create a Kustomization resource that runs under a service account
  create kustomization webapp \
    --source=webapp \
    --path="./deploy/overlays/staging" \
    --prune=true \
    --interval=5m \
    --validate=client \
    --sa-name=reconclier \
    --sa-namespace=staging
`,
	RunE: createKsCmdRun,
}

var (
	ksSource        string
	ksPath          string
	ksPrune         bool
	ksDependsOn     []string
	ksValidate      string
	ksHealthCheck   []string
	ksHealthTimeout time.Duration
	ksSAName        string
	ksSANamespace   string
)

func init() {
	createKsCmd.Flags().StringVar(&ksSource, "source", "", "GitRepository name")
	createKsCmd.Flags().StringVar(&ksPath, "path", "./", "path to the directory containing the Kustomization file")
	createKsCmd.Flags().BoolVar(&ksPrune, "prune", false, "enable garbage collection")
	createKsCmd.Flags().StringArrayVar(&ksHealthCheck, "health-check", nil, "workload to be included in the health assessment, in the format '<kind>/<name>.<namespace>'")
	createKsCmd.Flags().DurationVar(&ksHealthTimeout, "health-check-timeout", 2*time.Minute, "timeout of health checking operations")
	createKsCmd.Flags().StringVar(&ksValidate, "validate", "", "validate the manifests before applying them on the cluster, can be 'client' or 'server'")
	createKsCmd.Flags().StringArrayVar(&ksDependsOn, "depends-on", nil, "Kustomization that must be ready before this Kustomization can be applied")
	createKsCmd.Flags().StringVar(&ksSAName, "sa-name", "", "service account name")
	createKsCmd.Flags().StringVar(&ksSANamespace, "sa-namespace", "", "service account namespace")
	createCmd.AddCommand(createKsCmd)
}

func createKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("kustomization name is required")
	}
	name := args[0]

	if ksSource == "" {
		return fmt.Errorf("source is required")
	}
	if ksPath == "" {
		return fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(ksPath, "./") {
		return fmt.Errorf("path must begin with ./")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	if !export {
		logger.Generatef("generating kustomization")
	}

	emptyAPIGroup := ""
	kustomization := kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kustomizev1.KustomizationSpec{
			DependsOn: ksDependsOn,
			Interval: metav1.Duration{
				Duration: interval,
			},
			Path:  ksPath,
			Prune: ksPrune,
			SourceRef: corev1.TypedLocalObjectReference{
				APIGroup: &emptyAPIGroup,
				Kind:     "GitRepository",
				Name:     ksSource,
			},
			Suspend:    false,
			Validation: ksValidate,
		},
	}

	if len(ksHealthCheck) > 0 {
		healthChecks := make([]kustomizev1.WorkloadReference, 0)
		for _, w := range ksHealthCheck {
			kindObj := strings.Split(w, "/")
			if len(kindObj) != 2 {
				return fmt.Errorf("invalid health check '%s' must be in the format 'kind/name.namespace' %v", w, kindObj)
			}
			kind := kindObj[0]
			kinds := map[string]bool{
				"Deployment":  true,
				"DaemonSet":   true,
				"StatefulSet": true,
			}
			if !kinds[kind] {
				return fmt.Errorf("invalid health check kind '%s' can be Deployment, DaemonSet or StatefulSet", kind)
			}
			nameNs := strings.Split(kindObj[1], ".")
			if len(nameNs) != 2 {
				return fmt.Errorf("invalid health check '%s' must be in the format 'kind/name.namespace'", w)
			}

			healthChecks = append(healthChecks, kustomizev1.WorkloadReference{
				Kind:      kind,
				Name:      nameNs[0],
				Namespace: nameNs[1],
			})
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

	if export {
		return exportKs(kustomization)
	}

	logger.Actionf("applying kustomization")
	if err := upsertKustomization(ctx, kubeClient, kustomization); err != nil {
		return err
	}

	logger.Waitingf("waiting for kustomization sync")
	if err := wait.PollImmediate(pollInterval, timeout,
		isKustomizationReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("kustomization %s is ready", name)

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err = kubeClient.Get(ctx, namespacedName, &kustomization)
	if err != nil {
		return fmt.Errorf("kustomization sync failed: %w", err)
	}

	if kustomization.Status.LastAppliedRevision != "" {
		logger.Successf("applied revision %s", kustomization.Status.LastAppliedRevision)
	} else {
		return fmt.Errorf("kustomization sync failed")
	}

	return nil
}

func upsertKustomization(ctx context.Context, kubeClient client.Client, kustomization kustomizev1.Kustomization) error {
	namespacedName := types.NamespacedName{
		Namespace: kustomization.GetNamespace(),
		Name:      kustomization.GetName(),
	}

	var existing kustomizev1.Kustomization
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, &kustomization); err != nil {
				return err
			} else {
				logger.Successf("kustomization created")
				return nil
			}
		}
		return err
	}

	existing.Spec = kustomization.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return err
	}

	logger.Successf("kustomization updated")
	return nil
}

func isKustomizationReady(ctx context.Context, kubeClient client.Client, name, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		var kustomization kustomizev1.Kustomization
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := kubeClient.Get(ctx, namespacedName, &kustomization)
		if err != nil {
			return false, err
		}

		for _, condition := range kustomization.Status.Conditions {
			if condition.Type == sourcev1.ReadyCondition {
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
