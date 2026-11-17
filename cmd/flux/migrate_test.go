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
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

type writeToMemoryFS struct {
	fs.FS

	writtenFiles map[string][]byte
}

func (m *writeToMemoryFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	m.writtenFiles[name] = data
	return nil
}

type writtenFile struct {
	file       string
	goldenFile string
}

func TestFileSystemMigrator(t *testing.T) {
	for _, tt := range []struct {
		name         string
		path         string
		outputGolden string
		writtenFiles []writtenFile
		err          string
	}{
		{
			name: "errors out for single file that is a symlink",
			path: "testdata/migrate/file-system/single-file-link.yaml",
			err:  "file testdata/migrate/file-system/single-file-link.yaml is irregular",
		},
		{
			name: "errors out for single file with wrong extension",
			path: "testdata/migrate/file-system/single-file-wrong-ext.json",
			err:  "file testdata/migrate/file-system/single-file-wrong-ext.json does not match the specified extensions: .yaml, .yml",
		},
		{
			name:         "migrate single file",
			path:         "testdata/migrate/file-system/single-file.yaml",
			outputGolden: "testdata/migrate/file-system/single-file.yaml.output.golden",
			writtenFiles: []writtenFile{
				{
					file:       "testdata/migrate/file-system/single-file.yaml",
					goldenFile: "testdata/migrate/file-system/single-file.yaml.golden",
				},
			},
		},
		{
			name:         "migrate file removing spec fields not supported in the new version",
			path:         "testdata/migrate/file-system/removed-fields.yaml",
			outputGolden: "testdata/migrate/file-system/removed-fields.yaml.output.golden",
			writtenFiles: []writtenFile{
				{
					file:       "testdata/migrate/file-system/removed-fields.yaml",
					goldenFile: "testdata/migrate/file-system/removed-fields.yaml.golden",
				},
			},
		},
		{
			name:         "migrate files in directory",
			path:         "testdata/migrate/file-system/dir",
			outputGolden: "testdata/migrate/file-system/dir.output.golden",
			writtenFiles: []writtenFile{
				{
					file:       "testdata/migrate/file-system/dir/some-dir/another-file.yaml",
					goldenFile: "testdata/migrate/file-system/dir.golden/some-dir/another-file.yaml",
				},
				{
					file:       "testdata/migrate/file-system/dir/some-dir/another-file.yml",
					goldenFile: "testdata/migrate/file-system/dir.golden/some-dir/another-file.yml",
				},
				{
					file:       "testdata/migrate/file-system/dir/some-file.yaml",
					goldenFile: "testdata/migrate/file-system/dir.golden/some-file.yaml",
				},
				{
					file:       "testdata/migrate/file-system/dir/some-file.yml",
					goldenFile: "testdata/migrate/file-system/dir.golden/some-file.yml",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Store logger, replace with test logger, and restore at the end of the test.
			var testLogger bytes.Buffer
			oldLogger := logger
			logger = stderrLogger{&testLogger}
			t.Cleanup(func() { logger = oldLogger })

			// Open current working directory as root and build write-to-memory filesystem.
			pathRoot, err := os.OpenRoot(".")
			g.Expect(err).ToNot(HaveOccurred())
			t.Cleanup(func() { pathRoot.Close() })
			fileSystem := &writeToMemoryFS{
				FS:           pathRoot.FS(),
				writtenFiles: make(map[string][]byte),
			}

			// Prepare other inputs.
			const yes = true
			const dryRun = false
			extensions := []string{".yaml", ".yml"}
			latestVersions := map[schema.GroupKind]string{
				{Group: "image.toolkit.fluxcd.io", Kind: "ImageRepository"}:       "v1",
				{Group: "image.toolkit.fluxcd.io", Kind: "ImagePolicy"}:           "v1",
				{Group: "image.toolkit.fluxcd.io", Kind: "ImageUpdateAutomation"}: "v1",
				{Group: "notification.toolkit.fluxcd.io", Kind: "Alert"}:          "v1",
				{Group: "notification.toolkit.fluxcd.io", Kind: "Provider"}:       "v1",
			}

			// Run migration.
			err = NewFileSystemMigrator(fileSystem, yes, dryRun, tt.path, extensions, latestVersions).Run()
			if tt.err != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(Equal(tt.err))
				return
			}
			g.Expect(err).ToNot(HaveOccurred())

			// Assert logger output.
			b, err := os.ReadFile(tt.outputGolden)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(string(b)).To(Equal(testLogger.String()),
				"logger output does not match golden file %s", tt.outputGolden)

			// Assert which files were written.
			writtenFiles := make([]string, 0, len(fileSystem.writtenFiles))
			for name := range fileSystem.writtenFiles {
				writtenFiles = append(writtenFiles, name)
			}
			expectedWrittenFiles := make([]string, 0, len(tt.writtenFiles))
			for _, wf := range tt.writtenFiles {
				expectedWrittenFiles = append(expectedWrittenFiles, wf.file)
			}
			g.Expect(writtenFiles).To(ConsistOf(expectedWrittenFiles))

			// Assert contents of written files.
			for _, wf := range tt.writtenFiles {
				b, err := os.ReadFile(wf.goldenFile)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(string(fileSystem.writtenFiles[wf.file])).To(Equal(string(b)),
					"file %s does not match golden file %s", wf.file, wf.goldenFile)
			}
		})
	}
}

func TestLatestAPIVersionsNotification(t *testing.T) {
	alert := schema.GroupKind{Group: "notification.toolkit.fluxcd.io", Kind: "Alert"}
	provider := schema.GroupKind{Group: "notification.toolkit.fluxcd.io", Kind: "Provider"}

	want := map[string]string{
		"2.10": "v1",
		"2.9":  "v1beta3",
		"2.6":  "v1beta3",
	}

	g := NewWithT(t)
	g.Expect(latestAPIVersions[0].FluxVersion).To(Equal("2.10"))
	for _, v := range latestAPIVersions {
		version, ok := want[v.FluxVersion]
		if !ok {
			continue
		}
		g.Expect(v.LatestVersions[alert]).To(Equal(version), "Alert for Flux %s", v.FluxVersion)
		g.Expect(v.LatestVersions[provider]).To(Equal(version), "Provider for Flux %s", v.FluxVersion)
	}
}

func TestFileSystemMigratorRemovedFields(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "CRLF line endings",
			input: "apiVersion: notification.toolkit.fluxcd.io/v1beta3\r\nkind: Alert\r\nspec:\r\n" +
				"  summary: legacy\r\n  providerRef:\r\n    name: slack\r\n",
			expected: "apiVersion: notification.toolkit.fluxcd.io/v1\r\nkind: Alert\r\nspec:\r\n" +
				"  providerRef:\r\n    name: slack\r\n",
		},
		{
			name: "document separator followed by tab and comment",
			input: `apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Alert
metadata:
  name: alert
---	# next document
apiVersion: example.com/v1
kind: Other
spec:
  summary: keep
`,
			expected: `apiVersion: notification.toolkit.fluxcd.io/v1
kind: Alert
metadata:
  name: alert
---	# next document
apiVersion: example.com/v1
kind: Other
spec:
  summary: keep
`,
		},
		{
			name: "multi-line quoted scalars",
			input: `apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Alert
spec:
  summary: "hello \"
world"
  providerRef:
    name: slack
---
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Provider
spec:
  proxy: 'http://it''s
a proxy'
  type: generic
`,
			expected: `apiVersion: notification.toolkit.fluxcd.io/v1
kind: Alert
spec:
  providerRef:
    name: slack
---
apiVersion: notification.toolkit.fluxcd.io/v1
kind: Provider
spec:
  type: generic
`,
		},
		{
			name: "object embedded in a block scalar",
			input: `apiVersion: v1
kind: ConfigMap
data:
  example.txt: |
    apiVersion: notification.toolkit.fluxcd.io/v1beta3
    kind: Alert
    spec:
      summary: keep this text
`,
			expected: `apiVersion: v1
kind: ConfigMap
data:
  example.txt: |
    apiVersion: notification.toolkit.fluxcd.io/v1
    kind: Alert
    spec:
      summary: keep this text
`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			var testLogger bytes.Buffer
			oldLogger := logger
			logger = stderrLogger{&testLogger}
			t.Cleanup(func() { logger = oldLogger })

			const file = "manifest.yaml"
			fileSystem := &writeToMemoryFS{
				FS:           fstest.MapFS{file: &fstest.MapFile{Data: []byte(tt.input)}},
				writtenFiles: make(map[string][]byte),
			}
			latestVersions := map[schema.GroupKind]string{
				{Group: "notification.toolkit.fluxcd.io", Kind: "Alert"}:    "v1",
				{Group: "notification.toolkit.fluxcd.io", Kind: "Provider"}: "v1",
			}

			err := NewFileSystemMigrator(fileSystem, true, false, file, []string{".yaml"}, latestVersions).Run()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(string(fileSystem.writtenFiles[file])).To(Equal(tt.expected))
		})
	}
}

// patchRecorderClient wraps a client.Client and records the names of the patched objects.
type patchRecorderClient struct {
	client.Client
	patched []string
}

func (c *patchRecorderClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	c.patched = append(c.patched, obj.GetName())
	return c.Client.Patch(ctx, obj, patch, opts...)
}

func TestClusterMigrator_migrateCR_pagination(t *testing.T) {
	g := NewWithT(t)

	const total = 5
	builder := fake.NewClientBuilder().WithScheme(utils.NewScheme())
	names := make([]string, 0, total)
	for i := range total {
		name := fmt.Sprintf("test-%d", i)
		names = append(names, name)
		builder = builder.WithObjects(&kustomizev1.Kustomization{
			Name: name, Namespace: "default",
		})
	}

	// Paginate at 2 items per page, forcing multiple List round-trips.
	recorder := &patchRecorderClient{
		Client: &paginatedClient{Client: builder.Build(), pageSize: 2},
	}

	crd := &apiextensionsv1.CustomResourceDefinition{
		Name: "kustomizations.kustomize.toolkit.fluxcd.io",
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: kustomizev1.GroupVersion.Group,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				ListKind: kustomizev1.KustomizationKind + "List",
			},
		},
	}

	migrator := NewClusterMigrator(recorder, nil)
	err := migrator.migrateCR(context.Background(), crd, kustomizev1.GroupVersion.Version)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(recorder.patched).To(ConsistOf(names),
		"migrateCR should patch all resources across paginated responses")
}
