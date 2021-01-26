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
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fluxcd/pkg/apis/meta"

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var createImageRepositoryCmd = &cobra.Command{
	Use:   "repository <name>",
	Short: "Create or update an ImageRepository object",
	Long: `The create image repository command generates an ImageRepository resource.
An ImageRepository object specifies an image repository to scan.`,
	RunE: createImageRepositoryRun,
}

type imageRepoFlags struct {
	image         string
	secretRef     string
	certSecretRef string
	timeout       time.Duration
}

var imageRepoArgs = imageRepoFlags{}

func init() {
	flags := createImageRepositoryCmd.Flags()
	flags.StringVar(&imageRepoArgs.image, "image", "", "the image repository to scan; e.g., library/alpine")
	flags.StringVar(&imageRepoArgs.secretRef, "secret-ref", "", "the name of a docker-registry secret to use for credentials")
	flags.StringVar(&imageRepoArgs.certSecretRef, "cert-ref", "", "the name of a secret to use for TLS certificates")
	// NB there is already a --timeout in the global flags, for
	// controlling timeout on operations while e.g., creating objects.
	flags.DurationVar(&imageRepoArgs.timeout, "scan-timeout", 0, "a timeout for scanning; this defaults to the interval if not set")

	createImageCmd.AddCommand(createImageRepositoryCmd)
}

func createImageRepositoryRun(cmd *cobra.Command, args []string) error {
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
			Namespace: rootArgs.namespace,
			Labels:    labels,
		},
		Spec: imagev1.ImageRepositorySpec{
			Image:    imageRepoArgs.image,
			Interval: metav1.Duration{Duration: createArgs.interval},
		},
	}
	if imageRepoArgs.timeout != 0 {
		repo.Spec.Timeout = &metav1.Duration{Duration: imageRepoArgs.timeout}
	}
	if imageRepoArgs.secretRef != "" {
		repo.Spec.SecretRef = &meta.LocalObjectReference{
			Name: imageRepoArgs.secretRef,
		}
	}
	if imageRepoArgs.certSecretRef != "" {
		repo.Spec.CertSecretRef = &meta.LocalObjectReference{
			Name: imageRepoArgs.certSecretRef,
		}
	}

	if createArgs.export {
		return printExport(exportImageRepository(&repo))
	}

	// a temp value for use with the rest
	var existing imagev1.ImageRepository
	copyName(&existing, &repo)
	err = imageRepositoryType.upsertAndWait(imageRepositoryAdapter{&existing}, func() error {
		existing.Spec = repo.Spec
		existing.Labels = repo.Labels
		return nil
	})
	return err
}
