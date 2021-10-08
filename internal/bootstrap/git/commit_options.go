package git

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
	PrivateKeyPath string
	Passphrase     string
	KeyID          string
}

type GpgSigningOption struct {
	*GPGSigningInfo
}

func (w GpgSigningOption) ApplyToCommit(in *CommitOptions) {
	in.GPGSigningInfo = w.GPGSigningInfo
}

func WithGpgSigningOption(path, passphrase, keyID string) Option {
	return GpgSigningOption{
		GPGSigningInfo: &GPGSigningInfo{
			PrivateKeyPath: path,
			Passphrase:     passphrase,
			KeyID:          keyID,
		},
	}
}
