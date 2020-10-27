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

	"github.com/fluxcd/toolkit/internal/utils"
)

var supportedArchs = []string{"amd64", "arm", "arm64"}

type Arch string

func (a *Arch) String() string {
	return string(*a)
}

func (a *Arch) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no arch given, must be one of: %s",
			strings.Join(supportedArchs, ", "))
	}
	if !utils.ContainsItemString(supportedArchs, str) {
		return fmt.Errorf("unsupported arch '%s', must be one of: %s",
			str, strings.Join(supportedArchs, ", "))

	}
	*a = Arch(str)
	return nil
}

func (a *Arch) Type() string {
	return "arch"
}

func (a *Arch) Description() string {
	return fmt.Sprintf("cluster architecture, available options are: (%s)", strings.Join(supportedArchs, ", "))
}
