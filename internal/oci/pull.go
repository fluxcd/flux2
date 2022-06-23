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

	"github.com/fluxcd/pkg/untar"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
)

// Pull downloads an artifact from an OCI repository and extracts the content to the given directory.
func Pull(ctx context.Context, url, outDir string) (*Metadata, error) {
	ref, err := name.ParseReference(url)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	img, err := crane.Pull(url, craneOptions(ctx)...)
	if err != nil {
		return nil, err
	}

	digest, err := img.Digest()
	if err != nil {
		return nil, fmt.Errorf("parsing digest failed: %w", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("parsing manifest failed: %w", err)
	}

	meta, err := MetadataFromAnnotations(manifest.Annotations)
	if err != nil {
		return nil, err
	}
	meta.Digest = ref.Context().Digest(digest.String()).String()

	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("failed to list layers: %w", err)
	}

	if len(layers) < 1 {
		return nil, fmt.Errorf("no layers found in artifact")
	}

	blob, err := layers[0].Compressed()
	if err != nil {
		return nil, fmt.Errorf("extracting first layer failed: %w", err)
	}

	if _, err = untar.Untar(blob, outDir); err != nil {
		return nil, fmt.Errorf("failed to untar first layer: %w", err)
	}

	return meta, nil
}
