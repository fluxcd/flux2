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
	"time"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var reconcileSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Reconcile a GitRepository source",
	Long:  `The reconcile source command triggers a reconciliation of a GitRepository resource and waits for it to finish.`,
	Example: `  # Trigger a git pull for an existing source
  flux reconcile source git podinfo
`,
	RunE: reconcileSourceGitCmdRun,
}

func init() {
	reconcileSourceCmd.AddCommand(reconcileSourceGitCmd)
}

func reconcileSourceGitCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("source name is required")
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

	logger.Actionf("annotating GitRepository source %s in %s namespace", name, namespace)
	var gitRepository sourcev1.GitRepository
	err = kubeClient.Get(ctx, namespacedName, &gitRepository)
	if err != nil {
		return err
	}

	if gitRepository.Annotations == nil {
		gitRepository.Annotations = map[string]string{
			meta.ReconcileAtAnnotation: time.Now().Format(time.RFC3339Nano),
		}
	} else {
		gitRepository.Annotations[meta.ReconcileAtAnnotation] = time.Now().Format(time.RFC3339Nano)
	}
	if err := kubeClient.Update(ctx, &gitRepository); err != nil {
		return err
	}
	logger.Successf("GitRepository source annotated")

	logger.Waitingf("waiting for GitRepository source reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isGitRepositoryReady(ctx, kubeClient, namespacedName, &gitRepository)); err != nil {
		return err
	}
	logger.Successf("GitRepository source reconciliation completed")

	if gitRepository.Status.Artifact == nil {
		return fmt.Errorf("GitRepository source reconciliation completed but no artifact was found")
	}
	logger.Successf("fetched revision %s", gitRepository.Status.Artifact.Revision)
	return nil
}
