// +build e2e

package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var fluxBinary string

func init() {
	fluxBinary = os.Getenv("FLUX_E2E_BINARY")
	if fluxBinary == "" {
		fluxBinary = "../../bin/flux"
	}
}

func run(name string, args ...string) ([]string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	lines := strings.Split(string(out), "\n")
	return lines, err
}

func runAsBytes(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

func getAndWaitUntilObjectNotFound(args ...string) {
	for ; ; {
		var lines []string
		var err error

		lines, err = run(fluxBinary, append([]string{"get"}, args...)...)
		if err != nil {
			return
		}

		if args[0] == "source" &&
			args[1] == "git" &&
			lines[0] == "✗ no GitRepository objects found in flux-system namespace" {
			return
		}

		if args[0] == "kustomization" &&
			lines[0] == "✗ no Kustomization objects found in flux-system namespace" {
			return
		}

		time.Sleep(3 * time.Second)
	}
}

func deleteObject(t *testing.T, args ...string) {
	_, err := run(fluxBinary, append([]string{"delete"}, args...)...)
	assert.NoError(t, err)
}
