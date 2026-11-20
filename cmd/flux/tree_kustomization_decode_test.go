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
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func gzipBase64(t *testing.T, data []byte) string {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestDecodeHelmReleaseStorage(t *testing.T) {
	const manifest = "apiVersion: v1\nkind: ConfigMap\n"
	releaseJSON, err := json.Marshal(hrStorage{Name: "test", Manifest: manifest})
	if err != nil {
		t.Fatalf("marshal release: %v", err)
	}

	tests := []struct {
		name         string
		releaseData  string
		wantErr      bool
		wantManifest string
	}{
		{
			name:         "plain json release",
			releaseData:  base64.StdEncoding.EncodeToString(releaseJSON),
			wantManifest: manifest,
		},
		{
			name:         "gzipped json release",
			releaseData:  gzipBase64(t, releaseJSON),
			wantManifest: manifest,
		},
		{
			// Decodes to the two bytes "{}", which is shorter than the
			// gzip magic header. The unguarded b[0:3] slice used to panic.
			name:        "payload shorter than gzip header",
			releaseData: base64.StdEncoding.EncodeToString([]byte("{}")),
		},
		{
			// Decodes to an empty payload, also shorter than the header.
			name:        "empty payload",
			releaseData: "",
			wantErr:     true,
		},
		{
			name:        "invalid base64",
			releaseData: "not-base64!!",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rls, err := decodeHelmReleaseStorage([]byte(tt.releaseData))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rls.Manifest != tt.wantManifest {
				t.Fatalf("manifest mismatch: got %q, want %q", rls.Manifest, tt.wantManifest)
			}
		})
	}
}
