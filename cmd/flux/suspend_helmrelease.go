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
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/spf13/cobra"
)

var suspendHrCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Suspend reconciliation of HelmRelease",
	Long:    "The suspend command disables the reconciliation of a HelmRelease resource.",
	Example: `  # Suspend reconciliation for an existing Helm release
  flux suspend hr podinfo
`,
	RunE: suspendCommand{
		apiType: helmReleaseType,
		object:  &helmReleaseAdapter{&helmv2.HelmRelease{}},
	}.run,
}

func init() {
	suspendCmd.AddCommand(suspendHrCmd)
}

func (obj helmReleaseAdapter) isSuspended() bool {
	return obj.HelmRelease.Spec.Suspend
}

func (obj helmReleaseAdapter) setSuspended() {
	obj.HelmRelease.Spec.Suspend = true
}
