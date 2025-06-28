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
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// KubeConfigArgs contains the args related to kubeconfig and cluster access
type KubeConfigArgs struct {
	KubeConfig          string
	KubeContext         string
	InsecureSkipVerify  bool
	Namespace           *string
	defaultClientConfig clientcmd.ClientConfig
}

// NewKubeConfigArgs returns a new KubeConfigArgs
func NewKubeConfigArgs() *KubeConfigArgs {
	namespace := ""
	return &KubeConfigArgs{
		Namespace: &namespace,
	}
}

// BindFlags binds the KubeConfigArgs fields to the given flag set
func (ka *KubeConfigArgs) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&ka.KubeConfig, "kubeconfig", "",
		"Path to the kubeconfig file")
	flags.StringVar(&ka.KubeContext, "context", "",
		"Kubernetes context to use")
	flags.BoolVar(&ka.InsecureSkipVerify, "insecure-skip-tls-verify", false,
		"Skip TLS certificate validation when connecting to the Kubernetes API server")
	flags.StringVar(ka.Namespace, "namespace", "", 
		"The namespace scope for this operation")
}

// loadKubeConfig loads the kubeconfig file based on the provided args and KUBECONFIG env var
func (ka *KubeConfigArgs) loadKubeConfig() (*clientcmdapi.Config, error) {
	// If kubeconfig is explicitly provided, use it
	if ka.KubeConfig != "" {
		return clientcmd.LoadFromFile(ka.KubeConfig)
	}

	// Check if KUBECONFIG env var is set
	kubeconfigEnv := os.Getenv("KUBECONFIG")
	if kubeconfigEnv != "" {
		// KUBECONFIG can contain multiple paths
		paths := filepath.SplitList(kubeconfigEnv)
		if len(paths) > 1 {
			// Merge multiple kubeconfig files
			loadingRules := clientcmd.ClientConfigLoadingRules{
				Precedence: paths,
			}
			return loadingRules.Load()
		} else if len(paths) == 1 {
			// Single path in KUBECONFIG
			return clientcmd.LoadFromFile(paths[0])
		}
	}

	// Fall back to default kubeconfig location
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return loadingRules.Load()
}

// kubeConfig returns a complete client config based on the provided args
func (ka *KubeConfigArgs) kubeConfig(explicitPath string) (clientcmd.ClientConfig, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if explicitPath != "" {
		loadingRules.ExplicitPath = explicitPath
	} else if ka.KubeConfig != "" {
		loadingRules.ExplicitPath = ka.KubeConfig
	} else if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		// KUBECONFIG is a list of files separated by path list separator
		paths := filepath.SplitList(kubeconfig)
		if len(paths) > 0 {
			loadingRules.Precedence = paths
		}
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	if ka.KubeContext != "" {
		configOverrides.CurrentContext = ka.KubeContext
	}
	if ka.InsecureSkipVerify {
		configOverrides.ClusterInfo.InsecureSkipTLSVerify = true
	}
	if *ka.Namespace != "" {
		configOverrides.Context.Namespace = *ka.Namespace
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	), nil
}

// KubeConfig returns a complete rest.Config based on the provided args
func (ka *KubeConfigArgs) KubeConfig(explicitPath string) (*rest.Config, error) {
	clientConfig, err := ka.kubeConfig(explicitPath)
	if err != nil {
		return nil, err
	}

	config, err := clientConfig.ClientConfig()
	if err != nil {
		if strings.Contains(err.Error(), "context") && strings.Contains(err.Error(), "does not exist") {
			return nil, err
		}
		if strings.Contains(err.Error(), "cluster") && strings.Contains(err.Error(), "does not exist") {
			return nil, err
		}
		if strings.Contains(err.Error(), "user") && strings.Contains(err.Error(), "does not exist") {
			return nil, err
		}
		return nil, err
	}

	// Apply InsecureSkipVerify to rest.Config
	if ka.InsecureSkipVerify {
		config.TLSClientConfig.Insecure = true
		config.TLSClientConfig.CAData = nil
		config.TLSClientConfig.CAFile = ""
	}

	return config, nil
}
