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
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fluxcd/pkg/apis/meta"

	autov1 "github.com/fluxcd/image-automation-controller/api/v1alpha1"
)

var createImageUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Create or update an ImageUpdateAutomation object",
	Long: `The create image update command generates an ImageUpdateAutomation resource.
An ImageUpdateAutomation object specifies an automated update to images
mentioned in YAMLs in a git repository.`,
	RunE: createImageUpdateRun,
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
	flags := createImageUpdateCmd.Flags()
	flags.StringVar(&imageUpdateArgs.gitRepoRef, "git-repo-ref", "", "the name of a GitRepository resource with details of the upstream git repository")
	flags.StringVar(&imageUpdateArgs.branch, "branch", "", "the branch to checkout and push commits to")
	flags.StringVar(&imageUpdateArgs.commitTemplate, "commit-template", "", "a template for commit messages")
	flags.StringVar(&imageUpdateArgs.authorName, "author-name", "", "the name to use for commit author")
	flags.StringVar(&imageUpdateArgs.authorEmail, "author-email", "", "the email to use for commit author")

	createImageCmd.AddCommand(createImageUpdateCmd)
}

func createImageUpdateRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ImageUpdateAutomation name is required")
	}
	objectName := args[0]

	if imageUpdateArgs.gitRepoRef == "" {
		return fmt.Errorf("a reference to a GitRepository is required (--git-repo-ref)")
	}

	if imageUpdateArgs.branch == "" {
		return fmt.Errorf("the Git repoistory branch is required (--branch)")
	}

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	var update = autov1.ImageUpdateAutomation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectName,
			Namespace: rootArgs.namespace,
			Labels:    labels,
		},
		Spec: autov1.ImageUpdateAutomationSpec{
			Checkout: autov1.GitCheckoutSpec{
				GitRepositoryRef: meta.LocalObjectReference{
					Name: imageUpdateArgs.gitRepoRef,
				},
				Branch: imageUpdateArgs.branch,
			},
			Interval: metav1.Duration{Duration: createArgs.interval},
			Commit: autov1.CommitSpec{
				AuthorName:      imageUpdateArgs.authorName,
				AuthorEmail:     imageUpdateArgs.authorEmail,
				MessageTemplate: imageUpdateArgs.commitTemplate,
			},
		},
	}

	if createArgs.export {
		return printExport(exportImageUpdate(&update))
	}

	var existing autov1.ImageUpdateAutomation
	copyName(&existing, &update)
	err = imageUpdateAutomationType.upsertAndWait(imageUpdateAutomationAdapter{&existing}, func() error {
		existing.Spec = update.Spec
		existing.Labels = update.Labels
		return nil
	})
	return err
}
