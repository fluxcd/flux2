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

	"github.com/manifoldco/promptui"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	"github.com/fluxcd/flux2/v2/pkg/manifestgen"
)

// bootstrapLabels are labels put on a resource by kustomize-controller. These labels on the CRD indicates
// that flux has been bootstrapped.
var bootstrapLabels = []string{
	fmt.Sprintf("%s/name", kustomizev1.GroupVersion.Group),
	fmt.Sprintf("%s/namespace", kustomizev1.GroupVersion.Group),
}

// fluxClusterInfo contains information about an existing flux installation on a cluster.
type fluxClusterInfo struct {
	// bootstrapped indicates that Flux was installed using the `flux bootstrap` command.
	bootstrapped bool
	// managedBy is the name of the tool being used to manage the installation of Flux.
	managedBy string
	// partOf indicates which distribution the instance is a part of.
	partOf string
	// version is the Flux version number in semver format.
	version string
}

// getFluxClusterInfo returns information on the Flux installation running on the cluster.
// If an error occurred, the returned error will be non-nil.
//
// This function retrieves the GitRepository CRD from the cluster and checks it
// for a set of labels used to determine the Flux version and how Flux was installed.
// It returns the NotFound error from the underlying library if it was unable to find
// the GitRepository CRD and this can be used to check if Flux is installed.
func getFluxClusterInfo(ctx context.Context, c client.Client) (fluxClusterInfo, error) {
	var info fluxClusterInfo
	crdMetadata := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiextensionsv1.SchemeGroupVersion.String(),
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("gitrepositories.%s", sourcev1.GroupVersion.Group),
		},
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(crdMetadata), crdMetadata); err != nil {
		return info, err
	}

	info.version = crdMetadata.Labels[manifestgen.VersionLabelKey]

	var present bool
	for _, l := range bootstrapLabels {
		_, present = crdMetadata.Labels[l]
	}
	if present {
		info.bootstrapped = true
	}

	// the `app.kubernetes.io/managed-by` label is not set by flux but might be set by other
	// tools used to install Flux e.g Helm.
	if manager, ok := crdMetadata.Labels["app.kubernetes.io/managed-by"]; ok {
		info.managedBy = manager
	}

	if partOf, ok := crdMetadata.Labels[manifestgen.PartOfLabelKey]; ok {
		info.partOf = partOf
	}
	return info, nil
}

// confirmFluxInstallOverride displays a prompt to the user so that they can confirm before overriding
// a Flux installation. It returns nil if the installation should continue,
// promptui.ErrAbort if the user doesn't confirm, or an error encountered.
func confirmFluxInstallOverride(info fluxClusterInfo) error {
	// no need to display prompt if installation is managed by Flux
	if installManagedByFlux(info.managedBy) {
		return nil
	}

	display := fmt.Sprintf("Flux %s has been installed on this cluster with %s!", info.version, info.managedBy)
	fmt.Fprintln(rootCmd.ErrOrStderr(), display)
	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("Are you sure you want to override the %s installation? Y/N", info.managedBy),
		IsConfirm: true,
	}
	_, err := prompt.Run()
	return err
}

func (info fluxClusterInfo) distribution() string {
	distribution := info.version
	if info.partOf != "" {
		distribution = fmt.Sprintf("%s-%s", info.partOf, info.version)
	}
	return distribution
}

func installManagedByFlux(manager string) bool {
	return manager == "" || manager == "flux"
}
