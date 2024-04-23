/*
Copyright 2024 The Flux authors

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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	"github.com/notaryproject/notation-go/verifier/trustpolicy"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var createSecretNotationCmd = &cobra.Command{
	Use:   "notation [name]",
	Short: "Create or update a Kubernetes secret for verifications of artifacts signed by Notation",
	Long:  withPreviewNote(`The create secret notation command generates a Kubernetes secret with root ca certificates and trust policy.`),
	Example: ` # Create a Notation configuration secret on disk and encrypt it with Mozilla SOPS
  flux create secret notation my-notation-cert \
    --namespace=my-namespace \
    --trust-policy-file=./my-trust-policy.json \
    --ca-cert-file=./my-cert.crt \
    --export > my-notation-cert.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place my-notation-cert.yaml`,

	RunE: createSecretNotationCmdRun,
}

type secretNotationFlags struct {
	trustPolicyFile string
	caCrtFile       []string
}

var secretNotationArgs secretNotationFlags

func init() {
	createSecretNotationCmd.Flags().StringVar(&secretNotationArgs.trustPolicyFile, "trust-policy-file", "", "notation trust policy file path")
	createSecretNotationCmd.Flags().StringSliceVar(&secretNotationArgs.caCrtFile, "ca-cert-file", []string{}, "root ca cert file path")

	createSecretCmd.AddCommand(createSecretNotationCmd)
}

func createSecretNotationCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("name is required")
	}

	if secretNotationArgs.caCrtFile == nil || len(secretNotationArgs.caCrtFile) == 0 {
		return fmt.Errorf("--ca-cert-file is required")
	}

	if secretNotationArgs.trustPolicyFile == "" {
		return fmt.Errorf("--trust-policy-file is required")
	}

	name := args[0]

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	policy, err := os.ReadFile(secretNotationArgs.trustPolicyFile)
	if err != nil {
		return fmt.Errorf("unable to read trust policy file: %w", err)
	}

	var doc trustpolicy.Document

	if err := json.Unmarshal(policy, &doc); err != nil {
		return fmt.Errorf("failed to unmarshal trust policy %s: %w", secretNotationArgs.trustPolicyFile, err)
	}

	if err := doc.Validate(); err != nil {
		return fmt.Errorf("invalid trust policy: %w", err)
	}

	var (
		caCerts []sourcesecret.VerificationCrt
		fileErr error
	)
	for _, caCrtFile := range secretNotationArgs.caCrtFile {
		fileName := filepath.Base(caCrtFile)
		if !strings.HasSuffix(fileName, ".crt") && !strings.HasSuffix(fileName, ".pem") {
			fileErr = errors.Join(fileErr, fmt.Errorf("%s must end with either .crt or .pem", fileName))
			continue
		}
		caBundle, err := os.ReadFile(caCrtFile)
		if err != nil {
			fileErr = errors.Join(fileErr, fmt.Errorf("unable to read TLS CA file: %w", err))
			continue
		}
		caCerts = append(caCerts, sourcesecret.VerificationCrt{Name: fileName, CACrt: caBundle})
	}

	if fileErr != nil {
		return fileErr
	}

	if len(caCerts) == 0 {
		return fmt.Errorf("no CA certs found")
	}

	opts := sourcesecret.Options{
		Name:             name,
		Namespace:        *kubeconfigArgs.Namespace,
		Labels:           labels,
		VerificationCrts: caCerts,
		TrustPolicy:      policy,
	}
	secret, err := sourcesecret.Generate(opts)
	if err != nil {
		return err
	}

	if createArgs.export {
		rootCmd.Println(secret.Content)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()
	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}
	var s corev1.Secret
	if err := yaml.Unmarshal([]byte(secret.Content), &s); err != nil {
		return err
	}
	if err := upsertSecret(ctx, kubeClient, s); err != nil {
		return err
	}

	logger.Actionf("notation configuration secret '%s' created in '%s' namespace", name, *kubeconfigArgs.Namespace)
	return nil
}
