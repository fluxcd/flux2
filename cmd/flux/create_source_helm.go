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
	"net/url"
	"os"

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
)

var createSourceHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Create or update a HelmRepository source",
	Long: withPreviewNote(`The create source helm command generates a HelmRepository resource and waits for it to fetch the index.
For private Helm repositories, the basic authentication credentials are stored in a Kubernetes secret.`),
	Example: `  # Create a source for an HTTPS public Helm repository
  flux create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --interval=10m

  # Create a source for an HTTPS Helm repository using basic authentication
  flux create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --username=username \
    --password=password

  # Create a source for an HTTPS Helm repository using TLS authentication
  flux create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --cert-file=./cert.crt \
    --key-file=./key.crt \
    --ca-file=./ca.crt

  # Create a source for an OCI Helm repository
  flux create source helm podinfo \
    --url=oci://ghcr.io/stefanprodan/charts/podinfo \
    --username=username \
    --password=password

  # Create a source for an OCI Helm repository using an existing secret with basic auth or dockerconfig credentials
  flux create source helm podinfo \
    --url=oci://ghcr.io/stefanprodan/charts/podinfo \
    --secret-ref=docker-config`,
	RunE: createSourceHelmCmdRun,
}

type sourceHelmFlags struct {
	url             string
	username        string
	password        string
	certFile        string
	keyFile         string
	caFile          string
	secretRef       string
	ociProvider     string
	passCredentials bool
}

var sourceHelmArgs sourceHelmFlags

func init() {
	createSourceHelmCmd.Flags().StringVar(&sourceHelmArgs.url, "url", "", "Helm repository address")
	createSourceHelmCmd.Flags().StringVarP(&sourceHelmArgs.username, "username", "u", "", "basic authentication username")
	createSourceHelmCmd.Flags().StringVarP(&sourceHelmArgs.password, "password", "p", "", "basic authentication password")
	createSourceHelmCmd.Flags().StringVar(&sourceHelmArgs.certFile, "cert-file", "", "TLS authentication cert file path")
	createSourceHelmCmd.Flags().StringVar(&sourceHelmArgs.keyFile, "key-file", "", "TLS authentication key file path")
	createSourceHelmCmd.Flags().StringVar(&sourceHelmArgs.caFile, "ca-file", "", "TLS authentication CA file path")
	createSourceHelmCmd.Flags().StringVarP(&sourceHelmArgs.secretRef, "secret-ref", "", "", "the name of an existing secret containing TLS, basic auth or docker-config credentials")
	createSourceHelmCmd.Flags().StringVar(&sourceHelmArgs.ociProvider, "oci-provider", "", "OCI provider for authentication")
	createSourceHelmCmd.Flags().BoolVarP(&sourceHelmArgs.passCredentials, "pass-credentials", "", false, "pass credentials to all domains")

	createSourceCmd.AddCommand(createSourceHelmCmd)
}

func createSourceHelmCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if sourceHelmArgs.url == "" {
		return fmt.Errorf("url is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if _, err := url.Parse(sourceHelmArgs.url); err != nil {
		return fmt.Errorf("url parse failed: %w", err)
	}

	helmRepository := &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *kubeconfigArgs.Namespace,
			Labels:    sourceLabels,
		},
		Spec: sourcev1.HelmRepositorySpec{
			URL: sourceHelmArgs.url,
			Interval: metav1.Duration{
				Duration: createArgs.interval,
			},
		},
	}

	url, err := url.Parse(sourceHelmArgs.url)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	if url.Scheme == sourcev1.HelmRepositoryTypeOCI {
		helmRepository.Spec.Type = sourcev1.HelmRepositoryTypeOCI
		helmRepository.Spec.Provider = sourceHelmArgs.ociProvider
	}

	if createSourceArgs.fetchTimeout > 0 {
		helmRepository.Spec.Timeout = &metav1.Duration{Duration: createSourceArgs.fetchTimeout}
	}

	if sourceHelmArgs.secretRef != "" {
		helmRepository.Spec.SecretRef = &meta.LocalObjectReference{
			Name: sourceHelmArgs.secretRef,
		}
		helmRepository.Spec.PassCredentials = sourceHelmArgs.passCredentials
	}

	if createArgs.export {
		return printExport(exportHelmRepository(helmRepository))
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	caBundle := []byte{}
	if sourceHelmArgs.caFile != "" {
		var err error
		caBundle, err = os.ReadFile(sourceHelmArgs.caFile)
		if err != nil {
			return fmt.Errorf("unable to read TLS CA file: %w", err)
		}
	}

	var certFile, keyFile []byte
	if sourceHelmArgs.certFile != "" && sourceHelmArgs.keyFile != "" {
		if certFile, err = os.ReadFile(sourceHelmArgs.certFile); err != nil {
			return fmt.Errorf("failed to read cert file: %w", err)
		}
		if keyFile, err = os.ReadFile(sourceHelmArgs.keyFile); err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}
	}

	logger.Generatef("generating HelmRepository source")
	if sourceHelmArgs.secretRef == "" {
		secretName := fmt.Sprintf("helm-%s", name)
		secretOpts := sourcesecret.Options{
			Name:         secretName,
			Namespace:    *kubeconfigArgs.Namespace,
			Username:     sourceHelmArgs.username,
			Password:     sourceHelmArgs.password,
			CAFile:       caBundle,
			CertFile:     certFile,
			KeyFile:      keyFile,
			ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
		}
		secret, err := sourcesecret.Generate(secretOpts)
		if err != nil {
			return err
		}
		var s corev1.Secret
		if err = yaml.Unmarshal([]byte(secret.Content), &s); err != nil {
			return err
		}
		if len(s.StringData) > 0 {
			logger.Actionf("applying secret with repository credentials")
			if err := upsertSecret(ctx, kubeClient, s); err != nil {
				return err
			}
			helmRepository.Spec.SecretRef = &meta.LocalObjectReference{
				Name: secretName,
			}
			helmRepository.Spec.PassCredentials = sourceHelmArgs.passCredentials
			logger.Successf("authentication configured")
		}
	}

	logger.Actionf("applying HelmRepository source")
	namespacedName, err := upsertHelmRepository(ctx, kubeClient, helmRepository)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for HelmRepository source reconciliation")
	readyConditionFunc := isObjectReadyConditionFunc(kubeClient, namespacedName, helmRepository)
	if helmRepository.Spec.Type == sourcev1.HelmRepositoryTypeOCI {
		// HelmRepository type OCI is a static object.
		readyConditionFunc = isStaticObjectReadyConditionFunc(kubeClient, namespacedName, helmRepository)
	}
	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true, readyConditionFunc); err != nil {
		return err
	}
	logger.Successf("HelmRepository source reconciliation completed")

	if helmRepository.Spec.Type == sourcev1.HelmRepositoryTypeOCI {
		// OCI repos don't expose any artifact so we just return early here
		return nil
	}

	if helmRepository.Status.Artifact == nil {
		return fmt.Errorf("HelmRepository source reconciliation completed but no artifact was found")
	}
	logger.Successf("fetched revision: %s", helmRepository.Status.Artifact.Revision)
	return nil
}

func upsertHelmRepository(ctx context.Context, kubeClient client.Client,
	helmRepository *sourcev1.HelmRepository) (types.NamespacedName, error) {
	namespacedName := types.NamespacedName{
		Namespace: helmRepository.GetNamespace(),
		Name:      helmRepository.GetName(),
	}

	var existing sourcev1.HelmRepository
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, helmRepository); err != nil {
				return namespacedName, err
			} else {
				logger.Successf("source created")
				return namespacedName, nil
			}
		}
		return namespacedName, err
	}

	existing.Labels = helmRepository.Labels
	existing.Spec = helmRepository.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return namespacedName, err
	}
	helmRepository = &existing
	logger.Successf("source updated")
	return namespacedName, nil
}
