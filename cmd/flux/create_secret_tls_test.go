package main

import (
	"testing"
)

func TestCreateTlsSecret(t *testing.T) {
	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			args:   "create secret tls",
			assert: assertError("name is required"),
		},
		{
			args:   "create secret tls certs --namespace=my-namespace --tls-crt-file=./testdata/create_secret/tls/test-cert.pem --tls-key-file=./testdata/create_secret/tls/test-key.pem --ca-crt-file=./testdata/create_secret/tls/test-ca.pem --export",
			assert: assertGoldenFile("testdata/create_secret/tls/secret-tls.yaml"),
		},
		{
			args:   "create secret tls certs --namespace=my-namespace --cert-file=./testdata/create_secret/tls/test-cert.pem --key-file=./testdata/create_secret/tls/test-key.pem --ca-file=./testdata/create_secret/tls/test-ca.pem --export",
			assert: assertGoldenFile("testdata/create_secret/tls/deprecated-secret-tls.yaml"),
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
