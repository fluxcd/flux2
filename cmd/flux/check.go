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
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/pkg/version"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/v2/pkg/status"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Args:  cobra.NoArgs,
	Short: "Check requirements and installation",
	Long: withPreviewNote(`The check command will perform a series of checks to validate that
the local environment is configured correctly and if the installed components are healthy.`),
	Example: `  # Run pre-installation checks
  flux check --pre

  # Run installation checks
  flux check`,
	RunE: runCheckCmd,
}

type checkFlags struct {
	pre             bool
	components      []string
	extraComponents []string
	pollInterval    time.Duration
}

type checkResult struct {
	Title   string
	Entries []checkEntry
}

type checkEntry struct {
	Name   string
	Failed bool
}

func (cr *checkResult) Failed() bool {
	if cr == nil {
		return false
	}
	for _, entry := range cr.Entries {
		if entry.Failed {
			return true
		}
	}
	return false
}

var kubernetesConstraints = []string{
	">=1.30.0-0",
}

var checkArgs checkFlags

func init() {
	checkCmd.Flags().BoolVarP(&checkArgs.pre, "pre", "", false,
		"only run pre-installation checks")
	checkCmd.Flags().StringSliceVar(&checkArgs.components, "components", rootArgs.defaults.Components,
		"list of components, accepts comma-separated values")
	checkCmd.Flags().StringSliceVar(&checkArgs.extraComponents, "components-extra", nil,
		"list of components in addition to those supplied or defaulted, accepts comma-separated values")
	checkCmd.Flags().DurationVar(&checkArgs.pollInterval, "poll-interval", 5*time.Second,
		"how often the health checker should poll the cluster for the latest state of the resources.")
	rootCmd.AddCommand(checkCmd)
}

func runCheckCmd(_ *cobra.Command, _ []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	cfg, err := utils.KubeConfig(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return fmt.Errorf("kubernetes client initialization failed: %w", err)
	}

	kubeClient, err := client.New(cfg, client.Options{Scheme: utils.NewScheme()})
	if err != nil {
		return fmt.Errorf("error creating kubernetes client: %w", err)
	}

	if !runPreChecks(ctx, cfg) {
		return errors.New("pre-requisites checks failed")
	}

	if checkArgs.pre {
		logger.Actionf("All pre-requisites checks passed")
		return nil
	}

	if !runChecks(ctx, kubeClient) {
		return errors.New("checks failed")
	}

	logger.Actionf("All checks passed")
	return nil
}

func runPreChecks(_ context.Context, kConfig *rest.Config) bool {
	checks := []func() checkResult{
		fluxCheck,
		func() checkResult {
			return kubernetesCheck(kConfig, kubernetesConstraints)
		},
	}

	return runGenericChecks(checks)
}

func runChecks(ctx context.Context, kClient client.Client) bool {
	checks := []func() checkResult{
		func() checkResult { return fluxClusterVersionCheck(ctx, kClient) },
		func() checkResult { return controllersCheck(ctx, kClient) },
		func() checkResult { return crdsCheck(ctx, kClient) },
	}

	return runGenericChecks(checks)
}

func runGenericChecks(checks []func() checkResult) bool {
	success := true
	for _, check := range checks {
		result := check()
		logCheckResult(result)
		if result.Failed() {
			success = false
		}
	}
	return success
}

func logCheckResult(res checkResult) {
	logger.Actionf("Checking %s", res.Title)
	for _, entry := range res.Entries {
		if entry.Failed {
			logger.Failuref("%s", entry.Name)
		} else {
			logger.Successf("%s", entry.Name)
		}
	}
}

func fluxCheck() checkResult {
	res := checkResult{Title: "flux pre-requisites"}

	curSv, err := version.ParseVersion(VERSION)
	if err != nil {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("error parsing current version: %s", err.Error()),
			Failed: true,
		})
		return res
	}

	// Exclude development builds.
	if curSv.Prerelease() != "" {
		res.Entries = append(res.Entries, checkEntry{
			Name: fmt.Sprintf("flux %s is a development build", curSv),
		})
		return res
	}

	latest, err := install.GetLatestVersion()
	if err != nil {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("error getting latest version: %s", err.Error()),
			Failed: true,
		})
		return res
	}

	latestSv, err := version.ParseVersion(latest)
	if err != nil {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("error parsing latest version: %s", err.Error()),
			Failed: true,
		})
		return res
	}

	if latestSv.GreaterThan(curSv) {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("flux %s <%s (new CLI version is available, please upgrade)", curSv, latestSv),
			Failed: true,
		})
	} else {
		res.Entries = append(res.Entries, checkEntry{
			Name: fmt.Sprintf("flux %s >=%s (latest CLI version)", curSv, latestSv),
		})
	}

	return res
}

func kubernetesCheck(cfg *rest.Config, constraints []string) checkResult {
	res := checkResult{Title: "kubernetes pre-requisites"}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("error creating kubernetes client: %s", err.Error()),
			Failed: true,
		})
		return res
	}

	kv, err := clientSet.Discovery().ServerVersion()
	if err != nil {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("error getting kubernetes version: %s", err.Error()),
			Failed: true,
		})
		return res
	}

	v, err := version.ParseVersion(kv.String())
	if err != nil {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("error parsing kubernetes version: %s", err.Error()),
			Failed: true,
		})
		return res
	}

	for _, constraint := range constraints {
		c, _ := semver.NewConstraint(constraint)
		entry := checkEntry{
			Name: fmt.Sprintf("kubernetes %s%s", v.String(), constraint),
		}
		if !c.Check(v) {
			entry.Failed = true
		}
		res.Entries = append(res.Entries, entry)
	}

	return res
}

func controllersCheck(ctx context.Context, kubeClient client.Client) checkResult {
	res := checkResult{Title: "flux controllers"}

	statusChecker, err := status.NewStatusCheckerWithClient(kubeClient, checkArgs.pollInterval, rootArgs.timeout, logger)
	if err != nil {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("error creating status checker: %s", err.Error()),
			Failed: true,
		})
		return res
	}

	selector := client.MatchingLabels{manifestgen.PartOfLabelKey: manifestgen.PartOfLabelValue}
	var list v1.DeploymentList
	ns := *kubeconfigArgs.Namespace

	if err := kubeClient.List(ctx, &list, client.InNamespace(ns), selector); err != nil {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("error listing deployments: %s", err.Error()),
			Failed: true,
		})
		return res
	}

	if len(list.Items) == 0 {
		res.Entries = append(res.Entries, checkEntry{
			Name: fmt.Sprintf("no controllers found in the '%s' namespace with the label selector '%s=%s'",
				ns, manifestgen.PartOfLabelKey, manifestgen.PartOfLabelValue),
			Failed: true,
		})
		return res
	}

	for _, d := range list.Items {
		ref, err := buildComponentObjectRefs(d.Name)
		if err != nil {
			res.Entries = append(res.Entries, checkEntry{
				Name:   fmt.Sprintf("error building component object refs for %s: %s", d.Name, err.Error()),
				Failed: true,
			})
			continue
		}

		if err := statusChecker.Assess(ref...); err != nil {
			res.Entries = append(res.Entries, checkEntry{
				Name:   fmt.Sprintf("error checking status of %s: %s", d.Name, err.Error()),
				Failed: true,
			})
		} else {
			for _, c := range d.Spec.Template.Spec.Containers {
				res.Entries = append(res.Entries, checkEntry{
					Name: fmt.Sprintf(c.Image),
				})
			}
		}
	}

	return res
}

func crdsCheck(ctx context.Context, kubeClient client.Client) checkResult {
	res := checkResult{Title: "flux crds"}

	selector := client.MatchingLabels{manifestgen.PartOfLabelKey: manifestgen.PartOfLabelValue}
	var list apiextensionsv1.CustomResourceDefinitionList

	if err := kubeClient.List(ctx, &list, client.InNamespace(*kubeconfigArgs.Namespace), selector); err != nil {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("error listing CRDs: %s", err.Error()),
			Failed: true,
		})
		return res
	}

	if len(list.Items) == 0 {
		res.Entries = append(res.Entries, checkEntry{
			Name: fmt.Sprintf("no crds found with the label selector '%s=%s'",
				manifestgen.PartOfLabelKey, manifestgen.PartOfLabelValue),
			Failed: true,
		})
		return res
	}

	for _, crd := range list.Items {
		versions := crd.Status.StoredVersions
		if len(versions) > 0 {
			res.Entries = append(res.Entries, checkEntry{
				Name: fmt.Sprintf("%s/%s", crd.Name, versions[len(versions)-1]),
			})
		} else {
			res.Entries = append(res.Entries, checkEntry{
				Name:   fmt.Sprintf("no stored versions for %s", crd.Name),
				Failed: true,
			})
		}
	}

	return res
}

func fluxClusterVersionCheck(ctx context.Context, kubeClient client.Client) checkResult {
	res := checkResult{Title: "flux installation status"}

	clusterInfo, err := getFluxClusterInfo(ctx, kubeClient)
	if err != nil {
		res.Entries = append(res.Entries, checkEntry{
			Name:   fmt.Sprintf("error getting cluster info: %s", err.Error()),
			Failed: true,
		})
		return res
	}

	if clusterInfo.distribution() != "" {
		res.Entries = append(res.Entries, checkEntry{
			Name: fmt.Sprintf("distribution: %s", clusterInfo.distribution()),
		})
	}

	res.Entries = append(res.Entries, checkEntry{
		Name: fmt.Sprintf("bootstrapped: %t", clusterInfo.bootstrapped),
	})

	return res
}
