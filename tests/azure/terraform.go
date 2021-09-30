package test

import (
	"context"
	"os/exec"
	"strings"
)

type whichTerraform struct{}

func (w *whichTerraform) ExecPath(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "which", "terraform")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	path := strings.TrimSuffix(string(output), "\n")
	return path, nil
}
