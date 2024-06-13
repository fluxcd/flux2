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

package main

import (
	"fmt"
	"strings"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
)

func getVersion(input string) (string, error) {
	if input == "" {
		return rootArgs.defaults.Version, nil
	}

	if input != install.MakeDefaultOptions().Version && !strings.HasPrefix(input, "v") {
		return "", fmt.Errorf("targeted version '%s' must be prefixed with 'v'", input)
	}

	if isEmbeddedVersion(input) {
		return input, nil
	}

	var err error
	if input == install.MakeDefaultOptions().Version {
		input, err = install.GetLatestVersion()
		if err != nil {
			return "", err
		}
	} else {
		if ok, err := install.ExistingVersion(input); err != nil || !ok {
			if err == nil {
				err = fmt.Errorf("targeted version '%s' does not exist", input)
			}
			return "", err
		}
	}

	if !utils.CompatibleVersion(VERSION, input) {
		return "", fmt.Errorf("targeted version '%s' is not compatible with your current version of flux (%s)", input, VERSION)
	}
	return input, nil
}

func isEmbeddedVersion(input string) bool {
	return input == rootArgs.defaults.Version
}
