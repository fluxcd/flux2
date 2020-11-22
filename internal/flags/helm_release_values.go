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

package flags

import (
	"fmt"
	"strings"

	"github.com/fluxcd/flux2/internal/utils"
)

var supportedHelmReleaseValuesFromKinds = []string{"Secret", "ConfigMap"}

type HelmReleaseValuesFrom struct {
	Kind string
	Name string
}

func (h *HelmReleaseValuesFrom) String() string {
	if h.Name == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", h.Kind, h.Name)
}

func (h *HelmReleaseValuesFrom) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no values given, please specify %s",
			h.Description())
	}

	sourceKind, sourceName := utils.ParseObjectKindName(str)
	if sourceKind == "" {
		return fmt.Errorf("invalid Kubernetes object reference '%s', must be in format <kind>/<name>", str)
	}
	if !utils.ContainsItemString(supportedHelmReleaseValuesFromKinds, sourceKind) {
		return fmt.Errorf("reference kind '%s' is not supported, can be one of: %s",
			sourceKind, strings.Join(supportedHelmReleaseValuesFromKinds, ", "))
	}

	h.Name = sourceName
	h.Kind = sourceKind

	return nil
}

func (h *HelmReleaseValuesFrom) Type() string {
	return "helmReleaseValuesFrom"
}

func (h *HelmReleaseValuesFrom) Description() string {
	return fmt.Sprintf(
		"Kubernetes object reference that contains the values.yaml data key in the format '<kind>/<name>',"+
			"where kind can be one of: (%s)",
		strings.Join(supportedHelmReleaseValuesFromKinds, ", "),
	)
}
