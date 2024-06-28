/*
Copyright 2024 The Flux authors

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

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

var supportedGitLabVisibilities = map[gitprovider.RepositoryVisibility]struct{}{
	gitprovider.RepositoryVisibilityPublic:   {},
	gitprovider.RepositoryVisibilityInternal: {},
	gitprovider.RepositoryVisibilityPrivate:  {},
}

// ValidateRepositoryVisibility validates a given RepositoryVisibility.
func ValidateRepositoryVisibility(r gitprovider.RepositoryVisibility) error {
	_, ok := supportedGitLabVisibilities[r]
	if !ok {
		return validation.ErrFieldEnumInvalid
	}
	return nil
}

type GitLabVisibility gitprovider.RepositoryVisibility

func (d *GitLabVisibility) String() string {
	return string(*d)
}

func (d *GitLabVisibility) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		str = string(gitprovider.RepositoryVisibilityPrivate)
	}
	var visibility = gitprovider.RepositoryVisibility(str)
	if ValidateRepositoryVisibility(visibility) != nil {
		return fmt.Errorf("unsupported visibility '%s'", str)
	}
	*d = GitLabVisibility(visibility)
	return nil
}

func (d *GitLabVisibility) Type() string {
	return "gitLabVisibility"
}

func (d *GitLabVisibility) Description() string {
	return fmt.Sprintf("specifies the visibility of the repository. Valid values are public, private, internal")
}
