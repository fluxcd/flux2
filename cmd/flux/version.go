package main

import (
	"fmt"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
)

func getVersion(input string) (string, error) {
	if input == "" {
		return rootArgs.defaults.Version, nil
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
