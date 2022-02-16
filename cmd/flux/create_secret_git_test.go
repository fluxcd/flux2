package main

import (
	"testing"
)

func TestCreateGitSecret(t *testing.T) {
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
