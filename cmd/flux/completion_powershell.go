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
	"os"

	"github.com/spf13/cobra"
)

var completionPowerShellCmd = &cobra.Command{
	Use:   "powershell",
	Short: "Generates powershell completion scripts",
	Long:  `The completion sub-command generates completion scripts for powershell.`,
	Example: `To load completion run

. <(flux completion powershell)

To configure your powershell shell to load completions for each session add to your powershell profile

Windows:

cd "$env:USERPROFILE\Documents\WindowsPowerShell\Modules"
flux completion powershell >> flux-completion.ps1

Linux:

cd "${XDG_CONFIG_HOME:-"$HOME/.config/"}/powershell/modules"
flux completion powershell >> flux-completions.ps1`,
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenPowerShellCompletion(os.Stdout)
	},
}

func init() {
	completionCmd.AddCommand(completionPowerShellCmd)
}
