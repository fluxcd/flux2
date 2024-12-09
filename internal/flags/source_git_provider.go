/*
Copyright 2024 The Flux authors

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
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

var supportedSourceGitProviders = []string{
	sourcev1.GitProviderGeneric,
	sourcev1.GitProviderAzure,
	sourcev1.GitProviderGitHub,
}

type SourceGitProvider string

func (p *SourceGitProvider) String() string {
	return string(*p)
}

func (p *SourceGitProvider) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no source Git provider given, please specify %s",
			p.Description())
	}
	if !utils.ContainsItemString(supportedSourceGitProviders, str) {
		return fmt.Errorf("source Git provider '%s' is not supported, must be one of: %v",
			str, p.Type())
	}
	*p = SourceGitProvider(str)
	return nil
}

func (p *SourceGitProvider) Type() string {
	return strings.Join(supportedSourceGitProviders, "|")
}

func (p *SourceGitProvider) Description() string {
	return "the Git provider name"
}
