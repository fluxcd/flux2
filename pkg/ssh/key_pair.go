package ssh

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

// KeyPair holds the public and private key PEM block bytes.
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
	priv, err := encodePrivateKeyToPEM(pk)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
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
	priv, err := encodePrivateKeyToPEM(pk)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

type Ed25519Generator struct{}

func NewEd25519Generator() KeyPairGenerator {
	return &Ed25519Generator{}
}

func (g *Ed25519Generator) Generate() (*KeyPair, error) {
	pk, pv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	pub, err := generatePublicKey(pk)
	if err != nil {
		return nil, err
	}
	priv, err := encodePrivateKeyToPEM(pv)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
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

// encodePrivateKeyToPEM encodes the given private key to a PEM block.
// The encoded format is PKCS#8 for universal support of the most
// common key types (rsa, ecdsa, ed25519).
func encodePrivateKeyToPEM(pk interface{}) ([]byte, error) {
	b, err := x509.MarshalPKCS8PrivateKey(pk)
	if err != nil {
		return nil, err
	}
	block := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: b,
	}
	return pem.EncodeToMemory(&block), nil
}
