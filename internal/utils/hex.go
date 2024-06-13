/*
Copyright 2023 The Flux authors

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

package utils

import (
	"regexp"
	"strings"
)

// hexRegexp matches any hexadecimal notation between 40 and 128 characters.
var hexRegexp = regexp.MustCompile(`\b[a-f0-9]{40,128}\b`)

// TruncateHex will replace any hexadecimal notation between 40 and 128
// characters (SHA-1 up to SHA-512) within the given string with a truncated
// version of 8 characters.
func TruncateHex(str string) string {
	if str == "" {
		return ""
	}
	hits := hexRegexp.FindAllString(str, -1)
	for _, v := range hits {
		str = strings.Replace(str, v, string([]rune(v)[:8]), -1)
	}
	return str
}
