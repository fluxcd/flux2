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

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
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
	// version is the Flux version number in semver format.
	version string
}

// getFluxClusterInfo returns information on the Flux installation running on the cluster.
// If the information cannot be retrieved, the boolean return value will be false.
// If an error occurred, the returned error will be non-nil.
//
// This function retrieves the GitRepository CRD from the cluster and checks it
// for a set of labels used to determine the Flux version and how Flux was installed.
func getFluxClusterInfo(ctx context.Context, c client.Client) (fluxClusterInfo, bool, error) {
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
		if errors.IsNotFound(err) {
			return info, false, nil
		}
		return info, false, err
	}

	info.version = crdMetadata.Labels["app.kubernetes.io/version"]

	var present bool
	for _, l := range bootstrapLabels {
		_, present = crdMetadata.Labels[l]
	}
	if present {
		info.bootstrapped = true
	}

	if manager, ok := crdMetadata.Labels["app.kubernetes.io/managed-by"]; ok {
		info.managedBy = manager
	}
	return info, true, nil
}
