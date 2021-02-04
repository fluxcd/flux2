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

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var createImagePolicyCmd = &cobra.Command{
	Use:   "policy <name>",
	Short: "Create or update an ImagePolicy object",
	Long: `The create image policy command generates an ImagePolicy resource.
An ImagePolicy object calculates a "latest image" given an image
repository and a policy, e.g., semver.

The image that sorts highest according to the policy is recorded in
the status of the object.`,
	RunE: createImagePolicyRun}

type imagePolicyFlags struct {
	imageRef      string
	semver        string
	alpha         string
	filterRegex   string
	filterExtract string
}

var imagePolicyArgs = imagePolicyFlags{}

func init() {
	flags := createImagePolicyCmd.Flags()
	flags.StringVar(&imagePolicyArgs.imageRef, "image-ref", "", "the name of an image repository object")
	flags.StringVar(&imagePolicyArgs.semver, "select-semver", "", "a semver range to apply to tags; e.g., '1.x'")
	flags.StringVar(&imagePolicyArgs.alpha, "select-alpha", "", "use alphabetical sorting to select image; either \"asc\" meaning select the last, or \"desc\" meaning select the first")
	flags.StringVar(&imagePolicyArgs.filterRegex, "filter-regex", "", "regular expression pattern used to filter the image tags")
	flags.StringVar(&imagePolicyArgs.filterExtract, "filter-extract", "", "replacement pattern (using capture groups from --filter-regex) to use for sorting")

	createImageCmd.AddCommand(createImagePolicyCmd)
}

// getObservedGeneration is implemented here, since it's not
// (presently) needed elsewhere.
func (obj imagePolicyAdapter) getObservedGeneration() int64 {
	return obj.ImagePolicy.Status.ObservedGeneration
}

func createImagePolicyRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ImagePolicy name is required")
	}
	objectName := args[0]

	if imagePolicyArgs.imageRef == "" {
		return fmt.Errorf("the name of an ImageRepository in the namespace is required (--image-ref)")
	}

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	var policy = imagev1.ImagePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectName,
			Namespace: rootArgs.namespace,
			Labels:    labels,
		},
		Spec: imagev1.ImagePolicySpec{
			ImageRepositoryRef: meta.LocalObjectReference{
				Name: imagePolicyArgs.imageRef,
			},
		},
	}

	switch {
	case imagePolicyArgs.semver != "":
		policy.Spec.Policy.SemVer = &imagev1.SemVerPolicy{
			Range: imagePolicyArgs.semver,
		}
	case imagePolicyArgs.alpha != "":
		if imagePolicyArgs.alpha != "desc" && imagePolicyArgs.alpha != "asc" {
			return fmt.Errorf("--select-alpha must be one of [\"asc\", \"desc\"]")
		}
		policy.Spec.Policy.Alphabetical = &imagev1.AlphabeticalPolicy{
			Order: imagePolicyArgs.alpha,
		}
	default:
		return fmt.Errorf("a policy must be provided with either --select-semver or --select-alpha")
	}

	if imagePolicyArgs.filterRegex != "" {
		policy.Spec.FilterTags = &imagev1.TagFilter{
			Pattern: imagePolicyArgs.filterRegex,
		}

		if imagePolicyArgs.filterExtract != "" {
			policy.Spec.FilterTags.Extract = imagePolicyArgs.filterExtract
		}
	} else if imagePolicyArgs.filterExtract != "" {
		return fmt.Errorf("cannot specify --filter-extract without specifying --filter-regex")
	}

	if createArgs.export {
		return printExport(exportImagePolicy(&policy))
	}

	var existing imagev1.ImagePolicy
	copyName(&existing, &policy)
	err = imagePolicyType.upsertAndWait(imagePolicyAdapter{&existing}, func() error {
		existing.Spec = policy.Spec
		existing.SetLabels(policy.Labels)
		return nil
	})
	return err
}
