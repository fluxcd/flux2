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
	"github.com/spf13/cobra"
)

var getImageCmd = &cobra.Command{
	Use:     "images",
	Aliases: []string{"image"},
	Short:   "Get image automation object status",
	Long:    `The get image sub-commands print the status of image automation objects.`,
}

func init() {
	getCmd.AddCommand(getImageCmd)
}
