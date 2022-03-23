/*
Copyright 2021 The Flux authors

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

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/internal/utils"
)

var supportedGitImplementations = []string{sourcev1.GoGitImplementation, sourcev1.LibGit2Implementation}

type GitImplementation string

func (i *GitImplementation) String() string {
	return string(*i)
}

func (i *GitImplementation) Set(str string) error {
	if str == "" {
		return nil
	}
	if !utils.ContainsItemString(supportedGitImplementations, str) {
		return fmt.Errorf("unsupported Git implementation '%s', must be one of: %s",
			str, strings.Join(supportedGitImplementations, ", "))

	}
	*i = GitImplementation(str)
	return nil
}

func (i *GitImplementation) Type() string {
	return "gitImplementation"
}

func (i *GitImplementation) Description() string {
	return fmt.Sprintf("the Git implementation to use, available options are: (%s)", strings.Join(supportedGitImplementations, ", "))
}
