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

package flags

import (
	"fmt"
	"strings"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var supportedSourceOCIVerifyProviders = []string{
	"cosign",
}

type SourceOCIVerifyProvider string

func (p *SourceOCIVerifyProvider) String() string {
	return string(*p)
}

func (p *SourceOCIVerifyProvider) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no source OCI verify provider given, please specify %s",
			p.Description())
	}
	if !utils.ContainsItemString(supportedSourceOCIVerifyProviders, str) {
		return fmt.Errorf("source OCI verify provider '%s' is not supported, must be one of: %v",
			str, strings.Join(supportedSourceOCIVerifyProviders, ", "))
	}
	*p = SourceOCIVerifyProvider(str)
	return nil
}

func (p *SourceOCIVerifyProvider) Type() string {
	return "sourceOCIVerifyProvider"
}

func (p *SourceOCIVerifyProvider) Description() string {
	return fmt.Sprintf(
		"the OCI verify provider name to use for signature verification, available options are: (%s)",
		strings.Join(supportedSourceOCIVerifyProviders, ", "),
	)
}
