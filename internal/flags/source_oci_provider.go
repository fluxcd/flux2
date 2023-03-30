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

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/pkg/oci"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var supportedSourceOCIProviders = []string{
	sourcev1.GenericOCIProvider,
	sourcev1.AmazonOCIProvider,
	sourcev1.AzureOCIProvider,
	sourcev1.GoogleOCIProvider,
}

var sourceOCIProvidersToOCIProvider = map[string]oci.Provider{
	sourcev1.GenericOCIProvider: oci.ProviderGeneric,
	sourcev1.AmazonOCIProvider:  oci.ProviderAWS,
	sourcev1.AzureOCIProvider:   oci.ProviderAzure,
	sourcev1.GoogleOCIProvider:  oci.ProviderGCP,
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
	return "sourceOCIProvider"
}

func (p *SourceOCIProvider) Description() string {
	return fmt.Sprintf(
		"the OCI provider name, available options are: (%s)",
		strings.Join(supportedSourceOCIProviders, ", "),
	)
}

func (p *SourceOCIProvider) ToOCIProvider() (oci.Provider, error) {
	value, ok := sourceOCIProvidersToOCIProvider[p.String()]
	if !ok {
		return 0, fmt.Errorf("no mapping between source OCI provider %s and OCI provider", p.String())
	}

	return value, nil
}
