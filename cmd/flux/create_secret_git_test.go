package main

import (
	"testing"
)

func TestCreateGitSecretNoArgs(t *testing.T) {
	cmd := cmdTestCase{
		args:        "create secret git",
		wantError:   true,
		goldenValue: "secret name is required",
	}
	cmd.runTestCmd(t)
}

func TestCreateGitBasicSecret(t *testing.T) {
	cmd := cmdTestCase{
		args:       "create secret git podinfo-auth --url=https://github.com/stefanprodan/podinfo --username=my-username --password=my-password --namespace=my-namespace --export",
		wantError:  false,
		goldenFile: "testdata/create/secret/git/secret-git-basic.yaml",
	}
	cmd.runTestCmd(t)
}

func TestCreateGitSSHSecret(t *testing.T) {
	cmd := cmdTestCase{
		args:       "create secret git podinfo-auth --url=ssh://git@github.com/stefanprodan/podinfo --private-key-file=./testdata/create/secret/git/rsa.private --namespace=my-namespace --export",
		wantError:  false,
		goldenFile: "testdata/create/secret/git/git-ssh-secret.yaml",
	}
	cmd.runTestCmd(t)
}

func TestCreateGitSSHPasswordSecret(t *testing.T) {
	cmd := cmdTestCase{
		args:       "create secret git podinfo-auth --url=ssh://git@github.com/stefanprodan/podinfo --private-key-file=./testdata/create/secret/git/rsa-password.private --password=password --namespace=my-namespace --export",
		wantError:  false,
		goldenFile: "testdata/create/secret/git/git-ssh-secret-password.yaml",
	}
	cmd.runTestCmd(t)
}
