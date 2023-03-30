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

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume suspended resources",
	Long:  "The resume sub-commands resume a suspended resource.",
}

type ResumeFlags struct {
	all  bool
	wait bool
}

var resumeArgs ResumeFlags

func init() {
	resumeCmd.PersistentFlags().BoolVarP(&resumeArgs.all, "all", "", false,
		"resume all resources in that namespace")
	resumeCmd.PersistentFlags().BoolVarP(&resumeArgs.wait, "wait", "", false,
		"waits for one resource to reconcile before moving to the next one")
	rootCmd.AddCommand(resumeCmd)
}

type resumable interface {
	adapter
	copyable
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

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	var listOpts []client.ListOption
	listOpts = append(listOpts, client.InNamespace(*kubeconfigArgs.Namespace))
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
		logger.Failuref("no %s objects found in %s namespace", resume.kind, *kubeconfigArgs.Namespace)
		return nil
	}

	for i := 0; i < resume.list.len(); i++ {
		logger.Actionf("resuming %s %s in %s namespace", resume.humanKind, resume.list.resumeItem(i).asClientObject().GetName(), *kubeconfigArgs.Namespace)
		obj := resume.list.resumeItem(i)
		patch := client.MergeFrom(obj.deepCopyClientObject())
		obj.setUnsuspended()
		if err := kubeClient.Patch(ctx, obj.asClientObject(), patch); err != nil {
			return err
		}

		logger.Successf("%s resumed", resume.humanKind)

		if resumeArgs.wait || !resumeArgs.all {
			namespacedName := types.NamespacedName{
				Name:      resume.list.resumeItem(i).asClientObject().GetName(),
				Namespace: *kubeconfigArgs.Namespace,
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
	}

	return nil
}
