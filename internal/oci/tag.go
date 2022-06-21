/*
Copyright 2022 The Flux authors

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

package oci

import (
	"context"
	"fmt"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
)

// Tag creates a new tag for the given artifact using the same OCI repository as the origin.
func Tag(ctx context.Context, url, tag string) (string, error) {
	ref, err := name.ParseReference(url)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if err := crane.Tag(url, tag, craneOptions(ctx)...); err != nil {
		return "", err
	}

	dst := ref.Context().Tag(tag)

	return dst.Name(), nil
}
