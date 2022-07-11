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
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
)

// List fetches the tags and their manifests for a given OCI repository.
func List(ctx context.Context, url string) ([]Metadata, error) {
	metas := make([]Metadata, 0)
	tags, err := crane.ListTags(url, craneOptions(ctx)...)
	if err != nil {
		return nil, fmt.Errorf("listing tags failed: %w", err)
	}

	sort.Slice(tags, func(i, j int) bool { return tags[i] > tags[j] })

	for _, tag := range tags {
		// exclude cosign signatures
		if strings.HasSuffix(tag, ".sig") {
			continue
		}

		meta := Metadata{
			URL: fmt.Sprintf("%s:%s", url, tag),
		}

		manifestJSON, err := crane.Manifest(meta.URL, craneOptions(ctx)...)
		if err != nil {
			return nil, fmt.Errorf("fetching manifest failed: %w", err)
		}

		manifest, err := gcrv1.ParseManifest(bytes.NewReader(manifestJSON))
		if err != nil {
			return nil, fmt.Errorf("parsing manifest failed: %w", err)
		}

		if m, err := MetadataFromAnnotations(manifest.Annotations); err == nil {
			meta.Revision = m.Revision
			meta.Source = m.Source
		}

		digest, err := crane.Digest(meta.URL, craneOptions(ctx)...)
		if err != nil {
			return nil, fmt.Errorf("fetching digest failed: %w", err)
		}
		meta.Digest = digest

		metas = append(metas, meta)
	}

	return metas, nil
}
