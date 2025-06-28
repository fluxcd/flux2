/*
Copyright 2023 The Flux authors

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
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
)

var reconcileHelmReleaseForceCmd = &cobra.Command{
	Use:     "force-helmrelease [name]",
	Aliases: []string{"force-hr"},
	Short:   "Force reconcile a HelmRelease resource",
	Long: `The force-helmrelease command forces the reconciliation of a HelmRelease resource.
This is useful when a HelmRelease is stuck in a 'Reconciliation in progress' state 
and you want to force a new reconciliation with updated values.`,
	Example: `  # Force reconciliation for a HelmRelease
  flux reconcile force-helmrelease podinfo --namespace default`,
	Args: cobra.ExactArgs(1),
	RunE: reconcileHelmReleaseForceRun,
}

type reconcileHelmReleaseForceFlags struct {
	name      string
	namespace string
}

var forceHrArgs = reconcileHelmReleaseForceFlags{}

func init() {
	reconcileHelmReleaseForceCmd.Flags().StringVarP(&forceHrArgs.namespace, "namespace", "n", "", "The namespace of the HelmRelease")
	reconcileHelmReleaseForceCmd.Flags().BoolVarP(&rootArgs.verbose, "verbose", "v", false, "Print reconciliation details")

	reconcileCmd.AddCommand(reconcileHelmReleaseForceCmd)
}

func reconcileHelmReleaseForceRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("HelmRelease name is required")
	}
	forceHrArgs.name = args[0]

	if forceHrArgs.namespace == "" {
		forceHrArgs.namespace = rootArgs.defaults.Namespace
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: forceHrArgs.namespace,
		Name:      forceHrArgs.name,
	}

	var helmRelease helmv2beta1.HelmRelease
	err = kubeClient.Get(ctx, namespacedName, &helmRelease)
	if err != nil {
		return err
	}

	logger.Actionf("annotating HelmRelease %s in %s namespace", forceHrArgs.name, forceHrArgs.namespace)
	if helmRelease.Annotations == nil {
		helmRelease.Annotations = make(map[string]string)
	}
	helmRelease.Annotations["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339Nano)

	patch := client.MergeFrom(helmRelease.DeepCopy())
	if err := kubeClient.Patch(ctx, &helmRelease, patch); err != nil {
		return err
	}

	logger.Successf("HelmRelease annotated")
	return nil
}
