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

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var supportedHelmChartSourceKinds = []string{sourcev1b2.HelmRepositoryKind, sourcev1.GitRepositoryKind, sourcev1b2.BucketKind}

type HelmChartSource struct {
	Kind      string
	Name      string
	Namespace string
}

func (s *HelmChartSource) String() string {
	if s.Name == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", s.Kind, s.Name)
}

func (s *HelmChartSource) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no helm chart source given, please specify %s",
			s.Description())
	}

	sourceKind, sourceName, sourceNamespace := utils.ParseObjectKindNameNamespace(str)
	if sourceKind == "" || sourceName == "" {
		return fmt.Errorf("invalid helm chart source '%s', must be in format <kind>/<name>", str)
	}
	cleanSourceKind, ok := utils.ContainsEqualFoldItemString(supportedHelmChartSourceKinds, sourceKind)
	if !ok {
		return fmt.Errorf("source kind '%s' is not supported, must be one of: %s",
			sourceKind, strings.Join(supportedHelmChartSourceKinds, ", "))
	}

	s.Kind = cleanSourceKind
	s.Name = sourceName
	s.Namespace = sourceNamespace

	return nil
}

func (s *HelmChartSource) Type() string {
	return "helmChartSource"
}

func (s *HelmChartSource) Description() string {
	return fmt.Sprintf(
		"source that contains the chart in the format '<kind>/<name>.<namespace>', "+
			"where kind must be one of: (%s)",
		strings.Join(supportedHelmChartSourceKinds, ", "),
	)
}
