package ssh

import (
	"encoding/base64"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ScanHostKey collects the given host's preferred public key for the
// Any errors (e.g. authentication  failures) are ignored, except if
// no key could be collected from the host.
func ScanHostKey(host string, timeout time.Duration) ([]byte, error) {
	col := &HostKeyCollector{}
	config := &ssh.ClientConfig{
		HostKeyCallback: col.StoreKey(),
		Timeout:         timeout,
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

// HostKeyCollector offers a StoreKey method which provides an
// HostKeyCallBack to collect public keys from an SSH server.
type HostKeyCollector struct {
	knownKeys []byte
}

// StoreKey stores the public key in bytes as returned by the host.
// To collect multiple public key types from the host, multiple
// SSH dials need with the ClientConfig HostKeyAlgorithms set to
// the algorithm you want to collect.
func (c *HostKeyCollector) StoreKey() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		c.knownKeys = append(
			c.knownKeys,
			fmt.Sprintf("%s %s %s\n", knownhosts.Normalize(hostname), key.Type(), base64.StdEncoding.EncodeToString(key.Marshal()))...,
		)
		return nil
	}
}

// GetKnownKeys returns the collected public keys in bytes.
func (c *HostKeyCollector) GetKnownKeys() []byte {
	return c.knownKeys
}
