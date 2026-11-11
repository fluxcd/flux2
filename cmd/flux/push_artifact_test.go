//go:build unit
// +build unit

/*
Copyright 2026 The Flux authors

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
	"bytes"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/gomega"

	"github.com/fluxcd/pkg/oci"
)

func TestPushArtifactReproducibleCreatedIsUTC(t *testing.T) {
	g := NewWithT(t)

	// TZ is read only once per process, so switch the local zone directly.
	local := time.Local
	time.Local = time.FixedZone("UTC+9", 9*60*60)
	t.Cleanup(func() {
		time.Local = local
		pushArtifactArgs.reproducible = false
	})

	g.Expect(setupRegistryServer(t.Context())).To(Succeed())

	ref := dockerReg + "/reproducible:v1"
	_, err := executeCommand("push artifact oci://" + ref +
		" --path=./testdata/diff-artifact/deployment.yaml --source=test --revision=test --reproducible")
	g.Expect(err).ToNot(HaveOccurred())

	raw, err := crane.Manifest(ref)
	g.Expect(err).ToNot(HaveOccurred())
	manifest, err := gcrv1.ParseManifest(bytes.NewReader(raw))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(manifest.Annotations).To(HaveKeyWithValue(oci.CreatedAnnotation, "1970-01-01T00:00:00Z"))
}

func TestPushArtifactAnnotationValueWithEquals(t *testing.T) {
	g := NewWithT(t)
	t.Cleanup(func() { pushArtifactArgs.annotations = nil })

	g.Expect(setupRegistryServer(t.Context())).To(Succeed())

	ref := dockerReg + "/annotations:v1"
	_, err := executeCommand("push artifact oci://" + ref +
		" --path=./testdata/diff-artifact/deployment.yaml --source=test --revision=test" +
		" --annotations=org.opencontainers.image.documentation=https://app.org/docs?page=1" +
		" --annotations=org.opencontainers.image.description=")
	g.Expect(err).ToNot(HaveOccurred())

	raw, err := crane.Manifest(ref)
	g.Expect(err).ToNot(HaveOccurred())
	manifest, err := gcrv1.ParseManifest(bytes.NewReader(raw))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(manifest.Annotations).To(HaveKeyWithValue("org.opencontainers.image.documentation", "https://app.org/docs?page=1"))
	g.Expect(manifest.Annotations).To(HaveKeyWithValue("org.opencontainers.image.description", ""))
}

func TestPushArtifactInvalidAnnotation(t *testing.T) {
	g := NewWithT(t)
	t.Cleanup(func() { pushArtifactArgs.annotations = nil })

	g.Expect(setupRegistryServer(t.Context())).To(Succeed())

	_, err := executeCommand("push artifact oci://" + dockerReg + "/annotations:invalid" +
		" --path=./testdata/diff-artifact/deployment.yaml --source=test --revision=test" +
		" --annotations=no-separator")
	g.Expect(err).To(MatchError(ContainSubstring("invalid annotation no-separator")))
}
