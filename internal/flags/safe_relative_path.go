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
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
)

type SafeRelativePath string

func (p *SafeRelativePath) String() string {
	return string(*p)
}

func (p *SafeRelativePath) ToSlash() string {
	return filepath.ToSlash(p.String())
}

func (p *SafeRelativePath) Set(str string) error {
	// The result of secure joining on a relative base dir is a flattened relative path.
	cleanP, err := securejoin.SecureJoin("./", strings.TrimSpace(str))
	if err != nil {
		return fmt.Errorf("invalid relative path '%s': %w", cleanP, err)
	}
	// NB: required, as a secure join of "./" will result in "."
	if cleanP == "." {
		cleanP = ""
	}
	cleanP = fmt.Sprintf("./%s", cleanP)
	*p = SafeRelativePath(cleanP)
	return nil
}

func (p *SafeRelativePath) Type() string {
	return "safeRelativePath"
}

func (p *SafeRelativePath) Description() string {
	return "secure relative path"
}
