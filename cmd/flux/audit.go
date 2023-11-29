/*
Copyright 2023 The Flux authors

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

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
)

var ctrlChecks = map[string]map[string]bool{
	"helm-controller": {
		"insecure-kubeconfig-exec": false,
		"insecure-kubeconfig-tls":  false,
	},
	"kustomize-controller": {
		"insecure-kubeconfig-exec": false,
		"insecure-kubeconfig-tls":  false,
		"no-remote-bases":          true,
	},
}

var multiTenancyCtrlChecks = map[string]map[string]bool{
	"helm-controller": {
		"no-cross-namespace-refs": true,
	},
	"kustomize-controller": {
		"no-cross-namespace-refs": true,
	},
	"notification-controller": {
		"no-cross-namespace-refs": true,
	},
	"image-reflector-controller": {
		"no-cross-namespace-refs": true,
	},
	"image-automation-controller": {
		"no-cross-namespace-refs": true,
	},
}

var multiTenancyFlag bool

var auditCmd = &cobra.Command{
	Use:     "audit",
	Short:   "Audit the Flux installation for security best practices",
	Long:    withPreviewNote("TBD"),
	Example: `  TBD`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
		defer cancel()

		kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
		if err != nil {
			return err
		}

		logger.Actionf("Starting audit")

		for ctrl, checks := range ctrlChecks {
			if err := auditController(ctx, kubeClient, ctrl, checks); err != nil {
				return fmt.Errorf("failed auditing %s: %w", ctrl, err)
			}
		}

		if err := auditSecretDecryption(ctx, kubeClient); err != nil {
			return fmt.Errorf("failed auditing Secret decryption: %w", err)
		}

		if multiTenancyFlag {
			logger.Actionf("Multi-tenancy lock-down")
			for ctrl, checks := range multiTenancyCtrlChecks {
				if err := auditController(ctx, kubeClient, ctrl, checks); err != nil {
					return fmt.Errorf("failed auditing %s for multi-tenancy lock-down: %w", ctrl, err)
				}
			}
		}

		return nil
	},
}

func auditSecretDecryption(ctx context.Context, c client.Client) error {
	var ksl kustomizev1.KustomizationList
	if err := c.List(ctx, &ksl); err != nil {
		return fmt.Errorf("failed to retrieve Kustomizations: %w", err)
	}

	success := true
	for _, ks := range ksl.Items {
		if ks.Status.Inventory == nil {
			continue
		}
		if ks.Spec.Decryption != nil {
			continue
		}
		for _, e := range ks.Status.Inventory.Entries {
			parts := strings.Split(e.ID, "_")
			if parts[2] == "" && parts[3] == "Secret" {
				success = false
				logger.Warningf("%s/%s doesn't have Secret decryption configured", ks.Namespace, ks.Name)
			}
		}
	}

	if success {
		logger.Successf("Secret decryption is configured for all Kustomizations that create Secrets")
	}

	return nil
}

func auditController(ctx context.Context, c client.Client, name string, flags map[string]bool) error {
	hcDeploys, err := getManagerArgs(ctx, c, name)
	if err != nil {
		return fmt.Errorf("failed to get %s flags: %w", name, err)
	}

	if len(hcDeploys) == 0 {
		logger.Warningf("No %s Deployment found, auditing skipped", name)
	} else {
		for name, args := range hcDeploys {
			for flag, desired := range flags {
				hcExec, err := assertBoolFlagValue(args, flag, desired)
				if err != nil {
					return fmt.Errorf("failed parsing %q args: %w", name, err)
				}
				if hcExec == desired {
					logger.Successf("%s: %s is %t", name, flag, desired)
				} else {
					logger.Warningf("%s: %s should be %t", name, flag, desired)
				}
			}
		}
	}
	return nil
}

func getManagerArgs(ctx context.Context, c client.Client, component string) (map[string][]string, error) {
	var deploys v1.DeploymentList
	if err := c.List(ctx, &deploys, client.MatchingLabels{
		"app.kubernetes.io/component": component,
	}); err != nil {
		return nil, fmt.Errorf("failed to retrieve %s deployments: %w", component, err)
	}

	res := make(map[string][]string, 0)

	for _, deploy := range deploys.Items {
		for _, ctr := range deploy.Spec.Template.Spec.Containers {
			if ctr.Name == "manager" {
				res[deploy.Name] = ctr.Args
			}
		}
	}

	return res, nil
}

func assertBoolFlagValue(args []string, flagName string, value bool) (bool, error) {
	fs := pflag.NewFlagSet("tmp", pflag.ContinueOnError)
	fs.ParseErrorsWhitelist.UnknownFlags = true
	f := fs.BoolP(flagName, "", false, "")
	if err := fs.Parse(args); err != nil {
		return false, fmt.Errorf("failed parsing args: %w", err)
	}
	return *f, nil
}

func init() {
	auditCmd.Flags().BoolVar(&multiTenancyFlag, "multi-tenancy", false, "Enable additional audit checks for multi-tenant clusters.")
	rootCmd.AddCommand(auditCmd)
}
