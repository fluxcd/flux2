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
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/fluxcd/flux2/internal/utils"
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume suspended resources",
	Long:  "The resume sub-commands resume a suspended resource.",
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}

type resumable interface {
	adapter
	statusable
	setUnsuspended()
	successMessage() string
}

type resumeCommand struct {
	apiType
	object resumable
}

func (resume resumeCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", resume.humanKind)
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

	err = kubeClient.Get(ctx, namespacedName, resume.object.asRuntimeObject())
	if err != nil {
		return err
	}

	logger.Actionf("resuming %s %s in %s namespace", resume.humanKind, name, namespace)
	resume.object.setUnsuspended()
	if err := kubeClient.Update(ctx, resume.object.asRuntimeObject()); err != nil {
		return err
	}
	logger.Successf("%s resumed", resume.humanKind)

	logger.Waitingf("waiting for %s reconciliation", resume.kind)
	if err := wait.PollImmediate(pollInterval, timeout,
		isReady(ctx, kubeClient, namespacedName, resume.object)); err != nil {
		return err
	}
	logger.Successf("%s reconciliation completed", resume.kind)
	logger.Successf(resume.object.successMessage())
	return nil
}
