/*
Copyright 2022 The Flux authors

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
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/apis/meta"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/internal/utils"
)

var createSourceOCIRepositoryCmd = &cobra.Command{
	Use:   "oci [name]",
	Short: "Create or update an OCIRepository",
	Long:  withPreviewNote(`The create source oci command generates an OCIRepository resource and waits for it to be ready.`),
	Example: `  # Create an OCIRepository for a public container image
  flux create source oci podinfo \
    --url=oci://ghcr.io/stefanprodan/manifests/podinfo \
    --tag=6.1.6 \
    --interval=10m
`,
	RunE: createSourceOCIRepositoryCmdRun,
}

type sourceOCIRepositoryFlags struct {
	url             string
	tag             string
	semver          string
	digest          string
	secretRef       string
	serviceAccount  string
	certSecretRef   string
	verifyProvider  flags.SourceOCIVerifyProvider
	verifySecretRef string
	ignorePaths     []string
	provider        flags.SourceOCIProvider
	insecure        bool
}

var sourceOCIRepositoryArgs = newSourceOCIFlags()

func newSourceOCIFlags() sourceOCIRepositoryFlags {
	return sourceOCIRepositoryFlags{
		provider: flags.SourceOCIProvider(sourcev1.GenericOCIProvider),
	}
}

func init() {
	createSourceOCIRepositoryCmd.Flags().Var(&sourceOCIRepositoryArgs.provider, "provider", sourceOCIRepositoryArgs.provider.Description())
	createSourceOCIRepositoryCmd.Flags().StringVar(&sourceOCIRepositoryArgs.url, "url", "", "the OCI repository URL")
	createSourceOCIRepositoryCmd.Flags().StringVar(&sourceOCIRepositoryArgs.tag, "tag", "", "the OCI artifact tag")
	createSourceOCIRepositoryCmd.Flags().StringVar(&sourceOCIRepositoryArgs.semver, "tag-semver", "", "the OCI artifact tag semver range")
	createSourceOCIRepositoryCmd.Flags().StringVar(&sourceOCIRepositoryArgs.digest, "digest", "", "the OCI artifact digest")
	createSourceOCIRepositoryCmd.Flags().StringVar(&sourceOCIRepositoryArgs.secretRef, "secret-ref", "", "the name of the Kubernetes image pull secret (type 'kubernetes.io/dockerconfigjson')")
	createSourceOCIRepositoryCmd.Flags().StringVar(&sourceOCIRepositoryArgs.serviceAccount, "service-account", "", "the name of the Kubernetes service account that refers to an image pull secret")
	createSourceOCIRepositoryCmd.Flags().StringVar(&sourceOCIRepositoryArgs.certSecretRef, "cert-ref", "", "the name of a secret to use for TLS certificates")
	createSourceOCIRepositoryCmd.Flags().Var(&sourceOCIRepositoryArgs.verifyProvider, "verify-provider", sourceOCIRepositoryArgs.verifyProvider.Description())
	createSourceOCIRepositoryCmd.Flags().StringVar(&sourceOCIRepositoryArgs.verifySecretRef, "verify-secret-ref", "", "the name of a secret to use for signature verification")
	createSourceOCIRepositoryCmd.Flags().StringSliceVar(&sourceOCIRepositoryArgs.ignorePaths, "ignore-paths", nil, "set paths to ignore resources (can specify multiple paths with commas: path1,path2)")
	createSourceOCIRepositoryCmd.Flags().BoolVar(&sourceOCIRepositoryArgs.insecure, "insecure", false, "for when connecting to a non-TLS registries over plain HTTP")

	createSourceCmd.AddCommand(createSourceOCIRepositoryCmd)
}

func createSourceOCIRepositoryCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if sourceOCIRepositoryArgs.url == "" {
		return fmt.Errorf("url is required")
	}

	if sourceOCIRepositoryArgs.semver == "" && sourceOCIRepositoryArgs.tag == "" && sourceOCIRepositoryArgs.digest == "" {
		return fmt.Errorf("--tag, --tag-semver or --digest is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	var ignorePaths *string
	if len(sourceOCIRepositoryArgs.ignorePaths) > 0 {
		ignorePathsStr := strings.Join(sourceOCIRepositoryArgs.ignorePaths, "\n")
		ignorePaths = &ignorePathsStr
	}

	repository := &sourcev1.OCIRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *kubeconfigArgs.Namespace,
			Labels:    sourceLabels,
		},
		Spec: sourcev1.OCIRepositorySpec{
			Provider: sourceOCIRepositoryArgs.provider.String(),
			URL:      sourceOCIRepositoryArgs.url,
			Insecure: sourceOCIRepositoryArgs.insecure,
			Interval: metav1.Duration{
				Duration: createArgs.interval,
			},
			Reference: &sourcev1.OCIRepositoryRef{},
			Ignore:    ignorePaths,
		},
	}

	if digest := sourceOCIRepositoryArgs.digest; digest != "" {
		repository.Spec.Reference.Digest = digest
	}
	if semver := sourceOCIRepositoryArgs.semver; semver != "" {
		repository.Spec.Reference.SemVer = semver
	}
	if tag := sourceOCIRepositoryArgs.tag; tag != "" {
		repository.Spec.Reference.Tag = tag
	}

	if createSourceArgs.fetchTimeout > 0 {
		repository.Spec.Timeout = &metav1.Duration{Duration: createSourceArgs.fetchTimeout}
	}

	if saName := sourceOCIRepositoryArgs.serviceAccount; saName != "" {
		repository.Spec.ServiceAccountName = saName
	}

	if secretName := sourceOCIRepositoryArgs.secretRef; secretName != "" {
		repository.Spec.SecretRef = &meta.LocalObjectReference{
			Name: secretName,
		}
	}

	if secretName := sourceOCIRepositoryArgs.certSecretRef; secretName != "" {
		repository.Spec.CertSecretRef = &meta.LocalObjectReference{
			Name: secretName,
		}
	}

	if provider := sourceOCIRepositoryArgs.verifyProvider.String(); provider != "" {
		repository.Spec.Verify = &sourcev1.OCIRepositoryVerification{
			Provider: provider,
		}
		if secretName := sourceOCIRepositoryArgs.verifySecretRef; secretName != "" {
			repository.Spec.Verify.SecretRef = &meta.LocalObjectReference{
				Name: secretName,
			}
		}
	} else if sourceOCIRepositoryArgs.verifySecretRef != "" {
		return fmt.Errorf("a verification provider must be specified when a secret is specified")
	}

	if createArgs.export {
		return printExport(exportOCIRepository(repository))
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	logger.Actionf("applying OCIRepository")
	namespacedName, err := upsertOCIRepository(ctx, kubeClient, repository)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for OCIRepository reconciliation")
	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true,
		isObjectReadyConditionFunc(kubeClient, namespacedName, repository)); err != nil {
		return err
	}
	logger.Successf("OCIRepository reconciliation completed")

	if repository.Status.Artifact == nil {
		return fmt.Errorf("no artifact was found")
	}
	logger.Successf("fetched revision: %s", repository.Status.Artifact.Revision)
	return nil
}

func upsertOCIRepository(ctx context.Context, kubeClient client.Client,
	ociRepository *sourcev1.OCIRepository) (types.NamespacedName, error) {
	namespacedName := types.NamespacedName{
		Namespace: ociRepository.GetNamespace(),
		Name:      ociRepository.GetName(),
	}

	var existing sourcev1.OCIRepository
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, ociRepository); err != nil {
				return namespacedName, err
			} else {
				logger.Successf("OCIRepository created")
				return namespacedName, nil
			}
		}
		return namespacedName, err
	}

	existing.Labels = ociRepository.Labels
	existing.Spec = ociRepository.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return namespacedName, err
	}
	ociRepository = &existing
	logger.Successf("OCIRepository updated")
	return namespacedName, nil
}
