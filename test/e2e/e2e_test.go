// +build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gotest.tools/golden"
)

func TestFluxInstallManifests(t *testing.T) {
	actual, err := runAsBytes(fluxBinary, "install")
	assert.NoError(t, err)

	golden.Assert(t, string(actual), "../../../testdata/flux_install_manifests.golden")
}

func TestFluxInstallManifestsExtra(t *testing.T) {
	actual, err := runAsBytes(fluxBinary, "install", "--components-extra", "image-reflector-controller,image-automation-controller")
	assert.NoError(t, err)

	golden.Assert(t, string(actual), "../../../testdata/flux_install_manifests_extra.golden")
}
