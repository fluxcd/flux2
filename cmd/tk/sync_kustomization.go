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
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var syncKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Synchronize a Kustomization resource",
	Long: `
The sync kustomization command triggers a reconciliation of a Kustomization resource and waits for it to finish.`,
	Example: `  # Trigger a Kustomization apply outside of the reconciliation interval
  sync kustomization podinfo

  # Trigger a sync of the Kustomization's source and apply changes
  sync kustomization podinfo --with-source
`,
	RunE: syncKsCmdRun,
}

var (
	syncKsWithSource bool
)

func init() {
	syncKsCmd.Flags().BoolVar(&syncKsWithSource, "with-source", false, "synchronize kustomization source")

	syncCmd.AddCommand(syncKsCmd)
}

func syncKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("kustomization name is required")
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

	var kustomization kustomizev1.Kustomization
	err = kubeClient.Get(ctx, namespacedName, &kustomization)
	if err != nil {
		return err
	}

	if syncKsWithSource {
		err := syncSourceGitCmdRun(nil, []string{kustomization.Spec.SourceRef.Name})
		if err != nil {
			return err
		}
	} else {
		logger.Actionf("annotating kustomization %s in %s namespace", name, namespace)
		if kustomization.Annotations == nil {
			kustomization.Annotations = map[string]string{
				kustomizev1.SyncAtAnnotation: time.Now().String(),
			}
		} else {
			kustomization.Annotations[kustomizev1.SyncAtAnnotation] = time.Now().String()
		}
		if err := kubeClient.Update(ctx, &kustomization); err != nil {
			return err
		}
		logger.Successf("kustomization annotated")
	}

	logger.Waitingf("waiting for kustomization sync")
	if err := wait.PollImmediate(pollInterval, timeout,
		isKustomizationReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("kustomization sync completed")

	err = kubeClient.Get(ctx, namespacedName, &kustomization)
	if err != nil {
		return err
	}

	if kustomization.Status.LastAppliedRevision != "" {
		logger.Successf("applied revision %s", kustomization.Status.LastAppliedRevision)
	} else {
		return fmt.Errorf("kustomization sync failed")
	}
	return nil
}
