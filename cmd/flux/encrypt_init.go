/*
Copyright 2021 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"filippo.io/age"
	"github.com/fluxcd/flux2/internal/utils"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var encryptInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Init SOPS encryption with age identity",
	Long:  "The encryption init command creates a new age identity and writes a .sops.yaml file to the current working directory.",
	Example: `  # Init SOPS encryption with a new age identity
  flux encryption init`,
	RunE: encryptInitCmdRun,
}

func init() {
	encryptCmd.AddCommand(encryptInitCmd)
}

func encryptInitCmdRun(cmd *cobra.Command, args []string) error {
	// Confirm our current path is in a Git repository
	path, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := git.PlainOpen(path); err != nil {
		if err == git.ErrRepositoryNotExists {
			err = fmt.Errorf("'%s' is not in a Git repository", path)
		}
		return err
	}

	// Abort early if .sops.yaml already exists
	sopsCfgPath := filepath.Join(path, ".sops.yaml")
	if _, err := os.Stat(sopsCfgPath); err == nil || os.IsExist(err) {
		return fmt.Errorf("'%s' already contains a .sops.yaml config", path)
	}

	// Generate a new identity
	i, err := age.GenerateX25519Identity()
	if err != nil {
		return err
	}
	logger.Successf("Generated identity %s", i.Recipient().String())

	// Attempt to configure identity in .sops.yaml
	const sopsCfg = `creation_rules:
  - path_regex: .*.yaml
    encrypted_regex: ^(data|stringData)$
    age: %s
`
	if err := ioutil.WriteFile(sopsCfgPath, []byte(fmt.Sprintf(sopsCfg, i.Recipient().String())), 0644); err != nil {
		logger.Failuref("Failed to write recipient to .sops.yaml file")
		return err
	}
	logger.Successf("Configured recipient in .sops.yaml file")

	// Init client
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()
	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	// Create a secret
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "sops-age",
			Namespace: rootArgs.namespace,
		},
		StringData: map[string]string{
			"flux-auto.age": i.String(),
		},
	}
	if err := kubeClient.Create(ctx, secret); err != nil {
		return err
	}
	logger.Successf(`Secret '%s' with private key created`, secret.Name)

	// TODO(hidde): lookup kustomize based on path ref? Do direct cluster mutation? (Preferably not!)
	//  Feels something is missing in general to provide a user experience improving bridge between "die hard"
	//  `--export` and "please do not do this" direct-apply-to-cluster.

	return nil
}
