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

package provider

import (
	"fmt"

	"github.com/fluxcd/go-git-providers/gitea"
	"github.com/fluxcd/go-git-providers/github"
	"github.com/fluxcd/go-git-providers/gitlab"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/stash"
)

// BuildGitProvider builds a gitprovider.Client for the provided
// Config. It returns an error if the Config.Provider
// is not supported, or if the construction of the client fails.
func BuildGitProvider(config Config) (gitprovider.Client, error) {
	var client gitprovider.Client
	var err error
	switch config.Provider {
	case GitProviderGitHub:
		opts := []gitprovider.ClientOption{
			gitprovider.WithOAuth2Token(config.Token),
		}
		if config.Hostname != "" {
			opts = append(opts, gitprovider.WithDomain(config.Hostname))
		}
		if config.CaBundle != nil {
			opts = append(opts, gitprovider.WithCustomCAPostChainTransportHook(config.CaBundle))
		}
		if client, err = github.NewClient(opts...); err != nil {
			return nil, err
		}
	case GitProviderGitea:
		opts := []gitprovider.ClientOption{}
		if config.Hostname != "" {
			opts = append(opts, gitprovider.WithDomain(config.Hostname))
		}
		if config.CaBundle != nil {
			opts = append(opts, gitprovider.WithCustomCAPostChainTransportHook(config.CaBundle))
		}
		if client, err = gitea.NewClient(config.Token, opts...); err != nil {
			return nil, err
		}
	case GitProviderGitLab:
		opts := []gitprovider.ClientOption{
			gitprovider.WithConditionalRequests(true),
		}
		if config.Hostname != "" {
			opts = append(opts, gitprovider.WithDomain(config.Hostname))
		}
		if config.CaBundle != nil {
			opts = append(opts, gitprovider.WithCustomCAPostChainTransportHook(config.CaBundle))
		}
		if client, err = gitlab.NewClient(config.Token, "", opts...); err != nil {
			return nil, err
		}
	case GitProviderStash:
		opts := []gitprovider.ClientOption{}
		if config.Hostname != "" {
			opts = append(opts, gitprovider.WithDomain(config.Hostname))
		}
		if config.CaBundle != nil {
			opts = append(opts, gitprovider.WithCustomCAPostChainTransportHook(config.CaBundle))
		}
		if client, err = stash.NewStashClient(config.Username, config.Token, opts...); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported Git provider '%s'", config.Provider)
	}
	return client, err
}
