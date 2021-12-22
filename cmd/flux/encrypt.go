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
	"github.com/spf13/cobra"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypt secrets using SOPS",
	Long:  "The encrypt sub-commands initialise and manage Secret encryption using SOPS.",
}

type encryptFlags struct {
	export bool
}

var encryptArgs encryptFlags

func init() {
	encryptCmd.PersistentFlags().BoolVar(&encryptArgs.export, "export", false, "export in YAML format to stdout")

	rootCmd.AddCommand(encryptCmd)
}
