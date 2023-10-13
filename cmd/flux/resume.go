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
	"sort"
	"sync"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume suspended resources",
	Long:  `The resume sub-commands resume a suspended resource.`,
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
	isStatic() bool
	successMessage() string
}

type resumeCommand struct {
	apiType
	client          client.WithWatch
	list            listResumable
	namespace       string
	shouldReconcile bool
}

type listResumable interface {
	listAdapter
	resumeItem(i int) resumable
}

type reconcileResponse struct {
	resumable
	err error
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
	resume.client = kubeClient
	resume.namespace = *kubeconfigArgs.Namespace

	// require waiting for the object(s) if the user has not provided the --wait flag and gave exactly
	// one object to resume. This is necessary to maintain backwards compatibility with prior versions
	// of this command. Otherwise just follow the value of the --wait flag (including its default).
	resume.shouldReconcile = !resumeCmd.PersistentFlags().Changed("wait") && len(args) == 1 || resumeArgs.wait

	resumables, err := resume.getPatchedResumables(ctx, args)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(resumables))

	resultChan := make(chan reconcileResponse, len(resumables))
	for _, r := range resumables {
		go func(res resumable) {
			defer wg.Done()
			resultChan <- resume.reconcile(ctx, res)
		}(r)
	}

	go func() {
		defer close(resultChan)
		wg.Wait()
	}()

	reconcileResps := make([]reconcileResponse, 0, len(resumables))
	for c := range resultChan {
		reconcileResps = append(reconcileResps, c)
	}

	resume.printMessage(reconcileResps)

	return nil
}

// getPatchedResumables returns a list of the given resumable objects that have been patched to be resumed.
// If the args slice is empty, it patches all resumable objects in the given namespace.
func (resume *resumeCommand) getPatchedResumables(ctx context.Context, args []string) ([]resumable, error) {
	if len(args) < 1 {
		objs, err := resume.patch(ctx, []client.ListOption{
			client.InNamespace(resume.namespace),
		})
		if err != nil {
			return nil, fmt.Errorf("failed patching objects: %w", err)
		}

		return objs, nil
	}

	var resumables []resumable
	processed := make(map[string]struct{}, len(args))
	for _, arg := range args {
		if _, has := processed[arg]; has {
			continue // skip object that user might have provided more than once
		}
		processed[arg] = struct{}{}

		objs, err := resume.patch(ctx, []client.ListOption{
			client.InNamespace(resume.namespace),
			client.MatchingFields{
				"metadata.name": arg,
			},
		})
		if err != nil {
			return nil, err
		}

		resumables = append(resumables, objs...)
	}

	return resumables, nil
}

// Patches resumable objects by setting their status to unsuspended.
// Returns a slice of resumables that have been patched and any error encountered during patching.
func (resume resumeCommand) patch(ctx context.Context, listOpts []client.ListOption) ([]resumable, error) {
	if err := resume.client.List(ctx, resume.list.asClientList(), listOpts...); err != nil {
		return nil, err
	}

	if resume.list.len() == 0 {
		logger.Failuref("no %s objects found in %s namespace", resume.kind, resume.namespace)
		return nil, nil
	}

	var resumables []resumable

	for i := 0; i < resume.list.len(); i++ {
		obj := resume.list.resumeItem(i)
		logger.Actionf("resuming %s %s in %s namespace", resume.humanKind, obj.asClientObject().GetName(), resume.namespace)

		patch := client.MergeFrom(obj.deepCopyClientObject())
		obj.setUnsuspended()
		if err := resume.client.Patch(ctx, obj.asClientObject(), patch); err != nil {
			return nil, err
		}

		resumables = append(resumables, obj)

		logger.Successf("%s resumed", resume.humanKind)
	}

	return resumables, nil
}

// Waits for resumable object to be reconciled and returns the object and any error encountered while waiting.
// Returns an empty reconcileResponse, if shouldReconcile is false.
func (resume resumeCommand) reconcile(ctx context.Context, res resumable) reconcileResponse {
	if !resume.shouldReconcile {
		return reconcileResponse{}
	}

	namespacedName := types.NamespacedName{
		Name:      res.asClientObject().GetName(),
		Namespace: resume.namespace,
	}

	logger.Waitingf("waiting for %s reconciliation", resume.kind)

	readyConditionFunc := isObjectReadyConditionFunc(resume.client, namespacedName, res.asClientObject())
	if res.isStatic() {
		readyConditionFunc = isStaticObjectReadyConditionFunc(resume.client, namespacedName, res.asClientObject())
	}

	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true, readyConditionFunc); err != nil {
		return reconcileResponse{
			resumable: res,
			err:       err,
		}
	}

	return reconcileResponse{
		resumable: res,
		err:       nil,
	}
}

// Sorts the given reconcileResponses by resumable name and prints the success/error message for each response.
func (resume resumeCommand) printMessage(responses []reconcileResponse) {
	sort.Slice(responses, func(i, j int) bool {
		r1, r2 := responses[i], responses[j]
		if r1.resumable == nil || r2.resumable == nil {
			return false
		}
		return r1.asClientObject().GetName() <= r2.asClientObject().GetName()
	})

	// Print success/error message.
	for _, r := range responses {
		if r.resumable == nil {
			continue
		}
		if r.err != nil {
			logger.Failuref(r.err.Error())
		}
		logger.Successf("%s %s reconciliation completed", resume.kind, r.asClientObject().GetName())
		logger.Successf(r.successMessage())
	}
}
