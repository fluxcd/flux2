/*
Copyright 2022 The Flux authors

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

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var supportedSourceOCIProviders = []string{
	sourcev1.GenericOCIProvider,
	sourcev1.AmazonOCIProvider,
	sourcev1.AzureOCIProvider,
	sourcev1.GoogleOCIProvider,
}

type SourceOCIProvider string

func (p *SourceOCIProvider) String() string {
	return string(*p)
}

func (p *SourceOCIProvider) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no source OCI provider given, please specify %s",
			p.Description())
	}
	if !utils.ContainsItemString(supportedSourceOCIProviders, str) {
		return fmt.Errorf("source OCI provider '%s' is not supported, must be one of: %v",
			str, strings.Join(supportedSourceOCIProviders, ", "))
	}
	*p = SourceOCIProvider(str)
	return nil
}

func (p *SourceOCIProvider) Type() string {
	return strings.Join(supportedSourceOCIProviders, "|")
}

func (p *SourceOCIProvider) Description() string {
	return "the OCI provider name"
}
