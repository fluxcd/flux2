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
	"os"

	"github.com/spf13/cobra"
)

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generates zsh completion scripts",
	Example: `To load completion run

. <(gotk completion zsh) && compdef _gotk gotk

To configure your zsh shell to load completions for each session add to your zshrc

# ~/.zshrc or ~/.profile
command -v gotk >/dev/null && . <(gotk completion zsh) && compdef _gotk gotk

or write a cached file in one of the completion directories in your ${fpath}:

echo "${fpath// /\n}" | grep -i completion
gotk completions zsh > _gotk

mv _gotk ~/.oh-my-zsh/completions  # oh-my-zsh
mv _gotk ~/.zprezto/modules/completion/external/src/  # zprezto
`,
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd.GenZshCompletion(os.Stdout)
	},
}

func init() {
	completionCmd.AddCommand(completionZshCmd)
}
