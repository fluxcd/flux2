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
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete sources and resources",
	Long:  "The delete sub-commands delete sources and resources.",
}

var (
	deleteSilent bool
)

func init() {
	deleteCmd.PersistentFlags().BoolVarP(&deleteSilent, "silent", "s", false,
		"delete resource without asking for confirmation")

	rootCmd.AddCommand(deleteCmd)
}
