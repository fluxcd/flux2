package main

import (
	"testing"
)

func TestCreateHelmSecret(t *testing.T) {
	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			args:   "create secret helm",
			assert: assertError("secret name is required"),
		},
		{
			args:   "create secret helm helm-secret --username=my-username --password=my-password --namespace=my-namespace --export",
			assert: assertGoldenFile("testdata/create_secret/helm/secret-helm.yaml"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.args,
				assert: tt.assert,
			}
			cmd.runTestCmd(t)
		})
	}
}
