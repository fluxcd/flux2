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

	autov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

var createImageUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Create or update an ImageUpdateAutomation object",
	Long: withPreviewNote(`The create image update command generates an ImageUpdateAutomation resource.
An ImageUpdateAutomation object specifies an automated update to images
mentioned in YAMLs in a git repository.`),
	Example: `  # Configure image updates for the main repository created by flux bootstrap
  flux create image update flux-system \
    --git-repo-ref=flux-system \
    --git-repo-path="./clusters/my-cluster" \
    --checkout-branch=main \
    --author-name=flux \
    --author-email=flux@example.com \
    --commit-template="{{range .Updated.Images}}{{println .}}{{end}}"

  # Configure image updates to push changes to a different branch, if the branch doesn't exists it will be created
  flux create image update flux-system \
    --git-repo-ref=flux-system \
    --git-repo-path="./clusters/my-cluster" \
    --checkout-branch=main \
    --push-branch=image-updates \
    --author-name=flux \
    --author-email=flux@example.com \
    --commit-template="{{range .Updated.Images}}{{println .}}{{end}}"

  # Configure image updates for a Git repository in a different namespace
  flux create image update apps \
    --namespace=apps \
    --git-repo-ref=flux-system \
    --git-repo-namespace=flux-system \
    --git-repo-path="./clusters/my-cluster" \
    --checkout-branch=main \
    --push-branch=image-updates \
    --author-name=flux \
    --author-email=flux@example.com \
    --commit-template="{{range .Updated.Images}}{{println .}}{{end}}"
`,
	RunE: createImageUpdateRun,
}

type imageUpdateFlags struct {
	gitRepoName      string
	gitRepoNamespace string
	gitRepoPath      string
	checkoutBranch   string
	pushBranch       string
	commitTemplate   string
	authorName       string
	authorEmail      string
}

var imageUpdateArgs = imageUpdateFlags{}

func init() {
	flags := createImageUpdateCmd.Flags()
	flags.StringVar(&imageUpdateArgs.gitRepoName, "git-repo-ref", "", "the name of a GitRepository resource with details of the upstream Git repository")
	flags.StringVar(&imageUpdateArgs.gitRepoNamespace, "git-repo-namespace", "", "the namespace of the GitRepository resource, defaults to the ImageUpdateAutomation namespace")
	flags.StringVar(&imageUpdateArgs.gitRepoPath, "git-repo-path", "", "path to the directory containing the manifests to be updated, defaults to the repository root")
	flags.StringVar(&imageUpdateArgs.checkoutBranch, "checkout-branch", "", "the branch to checkout")
	flags.StringVar(&imageUpdateArgs.pushBranch, "push-branch", "", "the branch to push commits to, defaults to the checkout branch if not specified")
	flags.StringVar(&imageUpdateArgs.commitTemplate, "commit-template", "", "a template for commit messages")
	flags.StringVar(&imageUpdateArgs.authorName, "author-name", "", "the name to use for commit author")
	flags.StringVar(&imageUpdateArgs.authorEmail, "author-email", "", "the email to use for commit author")

	createImageCmd.AddCommand(createImageUpdateCmd)
}

func createImageUpdateRun(cmd *cobra.Command, args []string) error {
	objectName := args[0]

	if imageUpdateArgs.gitRepoName == "" {
		return fmt.Errorf("a reference to a GitRepository is required (--git-repo-ref)")
	}

	if imageUpdateArgs.checkoutBranch == "" {
		return fmt.Errorf("the Git repository branch is required (--checkout-branch)")
	}

	if imageUpdateArgs.authorName == "" {
		return fmt.Errorf("the author name is required (--author-name)")
	}

	if imageUpdateArgs.authorEmail == "" {
		return fmt.Errorf("the author email is required (--author-email)")
	}

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	var update = autov1.ImageUpdateAutomation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectName,
			Namespace: *kubeconfigArgs.Namespace,
			Labels:    labels,
		},
		Spec: autov1.ImageUpdateAutomationSpec{
			SourceRef: autov1.CrossNamespaceSourceReference{
				Kind:      sourcev1.GitRepositoryKind,
				Name:      imageUpdateArgs.gitRepoName,
				Namespace: imageUpdateArgs.gitRepoNamespace,
			},

			GitSpec: &autov1.GitSpec{
				Checkout: &autov1.GitCheckoutSpec{
					Reference: sourcev1.GitRepositoryRef{
						Branch: imageUpdateArgs.checkoutBranch,
					},
				},
				Commit: autov1.CommitSpec{
					Author: autov1.CommitUser{
						Name:  imageUpdateArgs.authorName,
						Email: imageUpdateArgs.authorEmail,
					},
					MessageTemplate: imageUpdateArgs.commitTemplate,
				},
			},
			Interval: metav1.Duration{
				Duration: createArgs.interval,
			},
		},
	}

	if imageUpdateArgs.pushBranch != "" {
		update.Spec.GitSpec.Push = &autov1.PushSpec{
			Branch: imageUpdateArgs.pushBranch,
		}
	}

	if imageUpdateArgs.gitRepoPath != "" {
		update.Spec.Update = &autov1.UpdateStrategy{
			Path:     imageUpdateArgs.gitRepoPath,
			Strategy: autov1.UpdateStrategySetters,
		}
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
