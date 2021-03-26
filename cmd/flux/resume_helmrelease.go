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
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/spf13/cobra"
)

var resumeHrCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Resume a suspended HelmRelease",
	Long: `The resume command marks a previously suspended HelmRelease resource for reconciliation and waits for it to
finish the apply.`,
	Example: `  # Resume reconciliation for an existing Helm release
  flux resume hr podinfo`,
	RunE: resumeCommand{
		apiType: helmReleaseType,
		object:  helmReleaseAdapter{&helmv2.HelmRelease{}},
	}.run,
}

func init() {
	resumeCmd.AddCommand(resumeHrCmd)
}

func (obj helmReleaseAdapter) getObservedGeneration() int64 {
	return obj.HelmRelease.Status.ObservedGeneration
}

func (obj helmReleaseAdapter) setUnsuspended() {
	obj.HelmRelease.Spec.Suspend = false
}

func (obj helmReleaseAdapter) successMessage() string {
	return fmt.Sprintf("applied revision %s", obj.Status.LastAppliedRevision)
}
