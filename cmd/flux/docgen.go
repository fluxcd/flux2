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
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

const fmTemplate = `---
title: "%s"
---
`

var (
	cmdDocPath string
)

var docgenCmd = &cobra.Command{
	Use:    "docgen",
	Short:  "Generate the documentation for the CLI commands.",
	Hidden: true,
	RunE:   docgenCmdRun,
}

func init() {
	docgenCmd.Flags().StringVar(&cmdDocPath, "path", "./docs/cmd", "path to write the generated documentation to")

	rootCmd.AddCommand(docgenCmd)
}

func docgenCmdRun(cmd *cobra.Command, args []string) error {
	err := doc.GenMarkdownTreeCustom(rootCmd, cmdDocPath, frontmatterPrepender, linkHandler)
	if err != nil {
		return err
	}
	return nil
}

func frontmatterPrepender(filename string) string {
	name := filepath.Base(filename)
	base := strings.TrimSuffix(name, path.Ext(name))
	title := strings.Replace(base, "_", " ", -1)
	return fmt.Sprintf(fmTemplate, title)
}

func linkHandler(name string) string {
	base := strings.TrimSuffix(name, path.Ext(name))
	return "../" + strings.ToLower(base) + "/"
}
