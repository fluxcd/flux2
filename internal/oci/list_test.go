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
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	. "github.com/onsi/gomega"
)

func Test_List(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	repo := "test-list" + randStringRunes(5)
	tags := []string{"v0.0.1", "v0.0.2", "v0.0.3"}
	source := "github.com/fluxcd/fluxv2"
	rev := "rev"
	m := Metadata{
		Source:   source,
		Revision: rev,
	}

	for _, tag := range tags {
		dst := fmt.Sprintf("%s/%s:%s", dockerReg, repo, tag)
		img, err := random.Image(1024, 1)
		g.Expect(err).ToNot(HaveOccurred())
		img = mutate.Annotations(img, m.ToAnnotations()).(gcrv1.Image)
		err = crane.Push(img, dst, craneOptions(ctx)...)
		g.Expect(err).ToNot(HaveOccurred())
	}

	metadata, err := List(ctx, fmt.Sprintf("%s/%s", dockerReg, repo))
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(len(metadata)).To(Equal(len(tags)))
	for _, meta := range metadata {
		tag, err := name.NewTag(meta.URL)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(tag.TagStr()).Should(BeElementOf(tags))

		g.Expect(meta.ToAnnotations()).To(Equal(m.ToAnnotations()))

		digest, err := crane.Digest(meta.URL, craneOptions(ctx)...)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(meta.Digest).To(Equal(digest))
	}
}
