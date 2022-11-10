package git

import (
	"github.com/ProtonMail/go-crypto/openpgp"
)

// Option is a some configuration that modifies options for a commit.
type Option interface {
	// ApplyToCommit applies this configuration to a given commit option.
	ApplyToCommit(*CommitOptions)
}

// CommitOptions contains options for making a commit.
type CommitOptions struct {
	*GPGSigningInfo
}

// GPGSigningInfo contains information for signing a commit.
type GPGSigningInfo struct {
	KeyRing    openpgp.EntityList
	Passphrase string
	KeyID      string
}

type GpgSigningOption struct {
	*GPGSigningInfo
}

func (w GpgSigningOption) ApplyToCommit(in *CommitOptions) {
	in.GPGSigningInfo = w.GPGSigningInfo
}

func WithGpgSigningOption(keyRing openpgp.EntityList, passphrase, keyID string) Option {
	// Return nil if no path is set, even if other options are configured.
	if len(keyRing) == 0 {
		return GpgSigningOption{}
	}

	return GpgSigningOption{
		GPGSigningInfo: &GPGSigningInfo{
			KeyRing:    keyRing,
			Passphrase: passphrase,
			KeyID:      keyID,
		},
	}
}
