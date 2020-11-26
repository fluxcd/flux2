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
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

var suspendSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Suspend reconciliation of a GitRepository",
	Long:  "The suspend command disables the reconciliation of a GitRepository resource.",
	Example: `  # Suspend reconciliation for an existing GitRepository
  flux suspend source git podinfo
`,
	RunE: suspendSourceGitCmdRun,
}

func init() {
	suspendSourceCmd.AddCommand(suspendSourceGitCmd)
}

func suspendSourceGitCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("source name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	var repository sourcev1.GitRepository
	err = kubeClient.Get(ctx, namespacedName, &repository)
	if err != nil {
		return err
	}

	logger.Actionf("suspending source %s in %s namespace", name, namespace)
	repository.Spec.Suspend = true
	if err := kubeClient.Update(ctx, &repository); err != nil {
		return err
	}
	logger.Successf("source suspended")

	return nil
}
