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
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
)

// Push creates an artifact from the given directory, uploads the artifact
// to the given OCI repository and returns the digest.
func Push(ctx context.Context, url, sourceDir string, meta Metadata) (string, error) {
	ref, err := name.ParseReference(url)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "oci")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "artifact.tgz")

	if err := Build(tmpFile, sourceDir); err != nil {
		return "", err
	}

	img, err := crane.Append(empty.Image, tmpFile)
	if err != nil {
		return "", fmt.Errorf("appeding content to artifact failed: %w", err)
	}

	img = mutate.Annotations(img, meta.ToAnnotations()).(gcrv1.Image)

	if err := crane.Push(img, url, craneOptions(ctx)...); err != nil {
		return "", fmt.Errorf("pushing artifact failed: %w", err)
	}

	digest, err := img.Digest()
	if err != nil {
		return "", fmt.Errorf("parsing artifact digest failed: %w", err)
	}

	return ref.Context().Digest(digest.String()).String(), err
}

func craneOptions(ctx context.Context) []crane.Option {
	return []crane.Option{
		crane.WithContext(ctx),
		crane.WithUserAgent("flux/v2"),
		crane.WithPlatform(&gcrv1.Platform{
			Architecture: "flux",
			OS:           "flux",
			OSVersion:    "v2",
		}),
	}
}
