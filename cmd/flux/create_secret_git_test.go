package main

import (
	"fmt"
	"os"
	"testing"
)

func TestCreateGitSecret(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "ca-crt")
	if err != nil {
		t.Fatal("could not create CA certificate file")
	}
	_, err = file.Write([]byte("ca-data"))
	if err != nil {
		t.Fatal("could not write to CA certificate file")
	}

	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "no args",
			args:   "create secret git",
			assert: assertError("name is required"),
		},
		{
			name:   "basic secret",
			args:   "create secret git podinfo-auth --url=https://github.com/stefanprodan/podinfo --username=my-username --password=my-password --namespace=my-namespace --export",
			assert: assertGoldenFile("./testdata/create_secret/git/secret-git-basic.yaml"),
		},
		{
			name:   "ssh key",
			args:   "create secret git podinfo-auth --url=ssh://git@github.com/stefanprodan/podinfo --private-key-file=./testdata/create_secret/git/ecdsa.private --namespace=my-namespace --export",
			assert: assertGoldenFile("testdata/create_secret/git/git-ssh-secret.yaml"),
		},
		{
			name:   "ssh key with password",
			args:   "create secret git podinfo-auth --url=ssh://git@github.com/stefanprodan/podinfo --private-key-file=./testdata/create_secret/git/ecdsa-password.private --password=password --namespace=my-namespace --export",
			assert: assertGoldenFile("testdata/create_secret/git/git-ssh-secret-password.yaml"),
		},
		{
			name:   "git authentication with bearer token",
			args:   "create secret git bearer-token-auth --url=https://github.com/stefanprodan/podinfo --bearer-token=ghp_baR2qnFF0O41WlucePL3udt2N9vVZS4R0hAS --namespace=my-namespace --export",
			assert: assertGoldenFile("testdata/create_secret/git/git-bearer-token.yaml"),
		},
		{
			name:   "git authentication with CA certificate",
			args:   fmt.Sprintf("create secret git ca-crt --url=https://github.com/stefanprodan/podinfo --password=my-password --username=my-username --ca-crt-file=%s --namespace=my-namespace --export", file.Name()),
			assert: assertGoldenFile("testdata/create_secret/git/secret-ca-crt.yaml"),
		},
		{
			name:   "git authentication with basic auth and bearer token",
			args:   "create secret git podinfo-auth --url=https://github.com/stefanprodan/podinfo --username=aaa --password=zzzz --bearer-token=aaaa --namespace=my-namespace --export",
			assert: assertError("user credentials and bearer token cannot be used together"),
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
