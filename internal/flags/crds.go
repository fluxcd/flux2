/*
Copyright 2021 The Flux authors

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

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var supportedCRDsPolicies = []string{
	string(helmv2.Skip),
	string(helmv2.Create),
	string(helmv2.CreateReplace),
}

type CRDsPolicy string

func (a *CRDsPolicy) String() string {
	return string(*a)
}

func (a *CRDsPolicy) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no upgrade CRDs policy given, must be one of: %s",
			strings.Join(supportedCRDsPolicies, ", "))
	}
	if !utils.ContainsItemString(supportedCRDsPolicies, str) {
		return fmt.Errorf("unsupported upgrade CRDs policy '%s', must be one of: %s",
			str, strings.Join(supportedCRDsPolicies, ", "))

	}
	*a = CRDsPolicy(str)
	return nil
}

func (a *CRDsPolicy) Type() string {
	return "crds"
}

func (a *CRDsPolicy) Description() string {
	return fmt.Sprintf("upgrade CRDs policy, available options are: (%s)", strings.Join(supportedCRDsPolicies, ", "))
}
