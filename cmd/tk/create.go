/*
Copyright 2020 The Flux CD contributors.

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

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create or update sources and resources",
	Long:  "The create sub-commands generate sources and resources.",
}

var (
	interval time.Duration
	export   bool
)

func init() {
	createCmd.PersistentFlags().DurationVarP(&interval, "interval", "", time.Minute, "source sync interval")
	createCmd.PersistentFlags().BoolVar(&export, "export", false, "export in YAML format to stdout")
	rootCmd.AddCommand(createCmd)
}
