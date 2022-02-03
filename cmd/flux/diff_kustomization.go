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
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/internal/build"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
)

var diffKsCmd = &cobra.Command{
	Use:     "kustomization",
	Aliases: []string{"ks"},
	Short:   "Diff Kustomization",
	Long: `The diff command does a build, then it performs a server-side dry-run and prints the diff.
Exit status: 0 No differences were found. 1 Differences were found. >1 diff failed with an error.`,
	Example: `# Preview local changes as they were applied on the cluster
flux diff kustomization my-app --path ./path/to/local/manifests`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE:              diffKsCmdRun,
}

type diffKsFlags struct {
	path string
}

var diffKsArgs diffKsFlags

func init() {
	diffKsCmd.Flags().StringVar(&diffKsArgs.path, "path", "", "Path to a local directory that matches the specified Kustomization.spec.path.)")
	diffCmd.AddCommand(diffKsCmd)
}

func diffKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", kustomizationType.humanKind)
	}
	name := args[0]

	if diffKsArgs.path == "" {
		return &RequestError{StatusCode: 2, Err: fmt.Errorf("invalid resource path %q", diffKsArgs.path)}
	}

	if fs, err := os.Stat(diffKsArgs.path); err != nil || !fs.IsDir() {
		return &RequestError{StatusCode: 2, Err: fmt.Errorf("invalid resource path %q", diffKsArgs.path)}
	}

	builder, err := build.NewBuilder(kubeconfigArgs, name, diffKsArgs.path, build.WithTimeout(rootArgs.timeout))
	if err != nil {
		return &RequestError{StatusCode: 2, Err: err}
	}

	// create a signal channel
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)

	errChan := make(chan error)
	go func() {
		output, hasChanged, err := builder.Diff()
		if err != nil {
			errChan <- &RequestError{StatusCode: 2, Err: err}
		}

		cmd.Print(output)

		if hasChanged {
			errChan <- &RequestError{StatusCode: 1, Err: fmt.Errorf("identified at least one change, exiting with non-zero exit code")}
		} else {
			errChan <- nil
		}
	}()

	select {
	case <-sigc:
		fmt.Println("Build cancelled... exiting.")
		return builder.Cancel()
	case err := <-errChan:
		if err != nil {
			return err
		}
	}

	return nil

}
