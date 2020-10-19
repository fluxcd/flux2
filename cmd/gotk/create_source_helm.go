/*
Copyright 2020 The Flux CD contributors.

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
	"io/ioutil"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/fluxcd/toolkit/internal/utils"
)

var createSourceHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Create or update a HelmRepository source",
	Long: `
The create source helm command generates a HelmRepository resource and waits for it to fetch the index.
For private Helm repositories, the basic authentication credentials are stored in a Kubernetes secret.`,
	Example: `  # Create a source from a public Helm repository
  gotk create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --interval=10m

  # Create a source from a Helm repository using basic authentication
  gotk create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --username=username \
    --password=password

  # Create a source from a Helm repository using TLS authentication
  gotk create source helm podinfo \
    --url=https://stefanprodan.github.io/podinfo \
    --cert-file=./cert.crt \
    --key-file=./key.crt \
    --ca-file=./ca.crt
`,
	RunE: createSourceHelmCmdRun,
}

var (
	sourceHelmURL       string
	sourceHelmUsername  string
	sourceHelmPassword  string
	sourceHelmCertFile  string
	sourceHelmKeyFile   string
	sourceHelmCAFile    string
	sourceHelmSecretRef string
)

func init() {
	createSourceHelmCmd.Flags().StringVar(&sourceHelmURL, "url", "", "Helm repository address")
	createSourceHelmCmd.Flags().StringVarP(&sourceHelmUsername, "username", "u", "", "basic authentication username")
	createSourceHelmCmd.Flags().StringVarP(&sourceHelmPassword, "password", "p", "", "basic authentication password")
	createSourceHelmCmd.Flags().StringVar(&sourceHelmCertFile, "cert-file", "", "TLS authentication cert file path")
	createSourceHelmCmd.Flags().StringVar(&sourceHelmKeyFile, "key-file", "", "TLS authentication key file path")
	createSourceHelmCmd.Flags().StringVar(&sourceHelmCAFile, "ca-file", "", "TLS authentication CA file path")
	createSourceHelmCmd.Flags().StringVarP(&sourceHelmSecretRef, "secret-ref", "", "", "the name of an existing secret containing TLS or basic auth credentials")

	createSourceCmd.AddCommand(createSourceHelmCmd)
}

func createSourceHelmCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("HelmRepository source name is required")
	}
	name := args[0]

	if sourceHelmURL == "" {
		return fmt.Errorf("url is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	tmpDir, err := ioutil.TempDir("", name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if _, err := url.Parse(sourceHelmURL); err != nil {
		return fmt.Errorf("url parse failed: %w", err)
	}

	helmRepository := &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    sourceLabels,
		},
		Spec: sourcev1.HelmRepositorySpec{
			URL: sourceHelmURL,
			Interval: metav1.Duration{
				Duration: interval,
			},
		},
	}

	if sourceHelmSecretRef != "" {
		helmRepository.Spec.SecretRef = &corev1.LocalObjectReference{
			Name: sourceHelmSecretRef,
		}
	}

	if export {
		return exportHelmRepository(*helmRepository)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig)
	if err != nil {
		return err
	}

	logger.Generatef("generating HelmRepository source")
	if sourceHelmSecretRef == "" {
		secretName := fmt.Sprintf("helm-%s", name)

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			StringData: map[string]string{},
		}

		if sourceHelmUsername != "" && sourceHelmPassword != "" {
			secret.StringData["username"] = sourceHelmUsername
			secret.StringData["password"] = sourceHelmPassword
		}

		if sourceHelmCertFile != "" && sourceHelmKeyFile != "" {
			cert, err := ioutil.ReadFile(sourceHelmCertFile)
			if err != nil {
				return fmt.Errorf("failed to read repository cert file '%s': %w", sourceHelmCertFile, err)
			}
			secret.StringData["certFile"] = string(cert)

			key, err := ioutil.ReadFile(sourceHelmKeyFile)
			if err != nil {
				return fmt.Errorf("failed to read repository key file '%s': %w", sourceHelmKeyFile, err)
			}
			secret.StringData["keyFile"] = string(key)
		}

		if sourceHelmCAFile != "" {
			ca, err := ioutil.ReadFile(sourceHelmCAFile)
			if err != nil {
				return fmt.Errorf("failed to read repository CA file '%s': %w", sourceHelmCAFile, err)
			}
			secret.StringData["caFile"] = string(ca)
		}

		if len(secret.StringData) > 0 {
			logger.Actionf("applying secret with repository credentials")
			if err := upsertSecret(ctx, kubeClient, secret); err != nil {
				return err
			}
			helmRepository.Spec.SecretRef = &corev1.LocalObjectReference{
				Name: secretName,
			}
			logger.Successf("authentication configured")
		}
	}

	logger.Actionf("applying HelmRepository source")
	namespacedName, err := upsertHelmRepository(ctx, kubeClient, helmRepository)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for HelmRepository source reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
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
