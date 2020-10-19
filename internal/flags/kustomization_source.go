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

package flags

import (
	"fmt"
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/fluxcd/toolkit/internal/utils"
)

var supportedKustomizationSourceKinds = []string{sourcev1.GitRepositoryKind, sourcev1.BucketKind}

type KustomizationSource struct {
	Kind string
	Name string
}

func (k *KustomizationSource) String() string {
	if k.Name == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", k.Kind, k.Name)
}

func (k *KustomizationSource) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no kustomization source given, please specify %s",
			k.Description())
	}

	sourceKind, sourceName := utils.ParseObjectKindName(str)
	if sourceKind == "" {
		sourceKind = sourcev1.GitRepositoryKind
	}
	if !utils.ContainsItemString(supportedKustomizationSourceKinds, sourceKind) {
		return fmt.Errorf("source kind '%s' is not supported, can be one of: %s",
			sourceKind, strings.Join(supportedKustomizationSourceKinds, ", "))
	}

	k.Name = sourceName
	k.Kind = sourceKind

	return nil
}

func (k *KustomizationSource) Type() string {
	return "kustomizationSource"
}

func (k *KustomizationSource) Description() string {
	return fmt.Sprintf(
		"source that contains the Kubernetes manifests in the format '[<kind>/]<name>',"+
			"where kind can be one of: (%s), if kind is not specified it defaults to GitRepository",
		strings.Join(supportedKustomizationSourceKinds, ", "),
	)
}
