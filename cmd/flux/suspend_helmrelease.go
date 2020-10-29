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
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/fluxcd/toolkit/internal/utils"
)

var suspendHrCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Suspend reconciliation of HelmRelease",
	Long:    "The suspend command disables the reconciliation of a HelmRelease resource.",
	Example: `  # Suspend reconciliation for an existing Helm release
  flux suspend hr podinfo
`,
	RunE: suspendHrCmdRun,
}

func init() {
	suspendCmd.AddCommand(suspendHrCmd)
}

func suspendHrCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("HelmRelease name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	var helmRelease helmv2.HelmRelease
	err = kubeClient.Get(ctx, namespacedName, &helmRelease)
	if err != nil {
		return err
	}

	logger.Actionf("suspending HelmRelease %s in %s namespace", name, namespace)
	helmRelease.Spec.Suspend = true
	if err := kubeClient.Update(ctx, &helmRelease); err != nil {
		return err
	}
	logger.Successf("HelmRelease suspended")

	return nil
}
