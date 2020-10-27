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

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/fluxcd/toolkit/internal/utils"
)

var supportedHelmChartSourceKinds = []string{sourcev1.HelmRepositoryKind, sourcev1.GitRepositoryKind, sourcev1.BucketKind}

type HelmChartSource struct {
	Kind string
	Name string
}

func (h *HelmChartSource) String() string {
	if h.Name == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", h.Kind, h.Name)
}

func (h *HelmChartSource) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no helm chart source given, please specify %s",
			h.Description())
	}

	sourceKind, sourceName := utils.ParseObjectKindName(str)
	if sourceKind == "" {
		return fmt.Errorf("invalid helm chart source '%s', must be in format <kind>/<name>", str)
	}
	if !utils.ContainsItemString(supportedHelmChartSourceKinds, sourceKind) {
		return fmt.Errorf("source kind '%s' is not supported, can be one of: %s",
			sourceKind, strings.Join(supportedHelmChartSourceKinds, ", "))
	}

	h.Name = sourceName
	h.Kind = sourceKind

	return nil
}

func (h *HelmChartSource) Type() string {
	return "helmChartSource"
}

func (h *HelmChartSource) Description() string {
	return fmt.Sprintf(
		"source that contains the chart in the format '<kind>/<name>',"+
			"where kind can be one of: (%s)",
		strings.Join(supportedHelmChartSourceKinds, ", "),
	)
}
