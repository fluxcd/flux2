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

// GitProvider holds a Git provider definition.
type GitProvider string

const (
	GitProviderGitHub GitProvider = "github"
	GitProviderGitea  GitProvider = "gitea"
	GitProviderGitLab GitProvider = "gitlab"
	GitProviderStash  GitProvider = "stash"
)

// Config defines the configuration for connecting to a GitProvider.
type Config struct {
	// Provider defines the GitProvider.
	Provider GitProvider

	// Hostname is the HTTP/S hostname of the Provider,
	// e.g. github.example.com.
	Hostname string

	// Username contains the username used to authenticate with
	// the Provider.
	Username string

	// Token contains the token used to authenticate with the
	// Provider.
	Token string

	// CABunle contains the CA bundle to use for the client.
	CaBundle []byte
}
