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
	"github.com/fluxcd/pkg/runtime/conditions"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
)

var createSourceHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Create or update a HelmRepository source",
	Long: `The create source helm command generates a HelmRepository resource and waits for it to fetch the index.
For private Helm repositories, the basic authentication credentials are stored in a Kubernetes secret.`,
	Example: `  # Create a source for a public Helm repository
  flux create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --interval=10m

  # Create a source for a Helm repository using basic authentication
  flux create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --username=username \
    --password=password

  # Create a source for a Helm repository using TLS authentication
  flux create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --cert-file=./cert.crt \
    --key-file=./key.crt \
    --ca-file=./ca.crt`,
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
	createSourceHelmCmd.Flags().StringVarP(&sourceHelmArgs.secretRef, "secret-ref", "", "", "the name of an existing secret containing TLS or basic auth credentials")
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

	logger.Generatef("generating HelmRepository source")
	if sourceHelmArgs.secretRef == "" {
		secretName := fmt.Sprintf("helm-%s", name)
		secretOpts := sourcesecret.Options{
			Name:         secretName,
			Namespace:    *kubeconfigArgs.Namespace,
			Username:     sourceHelmArgs.username,
			Password:     sourceHelmArgs.password,
			CertFilePath: sourceHelmArgs.certFile,
			KeyFilePath:  sourceHelmArgs.keyFile,
			CAFilePath:   sourceHelmArgs.caFile,
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
	if err := wait.PollImmediate(rootArgs.pollInterval, rootArgs.timeout,
		isHelmRepositoryReady(ctx, kubeClient, namespacedName, helmRepository)); err != nil {
		return err
	}
	logger.Successf("HelmRepository source reconciliation completed")

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

func isHelmRepositoryReady(ctx context.Context, kubeClient client.Client,
	namespacedName types.NamespacedName, helmRepository *sourcev1.HelmRepository) wait.ConditionFunc {
	return func() (bool, error) {
		err := kubeClient.Get(ctx, namespacedName, helmRepository)
		if err != nil {
			return false, err
		}

		if c := conditions.Get(helmRepository, meta.ReadyCondition); c != nil {
			// Confirm the Ready condition we are observing is for the
			// current generation
			if c.ObservedGeneration != helmRepository.GetGeneration() {
				return false, nil
			}

			// Further check the Status
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
