/*
Copyright 2023 The Flux authors

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
	"bytes"
	"io"
	"os"

	"github.com/mattn/go-shellwords"
)

// resetCmdArgs resets the root command's args and input to their defaults
func resetCmdArgs() {
	rootCmd.SetArgs(nil)
	rootCmd.SetIn(os.Stdin)
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
}

// executeCommand executes a command and returns its output
func executeCommand(cmd string) (string, error) {
	return executeCommandWithIn(cmd, nil)
}

// executeCommandWithIn executes a command with the provided input and returns its output
func executeCommandWithIn(cmd string, in io.Reader) (string, error) {
	defer resetCmdArgs()
	
	args, err := shellwords.Parse(cmd)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	if in != nil {
		rootCmd.SetIn(in)
	}

	_, err = rootCmd.ExecuteC()
	result := buf.String()

	return result, err
}
