//go:build !windows

/*
Copyright 2026 The Flux authors

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

package plugin

import (
	"os"
	"syscall"
)

// Exec replaces the current process with the plugin binary.
// This is what kubectl does — no signal forwarding or exit code propagation needed.
func Exec(path string, args []string) error {
	return syscall.Exec(path, append([]string{path}, args...), os.Environ())
}
