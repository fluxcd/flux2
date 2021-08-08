package main

import (
	"testing"
)

func TestCreateTlsSecretNoArgs(t *testing.T) {
	cmd := cmdTestCase{
		args:        "create secret tls",
		wantError:   true,
		goldenValue: "secret name is required",
	}
	cmd.runTestCmd(t)
}

func TestCreateTlsSecret(t *testing.T) {
	cmd := cmdTestCase{
		args:       "create secret tls certs --namespace=my-namespace --cert-file=./testdata/create/secret/tls/test-cert.pem --key-file=./testdata/create/secret/tls/test-key.pem --export",
		wantError:  false,
		goldenFile: "testdata/create/secret/tls/secret-tls.txt",
	}
	cmd.runTestCmd(t)
}
