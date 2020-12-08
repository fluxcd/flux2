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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/fluxcd/flux2/internal/utils"
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
	imageRef string
	semver   string
}

var imagePolicyArgs = imagePolicyFlags{}

func init() {
	flags := createImagePolicyCmd.Flags()
	flags.StringVar(&imagePolicyArgs.imageRef, "image-ref", "", "the name of an image repository object")
	flags.StringVar(&imagePolicyArgs.semver, "semver", "", "a semver range to apply to tags; e.g., '1.x'")

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
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: imagev1.ImagePolicySpec{
			ImageRepositoryRef: corev1.LocalObjectReference{
				Name: imagePolicyArgs.imageRef,
			},
		},
	}

	switch {
	case imagePolicyArgs.semver != "":
		policy.Spec.Policy.SemVer = &imagev1.SemVerPolicy{
			Range: imagePolicyArgs.semver,
		}
	default:
		return fmt.Errorf("a policy must be provided with --semver")
	}

	if export {
		return printExport(exportImagePolicy(&policy))
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

	logger.Generatef("generating ImagePolicy")
	logger.Actionf("applying ImagePolicy")
	namespacedName, err := upsertImagePolicy(ctx, kubeClient, &policy)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for ImagePolicy reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isReady(ctx, kubeClient, namespacedName, imagePolicyAdapter{&policy})); err != nil {
		return err
	}
	logger.Successf("ImagePolicy reconciliation completed")

	return nil
}

func upsertImagePolicy(ctx context.Context, kubeClient client.Client, policy *imagev1.ImagePolicy) (types.NamespacedName, error) {
	nsname := types.NamespacedName{
		Namespace: policy.GetNamespace(),
		Name:      policy.GetName(),
	}

	var existing imagev1.ImagePolicy
	existing.SetName(nsname.Name)
	existing.SetNamespace(nsname.Namespace)
	op, err := controllerutil.CreateOrUpdate(ctx, kubeClient, &existing, func() error {
		existing.Spec = policy.Spec
		existing.SetLabels(policy.Labels)
		return nil
	})
	if err != nil {
		return nsname, err
	}

	switch op {
	case controllerutil.OperationResultCreated:
		logger.Successf("ImagePolicy created")
	case controllerutil.OperationResultUpdated:
		logger.Successf("ImagePolicy updated")
	}
	return nsname, nil
}
