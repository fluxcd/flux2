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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generates zsh completion scripts",
	Long:  `The completion sub-command generates completion scripts for zsh.`,
	Example: `To load completion run

. <(flux completion zsh)

To configure your zsh shell to load completions for each session add to your zshrc

# ~/.zshrc or ~/.profile
command -v flux >/dev/null && . <(flux completion zsh)

or write a cached file in one of the completion directories in your ${fpath}:

echo "${fpath// /\n}" | grep -i completion
flux completion zsh > _flux

mv _flux ~/.oh-my-zsh/completions  # oh-my-zsh
mv _flux ~/.zprezto/modules/completion/external/src/  # zprezto`,
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenZshCompletion(os.Stdout)
		// Cobra doesn't source zsh completion file, explicitly doing it here
		fmt.Println("compdef _flux flux")
	},
}

func init() {
	completionCmd.AddCommand(completionZshCmd)
}
