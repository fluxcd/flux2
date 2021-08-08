package main

import (
	"testing"
)

func TestCreateHelmSecretNoArgs(t *testing.T) {
	cmd := cmdTestCase{
		args:        "create secret helm",
		wantError:   true,
		goldenValue: "secret name is required",
	}
	cmd.runTestCmd(t)
}

func TestCreateHelmSecret(t *testing.T) {
	cmd := cmdTestCase{
		args:       "create secret helm helm-secret --username=my-username --password=my-password --namespace=my-namespace --export",
		wantError:  false,
		goldenFile: "testdata/create/secret/helm/secret-helm.yaml",
	}
	cmd.runTestCmd(t)
}
