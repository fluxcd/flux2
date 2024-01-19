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
	"strings"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var supportedDecryptionProviders = []string{"sops"}

type DecryptionProvider string

func (d *DecryptionProvider) String() string {
	return string(*d)
}

func (d *DecryptionProvider) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		return fmt.Errorf("no decryption provider given, must be one of: %s",
			strings.Join(supportedDecryptionProviders, ", "))
	}
	if !utils.ContainsItemString(supportedDecryptionProviders, str) {
		return fmt.Errorf("unsupported decryption provider '%s', must be one of: %s",
			str, strings.Join(supportedDecryptionProviders, ", "))

	}
	*d = DecryptionProvider(str)
	return nil
}

func (d *DecryptionProvider) Type() string {
	return "decryptionProvider"
}

func (d *DecryptionProvider) Description() string {
	return fmt.Sprintf("decryption provider, available options are: (%s)", strings.Join(supportedDecryptionProviders, ", "))
}
