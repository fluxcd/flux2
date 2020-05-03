package keyscan

import (
	"encoding/base64"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func ScanKeys(host string) ([]byte, error) {
	col := &collector{}
	config := &ssh.ClientConfig{
		User:            "git",
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
