//go:build unit
// +build unit

package gogit

import (
	"testing"

	"github.com/fluxcd/flux2/internal/bootstrap/git"
)

func TestGetOpenPgpEntity(t *testing.T) {
	tests := []struct {
		name       string
		keyPath    string
		passphrase string
		id         string
		expectErr  bool
	}{
		{
			name:       "no default key id given",
			keyPath:    "testdata/private.key",
			passphrase: "flux",
			id:         "",
			expectErr:  false,
		},
		{
			name:       "key id given",
			keyPath:    "testdata/private.key",
			passphrase: "flux",
			id:         "0619327DBD777415",
			expectErr:  false,
		},
		{
			name:       "wrong key id",
			keyPath:    "testdata/private.key",
			passphrase: "flux",
			id:         "0619327DBD777416",
			expectErr:  true,
		},
		{
			name:       "wrong password",
			keyPath:    "testdata/private.key",
			passphrase: "fluxe",
			id:         "0619327DBD777415",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpgInfo := git.GPGSigningInfo{
				KeyRingPath: tt.keyPath,
				Passphrase:  tt.passphrase,
				KeyID:       tt.id,
			}

			_, err := getOpenPgpEntity(gpgInfo)
			if err != nil && !tt.expectErr {
				t.Errorf("unexpected error: %s", err)
			}
			if err == nil && tt.expectErr {
				t.Errorf("expected error when %s", tt.name)
			}
		})
	}
}
