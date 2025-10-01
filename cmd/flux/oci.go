/*
Copyright 2025 The Flux authors

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
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"

	"github.com/fluxcd/pkg/auth"
	"github.com/fluxcd/pkg/auth/azure"
	authutils "github.com/fluxcd/pkg/auth/utils"
)

// loginWithProvider gets a crane authentication option for the given provider and URL.
func loginWithProvider(ctx context.Context, url, provider string) (crane.Option, authn.Authenticator, error) {
	var opts []auth.Option
	if provider == azure.ProviderName {
		opts = append(opts, auth.WithAllowShellOut())
	}
	authenticator, err := authutils.GetArtifactRegistryCredentials(ctx, provider, url, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("could not login to provider %s with url %s: %w", provider, url, err)
	}
	return crane.WithAuth(authenticator), authenticator, nil
}
