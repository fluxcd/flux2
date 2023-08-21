/*
Copyright 2020 The Flux authors

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

package flags

import (
	"fmt"
	"strconv"
	"strings"
)

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
	if bits < 1024 {
		return fmt.Errorf("RSA key bit size must be at least 1024")
	}
	if bits == 0 || bits%8 != 0 {
		return fmt.Errorf("RSA key bit size must be a multiples of 8")
	}
	*b = RSAKeyBits(bits)
	return nil
}

func (b *RSAKeyBits) Type() string {
	return "rsaKeyBits"
}

func (b *RSAKeyBits) Description() string {
	return "SSH RSA public key bit size (multiplies of 8, min 1024)"
}
