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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml/goyaml.v2"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the client and server-side components version information.",
	Long:  `Print the client and server-side components version information for the current context.`,
	Example: `# Print client and server-side version 
	flux version

	# Print only client version
	flux version --client

	# Print information in json format
	flux version -o json
`,
	RunE: versionCmdRun,
}

type versionFlags struct {
	client bool
	output string
}

var versionArgs versionFlags

type versionInfo struct {
	Flux         string            `yaml:"flux"`
	Distribution string            `yaml:"distribution,omitempty"`
	Controller   map[string]string `yaml:"controller,inline"`
}

func init() {
	versionCmd.Flags().BoolVar(&versionArgs.client, "client", false,
		"print only client version")
	versionCmd.Flags().StringVarP(&versionArgs.output, "output", "o", "yaml",
		"the format in which the information should be printed. can be 'json' or 'yaml'")
	rootCmd.AddCommand(versionCmd)
}

func versionCmdRun(cmd *cobra.Command, args []string) error {
	if versionArgs.output != "yaml" && versionArgs.output != "json" {
		return fmt.Errorf("--output must be json or yaml, not %s", versionArgs.output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	// versionInfo struct and goyaml is used because we care about the order.
	// Without this `distribution` is printed before `flux` when the struct is marshalled.
	info := &versionInfo{
		Controller: map[string]string{},
	}
	info.Flux = rootArgs.defaults.Version

	if !versionArgs.client {
		kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
		if err != nil {
			return err
		}

		clusterInfo, err := getFluxClusterInfo(ctx, kubeClient)
		// ignoring not found errors because it means that the GitRepository CRD isn't installed but a user might
		// have other controllers(e.g notification-controller), and  we want to still return information for them.
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		if clusterInfo.distribution() != "" {
			info.Distribution = clusterInfo.distribution()
		}

		selector := client.MatchingLabels{manifestgen.PartOfLabelKey: manifestgen.PartOfLabelValue}
		var list v1.DeploymentList
		if err := kubeClient.List(ctx, &list, client.InNamespace(*kubeconfigArgs.Namespace), selector); err != nil {
			return err
		}

		if len(list.Items) == 0 {
			return fmt.Errorf("no deployments found in %s namespace", *kubeconfigArgs.Namespace)
		}

		for _, d := range list.Items {
			for _, c := range d.Spec.Template.Spec.Containers {
				name, tag, err := splitImageStr(c.Image)
				if err != nil {
					return err
				}
				info.Controller[name] = tag
			}
		}
	}

	var marshalled []byte
	var err error

	if versionArgs.output == "json" {
		marshalled, err = info.toJSON()
		marshalled = append(marshalled, "\n"...)
	} else {
		marshalled, err = yaml.Marshal(&info)
	}

	if err != nil {
		return err
	}

	rootCmd.Print(string(marshalled))
	return nil
}

func (info versionInfo) toJSON() ([]byte, error) {
	mapInfo := map[string]string{
		"flux": info.Flux,
	}

	if info.Distribution != "" {
		mapInfo["distribution"] = info.Distribution
	}
	for k, v := range info.Controller {
		mapInfo[k] = v
	}
	return json.MarshalIndent(&mapInfo, "", "  ")
}

func splitImageStr(image string) (string, string, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return "", "", fmt.Errorf("parsing image '%s' failed: %w", image, err)
	}

	reg := ref.Context().RegistryStr()
	repo := strings.TrimPrefix(image, reg)
	parts := strings.Split(repo, ":")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("missing image tag in image %s", image)
	}

	n, t := parts[0], strings.TrimPrefix(repo, parts[0]+":")
	nameArr := strings.Split(n, "/")
	return nameArr[len(nameArr)-1], t, nil
}
