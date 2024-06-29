package flags

import (
	"testing"
)

func TestGitLabVisibility_Set(t *testing.T) {
	tests := []struct {
		name      string
		str       string
		expect    string
		expectErr bool
	}{
		{"private", "private", "private", false},
		{"internal", "internal", "internal", false},
		{"public", "public", "public", false},
		{"unsupported", "unsupported", "", true},
		{"default", "", "private", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p GitLabVisibility
			if err := p.Set(tt.str); (err != nil) != tt.expectErr {
				t.Errorf("Set() error = %v, expectErr %v", err, tt.expectErr)
			}
			if str := p.String(); str != tt.expect {
				t.Errorf("Set() = %v, expect %v", str, tt.expect)
			}
		})
	}
}
