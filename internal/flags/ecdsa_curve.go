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
	"crypto/elliptic"
	"fmt"
	"sort"
	"strings"
)

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
	return fmt.Errorf("unsupported curve '%s', must be one of: %s", str, strings.Join(ecdsaCurves(), ", "))
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
