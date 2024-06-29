package flags

import (
	"fmt"
	"strings"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var supportedGitLabVisibilities = []string{
	"public",
	"internal",
	"private",
}

type GitLabVisibility string

func (d *GitLabVisibility) String() string {
	return string(*d)
}

func (d *GitLabVisibility) Set(str string) error {
	if strings.TrimSpace(str) == "" {
		str = "private"
	}
	if !utils.ContainsItemString(supportedGitLabVisibilities, str) {
		return fmt.Errorf("unsupported visibility '%s', must be one of: %s",
			str, strings.Join(supportedGitLabVisibilities, ", "))

	}
	*d = GitLabVisibility(str)
	return nil
}

func (d *GitLabVisibility) Type() string {
	return "GitLabVisibility"
}

func (d *GitLabVisibility) Description() string {
	return fmt.Sprintf("visibility, available options are: (%s)", strings.Join(supportedGitLabVisibilities, ", "))
}
