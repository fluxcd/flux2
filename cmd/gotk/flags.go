/*
Copyright 2020 The Flux CD contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/elliptic"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var supportedPublicKeyAlgorithms = []string{"rsa", "ecdsa", "ed25519"}

type PublicKeyAlgorithm string

func (a *PublicKeyAlgorithm) String() string {
	return string(*a)
}

func (a *PublicKeyAlgorithm) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no public key algorithm given, must be one of: %s",
			strings.Join(supportedPublicKeyAlgorithms, ", "))
	}
	for _, v := range supportedPublicKeyAlgorithms {
		if str == v {
			*a = PublicKeyAlgorithm(str)
			return nil
		}
	}
	return fmt.Errorf("unsupported public key algorithm '%s', must be one of: %s",
		str, strings.Join(supportedPublicKeyAlgorithms, ", "))
}

func (a *PublicKeyAlgorithm) Type() string {
	return "publicKeyAlgorithm"
}

func (a *PublicKeyAlgorithm) Description() string {
	return fmt.Sprintf("SSH public key algorithm (%s)", strings.Join(supportedPublicKeyAlgorithms, ", "))
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

func (b *RSAKeyBits) Description() string {
	return "SSH RSA public key bit size (multiplies of 8)"
}

type ECDSACurve struct {
	elliptic.Curve
}

var supportedECDSACurves = map[string]elliptic.Curve{
	"p256": elliptic.P256(),
	"p384": elliptic.P384(),
	"p521": elliptic.P521(),
}

func (c *ECDSACurve) String() string {
	if c.Curve == nil {
		return ""
	}
	return strings.ToLower(strings.Replace(c.Curve.Params().Name, "-", "", 1))
}

func (c *ECDSACurve) Set(str string) error {
	if v, ok := supportedECDSACurves[str]; ok {
		*c = ECDSACurve{v}
		return nil
	}
	return fmt.Errorf("unsupported curve '%s', should be one of: %s", str, strings.Join(ecdsaCurves(), ", "))
}

func (c *ECDSACurve) Type() string {
	return "ecdsaCurve"
}

func (c *ECDSACurve) Description() string {
	return fmt.Sprintf("SSH ECDSA public key curve (%s)", strings.Join(ecdsaCurves(), ", "))
}

func ecdsaCurves() []string {
	keys := make([]string, 0, len(supportedECDSACurves))
	for k := range supportedECDSACurves {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
