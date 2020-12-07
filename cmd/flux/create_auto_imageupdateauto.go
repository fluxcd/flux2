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
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/fluxcd/flux2/internal/utils"
	autov1 "github.com/fluxcd/image-automation-controller/api/v1alpha1"
	"github.com/fluxcd/pkg/apis/meta"
)

var createAutoImageUpdateCmd = &cobra.Command{
	Use:   "image-update <name>",
	Short: "Create or update an ImageUpdateAutomation object",
	Long: `The create auto image-update command generates an ImageUpdateAutomation resource.
An ImageUpdateAutomation object specifies an automated update to images
mentioned in YAMLs in a git repository.`,
	RunE: createAutoImageUpdateRun,
}

type imageUpdateFlags struct {
	// git checkout spec
	gitRepoRef string
	branch     string
	// commit spec
	commitTemplate string
	authorName     string
	authorEmail    string
}

var imageUpdateArgs = imageUpdateFlags{}

func init() {
	flags := createAutoImageUpdateCmd.Flags()
	flags.StringVar(&imageUpdateArgs.gitRepoRef, "git-repo-ref", "", "the name of a GitRepository resource with details of the upstream git repository")
	flags.StringVar(&imageUpdateArgs.branch, "branch", "", "the branch to push commits to")
	flags.StringVar(&imageUpdateArgs.commitTemplate, "commit-template", "", "a template for commit messages")
	flags.StringVar(&imageUpdateArgs.authorName, "author-name", "", "the name to use for commit author")
	flags.StringVar(&imageUpdateArgs.authorEmail, "author-email", "", "the email to use for commit author")

	createAutoCmd.AddCommand(createAutoImageUpdateCmd)
}

func createAutoImageUpdateRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ImageUpdateAutomation name is required")
	}
	objectName := args[0]

	if imageUpdateArgs.gitRepoRef == "" {
		return fmt.Errorf("a reference to a GitRepository is required (--git-repo-ref)")
	}

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	var update = autov1.ImageUpdateAutomation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: autov1.ImageUpdateAutomationSpec{
			Checkout: autov1.GitCheckoutSpec{
				GitRepositoryRef: corev1.LocalObjectReference{
					Name: imageUpdateArgs.gitRepoRef,
				},
				Branch: imageUpdateArgs.branch,
			},
			Interval: metav1.Duration{Duration: interval},
			Update: autov1.UpdateStrategy{
				Setters: &autov1.SettersStrategy{},
			},
			Commit: autov1.CommitSpec{
				AuthorName:      imageUpdateArgs.authorName,
				AuthorEmail:     imageUpdateArgs.authorEmail,
				MessageTemplate: imageUpdateArgs.commitTemplate,
			},
		},
	}

	if export {
		return printExport(exportImageUpdate(&update))
	}

	// I don't need these until attempting to upsert the object, but
	// for consistency with other create commands, the following are
	// given a chance to error out before reporting any progress.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return err
	}

	logger.Generatef("generating ImageUpdateAutomation")
	logger.Actionf("applying ImageUpdateAutomation")
	namespacedName, err := upsertImageUpdateAutomation(ctx, kubeClient, &update)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for ImageUpdateAutomation reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isImageUpdateAutomationReady(ctx, kubeClient, namespacedName, &update)); err != nil {
		return err
	}
	logger.Successf("ImageUpdateAutomation reconciliation completed")

	return nil
}

func upsertImageUpdateAutomation(ctx context.Context, kubeClient client.Client, update *autov1.ImageUpdateAutomation) (types.NamespacedName, error) {
	nsname := types.NamespacedName{
		Namespace: update.GetNamespace(),
		Name:      update.GetName(),
	}

	var existing autov1.ImageUpdateAutomation
	existing.SetName(nsname.Name)
	existing.SetNamespace(nsname.Namespace)
	op, err := controllerutil.CreateOrUpdate(ctx, kubeClient, &existing, func() error {
		existing.Spec = update.Spec
		existing.Labels = update.Labels
		return nil
	})
	if err != nil {
		return nsname, err
	}

	switch op {
	case controllerutil.OperationResultCreated:
		logger.Successf("ImageUpdateAutomation created")
	case controllerutil.OperationResultUpdated:
		logger.Successf("ImageUpdateAutomation updated")
	}
	return nsname, nil
}

func isImageUpdateAutomationReady(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, update *autov1.ImageUpdateAutomation) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, update)
		if err != nil {
			return false, err
		}

		// Confirm the state we are observing is for the current generation
		if update.Generation != update.Status.ObservedGeneration {
			return false, nil
		}

		if c := apimeta.FindStatusCondition(update.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case metav1.ConditionTrue:
				return true, nil
			case metav1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}
