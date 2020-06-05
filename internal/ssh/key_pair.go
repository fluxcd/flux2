package ssh

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

type KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
}

type KeyPairGenerator interface {
	Generate() (*KeyPair, error)
}

type RSAGenerator struct {
	bits int
}

func NewRSAGenerator(bits int) KeyPairGenerator {
	return &RSAGenerator{bits}
}

func (g *RSAGenerator) Generate() (*KeyPair, error) {
	pk, err := rsa.GenerateKey(rand.Reader, g.bits)
	if err != nil {
		return nil, err
	}
	err = pk.Validate()
	if err != nil {
		return nil, err
	}
	pub, err := generatePublicKey(&pk.PublicKey)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: encodePrivateKeyToPEM(pk),
	}, nil
}

type ECDSAGenerator struct {
	c elliptic.Curve
}

func NewECDSAGenerator(c elliptic.Curve) KeyPairGenerator {
	return &ECDSAGenerator{c}
}

func (g *ECDSAGenerator) Generate() (*KeyPair, error) {
	pk, err := ecdsa.GenerateKey(g.c, rand.Reader)
	if err != nil {
		return nil, err
	}
	pub, err := generatePublicKey(&pk.PublicKey)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: encodePrivateKeyToPEM(pk),
	}, nil
}

func generatePublicKey(pk interface{}) ([]byte, error) {
	b, err := ssh.NewPublicKey(pk)
	if err != nil {
		return nil, err
	}
	k := ssh.MarshalAuthorizedKey(b)
	return k, nil
}

func encodePrivateKeyToPEM(pk interface{}) []byte {
	b, _ := x509.MarshalPKCS8PrivateKey(pk)
	block := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: b,
	}
	return pem.EncodeToMemory(&block)
}
