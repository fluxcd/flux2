/*
Copyright 2020 The Flux authors

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
	"time"

	"github.com/spf13/cobra"
)

var createSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Create or update sources",
	Long:  `The create source sub-commands generate sources.`,
}

type createSourceFlags struct {
	fetchTimeout time.Duration
}

var createSourceArgs createSourceFlags

func init() {
	createSourceCmd.PersistentFlags().DurationVar(&createSourceArgs.fetchTimeout, "fetch-timeout", createSourceArgs.fetchTimeout,
		"set a timeout for fetch operations performed by source-controller (e.g. 'git clone' or 'helm repo update')")
	createCmd.AddCommand(createSourceCmd)
}
