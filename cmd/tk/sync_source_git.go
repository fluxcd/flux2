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
	sourcev1 "github.com/fluxcd/source-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

var syncSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Synchronize a GitRepository source",
	Long:  `The sync source command triggers a reconciliation of a GitRepository resource and waits for it to finish.`,
	Example: `  # Trigger a git pull for an existing source
  sync source git podinfo
`,
	RunE: syncSourceGitCmdRun,
}

func init() {
	syncSourceCmd.AddCommand(syncSourceGitCmd)
}

func syncSourceGitCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("source name is required")
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	logger.Actionf("annotating source %s in %s namespace", name, namespace)
	var gitRepository sourcev1.GitRepository
	err = kubeClient.Get(ctx, namespacedName, &gitRepository)
	if err != nil {
		return err
	}

	if gitRepository.Annotations == nil {
		gitRepository.Annotations = map[string]string{
			sourcev1.SyncAtAnnotation: time.Now().String(),
		}
	} else {
		gitRepository.Annotations[sourcev1.SyncAtAnnotation] = time.Now().String()
	}
	if err := kubeClient.Update(ctx, &gitRepository); err != nil {
		return err
	}
	logger.Successf("source annotated")

	logger.Waitingf("waiting for git sync")
	if err := wait.PollImmediate(pollInterval, timeout,
		isGitRepositoryReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("git sync completed")

	err = kubeClient.Get(ctx, namespacedName, &gitRepository)
	if err != nil {
		return err
	}

	if gitRepository.Status.Artifact != nil {
		logger.Successf("fetched revision %s", gitRepository.Status.Artifact.Revision)
	} else {
		return fmt.Errorf("git sync failed, artifact not found")
	}
	return nil
}
