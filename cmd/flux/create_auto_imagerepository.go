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
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/fluxcd/flux2/internal/utils"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
	"github.com/fluxcd/pkg/apis/meta"
)

var createAutoImageRepositoryCmd = &cobra.Command{
	Use:   "image-repository <name>",
	Short: "Create or update an ImageRepository object",
	Long: `The create auto image-repository command generates an ImageRepository resource.
An ImageRepository object specifies an image repository to scan.`,
	RunE: createAutoImageRepositoryRun,
}

type imageRepoFlags struct {
	image     string
	secretRef string
	timeout   time.Duration
}

var imageRepoArgs = imageRepoFlags{}

func init() {
	flags := createAutoImageRepositoryCmd.Flags()
	flags.StringVar(&imageRepoArgs.image, "image", "", "the image repository to scan; e.g., library/alpine")
	flags.StringVar(&imageRepoArgs.secretRef, "secret-ref", "", "the name of a docker-registry secret to use for credentials")
	// NB there is already a --timeout in the global flags, for
	// controlling timeout on operations while e.g., creating objects.
	flags.DurationVar(&imageRepoArgs.timeout, "scan-timeout", 0, "a timeout for scanning; this defaults to the interval if not set")

	createAutoCmd.AddCommand(createAutoImageRepositoryCmd)
}

func createAutoImageRepositoryRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ImageRepository name is required")
	}
	objectName := args[0]

	if imageRepoArgs.image == "" {
		return fmt.Errorf("an image repository (--image) is required")
	}

	if _, err := name.NewRepository(imageRepoArgs.image); err != nil {
		return fmt.Errorf("unable to parse image value: %w", err)
	}

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	var repo = imagev1.ImageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: imagev1.ImageRepositorySpec{
			Image:    imageRepoArgs.image,
			Interval: metav1.Duration{Duration: interval},
		},
	}
	if imageRepoArgs.timeout != 0 {
		repo.Spec.Timeout = &metav1.Duration{Duration: imageRepoArgs.timeout}
	}
	if imageRepoArgs.secretRef != "" {
		repo.Spec.SecretRef = &corev1.LocalObjectReference{
			Name: imageRepoArgs.secretRef,
		}
	}

	if export {
		return exportImageRepo(repo) // defined with export command
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

	logger.Generatef("generating ImageRepository")
	logger.Actionf("applying ImageRepository")
	namespacedName, err := upsertImageRepository(ctx, kubeClient, &repo)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for ImageRepository reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isImageRepositoryReady(ctx, kubeClient, namespacedName, &repo)); err != nil {
		return err
	}
	logger.Successf("ImageRepository reconciliation completed")

	return nil
}

func upsertImageRepository(ctx context.Context, kubeClient client.Client, repo *imagev1.ImageRepository) (types.NamespacedName, error) {
	nsname := types.NamespacedName{
		Namespace: repo.GetNamespace(),
		Name:      repo.GetName(),
	}

	var existing imagev1.ImageRepository
	existing.SetName(nsname.Name)
	existing.SetNamespace(nsname.Namespace)
	op, err := controllerutil.CreateOrUpdate(ctx, kubeClient, &existing, func() error {
		existing.Spec = repo.Spec
		existing.Labels = repo.Labels
		return nil
	})
	if err != nil {
		return nsname, err
	}

	switch op {
	case controllerutil.OperationResultCreated:
		logger.Successf("ImageRepository created")
	case controllerutil.OperationResultUpdated:
		logger.Successf("ImageRepository updated")
	}
	return nsname, nil
}

func isImageRepositoryReady(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, imageRepository *imagev1.ImageRepository) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, imageRepository)
		if err != nil {
			return false, err
		}

		// Confirm the state we are observing is for the current generation
		if imageRepository.Generation != imageRepository.Status.ObservedGeneration {
			return false, nil
		}

		if c := apimeta.FindStatusCondition(imageRepository.Status.Conditions, meta.ReadyCondition); c != nil {
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
