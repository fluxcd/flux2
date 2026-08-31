//go:build !e2e
// +build !e2e

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

import "testing"

func TestSuspendSelector_Set(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  SuspendSelector
		expectErr bool
	}{
		{name: "any", value: "any", expected: SuspendSelectorAny},
		{name: "suspended", value: "suspended", expected: SuspendSelectorSuspended},
		{name: "active", value: "active", expected: SuspendSelectorActive},
		{name: "empty", value: "", expectErr: true},
		{name: "uppercase", value: "Any", expectErr: true},
		{name: "boolean", value: "true", expectErr: true},
		{name: "whitespace", value: " any ", expectErr: true},
		{name: "unsupported", value: "unknown", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var selector SuspendSelector
			err := selector.Set(tt.value)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Set() error = %v, expectErr %v", err, tt.expectErr)
			}
			if selector != tt.expected {
				t.Fatalf("Set() = %q, expected %q", selector, tt.expected)
			}
		})
	}
}

func TestSuspendSelector_Matches(t *testing.T) {
	tests := []struct {
		name      string
		selector  SuspendSelector
		suspended bool
		expected  bool
	}{
		{name: "any matches active", selector: SuspendSelectorAny, expected: true},
		{name: "any matches suspended", selector: SuspendSelectorAny, suspended: true, expected: true},
		{name: "active matches active", selector: SuspendSelectorActive, expected: true},
		{name: "active rejects suspended", selector: SuspendSelectorActive, suspended: true},
		{name: "suspended rejects active", selector: SuspendSelectorSuspended},
		{name: "suspended matches suspended", selector: SuspendSelectorSuspended, suspended: true, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.selector.Matches(tt.suspended); got != tt.expected {
				t.Fatalf("Matches() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestSuspendSelector_Type(t *testing.T) {
	var selector SuspendSelector
	if got, expected := selector.Type(), "any|suspended|active"; got != expected {
		t.Fatalf("Type() = %q, expected %q", got, expected)
	}
}
