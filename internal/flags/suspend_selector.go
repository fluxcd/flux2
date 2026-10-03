/*
Copyright 2026 The Flux authors

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

const (
	SuspendSelectorAny       SuspendSelector = "any"
	SuspendSelectorSuspended SuspendSelector = "suspended"
	SuspendSelectorActive    SuspendSelector = "active"
)

var supportedSuspendSelectors = []string{
	string(SuspendSelectorAny),
	string(SuspendSelectorSuspended),
	string(SuspendSelectorActive),
}

type SuspendSelector string

func (s *SuspendSelector) String() string {
	return string(*s)
}

func (s *SuspendSelector) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no suspend selector given, must be one of: %s",
			strings.Join(supportedSuspendSelectors, ", "))
	}
	if !utils.ContainsItemString(supportedSuspendSelectors, str) {
		return fmt.Errorf("unsupported suspend selector '%s', must be one of: %s",
			str, strings.Join(supportedSuspendSelectors, ", "))
	}
	*s = SuspendSelector(str)
	return nil
}

func (s *SuspendSelector) Type() string {
	return strings.Join(supportedSuspendSelectors, "|")
}

func (s *SuspendSelector) Description() string {
	return "filter objects by suspension state"
}

func (s SuspendSelector) Matches(suspended bool) bool {
	switch s {
	case SuspendSelectorAny:
		return true
	case SuspendSelectorSuspended:
		return suspended
	case SuspendSelectorActive:
		return !suspended
	default:
		return false
	}
}
