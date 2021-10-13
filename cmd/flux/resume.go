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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/internal/utils"
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume suspended resources",
	Long:  "The resume sub-commands resume a suspended resource.",
}

type ResumeFlags struct {
	all bool
}

var resumeArgs ResumeFlags

func init() {
	resumeCmd.PersistentFlags().BoolVarP(&resumeArgs.all, "all", "", false,
		"resume all resources in that namespace")
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
	list   listResumable
}

type listResumable interface {
	listAdapter
	resumeItem(i int) resumable
}

func (resume resumeCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) < 1 && !resumeArgs.all {
		return fmt.Errorf("%s name is required", resume.humanKind)
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	var listOpts []client.ListOption
	listOpts = append(listOpts, client.InNamespace(rootArgs.namespace))
	if len(args) > 0 {
		listOpts = append(listOpts, client.MatchingFields{
			"metadata.name": args[0],
		})
	}

	err = kubeClient.List(ctx, resume.list.asClientList(), listOpts...)
	if err != nil {
		return err
	}

	if resume.list.len() == 0 {
		logger.Failuref("no %s objects found in %s namespace", resume.kind, rootArgs.namespace)
		return nil
	}

	for i := 0; i < resume.list.len(); i++ {
		logger.Actionf("resuming %s %s in %s namespace", resume.humanKind, resume.list.resumeItem(i).asClientObject().GetName(), rootArgs.namespace)
		resume.list.resumeItem(i).setUnsuspended()
		if err := kubeClient.Update(ctx, resume.list.resumeItem(i).asClientObject()); err != nil {
			return err
		}
		logger.Successf("%s resumed", resume.humanKind)

		namespacedName := types.NamespacedName{
			Name:      resume.list.resumeItem(i).asClientObject().GetName(),
			Namespace: rootArgs.namespace,
		}

		logger.Waitingf("waiting for %s reconciliation", resume.kind)
		if err := wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
			isReady(ctx, kubeClient, namespacedName, resume.list.resumeItem(i))); err != nil {
			logger.Failuref(err.Error())
			continue
		}
		logger.Successf("%s reconciliation completed", resume.kind)
		logger.Successf(resume.list.resumeItem(i).successMessage())
	}

	return nil
}
