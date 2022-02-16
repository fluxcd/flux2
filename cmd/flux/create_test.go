package main

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/rand"
)

func Test_validateObjectName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{
			name:  "flux-system",
			valid: true,
		},
		{
			name:  "-flux-system",
			valid: false,
		},
		{
			name:  "-flux-system-",
			valid: false,
		},
		{
			name:  "third.first",
			valid: false,
		},
		{
			name:  "THirdfirst",
			valid: false,
		},
		{
			name:  "THirdfirst",
			valid: false,
		},
		{
			name:  rand.String(63),
			valid: true,
		},
		{
			name:  rand.String(64),
			valid: false,
		},
	}

	for _, tt := range tests {
		valid := validateObjectName(tt.name)
		if valid != tt.valid {
			t.Errorf("expected name %q to return %t for validateObjectName func but got %t",
				tt.name, tt.valid, valid)
		}
	}
}
