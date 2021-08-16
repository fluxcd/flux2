// +build e2e

package main

import (
	"testing"
	"time"
)

func TestInstallNoArgs(t *testing.T) {
	cmd := cmdTestCase{
		args:            "install",
		wantError:       false,
		testClusterMode: ExistingClusterMode,
		goldenFile:      "testdata/install/install_no_args.golden",
	}
	cmd.runTestCmd(t)

	testUninstallSilent(t)
	time.Sleep(30 * time.Second)
}

func TestInstallExtraComponents(t *testing.T) {
	cmd := cmdTestCase{
		args:            "install --components-extra=image-reflector-controller,image-automation-controller",
		wantError:       false,
		testClusterMode: ExistingClusterMode,
		goldenFile:      "testdata/install/install_extra_components.golden",
	}
	cmd.runTestCmd(t)

	testUninstallSilentForExtraComponents(t)
	time.Sleep(30 * time.Second)
}

func testUninstallSilent(t *testing.T) {
	cmd := cmdTestCase{
		args:            "uninstall -s",
		wantError:       false,
		testClusterMode: ExistingClusterMode,
		goldenFile:      "testdata/uninstall/uninstall.golden",
	}
	cmd.runTestCmd(t)
}

func testUninstallSilentForExtraComponents(t *testing.T) {
	cmd := cmdTestCase{
		args:            "uninstall -s",
		wantError:       false,
		testClusterMode: ExistingClusterMode,
		goldenFile:      "testdata/uninstall/uninstall_extra_components.golden",
	}
	cmd.runTestCmd(t)
}
