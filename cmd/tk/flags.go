package main

import (
	"crypto/elliptic"
	"fmt"
	"strconv"
	"strings"
)

var supportedPublicKeyAlgorithms = []string{"rsa", "ecdsa"}

type PublicKeyAlgorithm string

func (a *PublicKeyAlgorithm) String() string {
	return string(*a)
}

func (a *PublicKeyAlgorithm) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		*a = PublicKeyAlgorithm(supportedPublicKeyAlgorithms[0])
		return nil
	}
	for _, v := range supportedPublicKeyAlgorithms {
		if str == v {
			*a = PublicKeyAlgorithm(str)
			return nil
		}
	}
	return fmt.Errorf(
		"unsupported public key algorithm '%s', must be one of: %s",
		str,
		strings.Join(supportedPublicKeyAlgorithms, ", "),
	)
}

func (a *PublicKeyAlgorithm) Type() string {
	return "publicKeyAlgorithm"
}

var defaultRSAKeyBits = 2048

type RSAKeyBits int

func (b *RSAKeyBits) String() string {
	return strconv.Itoa(int(*b))
}

func (b *RSAKeyBits) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		*b = RSAKeyBits(defaultRSAKeyBits)
		return nil
	}
	bits, err := strconv.Atoi(str)
	if err != nil {
		return err
	}
	if bits%8 != 0 {
		return fmt.Errorf("RSA key bit size should be a multiples of 8")
	}
	*b = RSAKeyBits(bits)
	return nil
}

func (b *RSAKeyBits) Type() string {
	return "rsaKeyBits"
}

type ECDSACurve struct {
	elliptic.Curve
}

var supportedECDSACurves = map[string]elliptic.Curve{
	"P-256": elliptic.P256(),
	"P-384": elliptic.P384(),
	"P-521": elliptic.P521(),
}

func (c *ECDSACurve) String() string {
	if c == nil || c.Curve == nil {
		return ""
	}
	return c.Curve.Params().Name
}

func (c *ECDSACurve) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		*c = ECDSACurve{supportedECDSACurves["P-384"]}
		return nil
	}
	for k, v := range supportedECDSACurves {
		if k == str {
			*c = ECDSACurve{v}
			return nil
		}
	}
	return fmt.Errorf("unsupported curve '%s', should be one of: P-256, P-384, P-521", str)
}

func (c *ECDSACurve) Type() string {
	return "ecdsaCurve"
}
