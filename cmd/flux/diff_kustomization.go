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

	"github.com/fluxcd/flux2/v2/internal/build"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
)

var diffKsCmd = &cobra.Command{
	Use:     "kustomization",
	Aliases: []string{"ks"},
	Short:   "Diff Kustomization",
	Long: `The diff command does a build, then it performs a server-side dry-run and prints the diff.
Exit status: 0 No differences were found. 1 Differences were found. >1 diff failed with an error.`,
	Example: `# Preview local changes as they were applied on the cluster
flux diff kustomization my-app --path ./path/to/local/manifests

# Preview using a local flux kustomization file
flux diff kustomization my-app --path ./path/to/local/manifests \
	--kustomization-file ./path/to/local/my-app.yaml

# Exclude files by providing a comma separated list of entries that follow the .gitignore pattern fromat.
flux diff kustomization my-app --path ./path/to/local/manifests \
	--kustomization-file ./path/to/local/my-app.yaml \
	--ignore-paths "/to_ignore/**/*.yaml,ignore.yaml"`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE:              diffKsCmdRun,
}

type diffKsFlags struct {
	kustomizationFile string
	path              string
	ignorePaths       []string
	progressBar       bool
}

var diffKsArgs diffKsFlags

func init() {
	diffKsCmd.Flags().StringVar(&diffKsArgs.path, "path", "", "Path to a local directory that matches the specified Kustomization.spec.path.")
	diffKsCmd.Flags().BoolVar(&diffKsArgs.progressBar, "progress-bar", true, "Boolean to set the progress bar. The default value is true.")
	diffKsCmd.Flags().StringSliceVar(&diffKsArgs.ignorePaths, "ignore-paths", nil, "set paths to ignore in .gitignore format")
	diffKsCmd.Flags().StringVar(&diffKsArgs.kustomizationFile, "kustomization-file", "", "Path to the Flux Kustomization YAML file.")
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

	if diffKsArgs.kustomizationFile != "" {
		if fs, err := os.Stat(diffKsArgs.kustomizationFile); os.IsNotExist(err) || fs.IsDir() {
			return fmt.Errorf("invalid kustomization file %q", diffKsArgs.kustomizationFile)
		}
	}

	var (
		builder *build.Builder
		err     error
	)
	if diffKsArgs.progressBar {
		builder, err = build.NewBuilder(name, diffKsArgs.path,
			build.WithClientConfig(kubeconfigArgs, kubeclientOptions),
			build.WithTimeout(rootArgs.timeout),
			build.WithKustomizationFile(diffKsArgs.kustomizationFile),
			build.WithProgressBar(),
			build.WithIgnore(diffKsArgs.ignorePaths),
		)
	} else {
		builder, err = build.NewBuilder(name, diffKsArgs.path,
			build.WithClientConfig(kubeconfigArgs, kubeclientOptions),
			build.WithTimeout(rootArgs.timeout),
			build.WithKustomizationFile(diffKsArgs.kustomizationFile),
			build.WithIgnore(diffKsArgs.ignorePaths),
		)
	}

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
