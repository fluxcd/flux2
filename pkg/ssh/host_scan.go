package ssh

import (
	"encoding/base64"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ScanHostKey collects the given host's preferred public key for the
// algorithm of the given key pair. Any errors (e.g. authentication
// failures) are ignored, except if no key could be collected from the
// host.
func ScanHostKey(host string, user string, pair *KeyPair) ([]byte, error) {
	signer, err := ssh.ParsePrivateKey(pair.PrivateKey)
	if err != nil {
		return nil, err
	}
	col := &collector{}
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: col.StoreKey(),
	}
	client, err := ssh.Dial("tcp", host, config)
	if err == nil {
		defer client.Close()
	}
	if len(col.knownKeys) > 0 {
		return col.knownKeys, nil
	}
	return col.knownKeys, err
}

type collector struct {
	knownKeys []byte
}

func (c *collector) StoreKey() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		c.knownKeys = append(
			c.knownKeys,
			fmt.Sprintf("%s %s %s\n", knownhosts.Normalize(hostname), key.Type(), base64.StdEncoding.EncodeToString(key.Marshal()))...,
		)
		return nil
	}
}
