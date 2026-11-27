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

package install

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestVersion(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr bool
	}{
		{name: "latest release", status: http.StatusOK, body: `{"tag_name":"v2.10.1"}`, want: "v2.10.1"},
		{name: "rate limited", status: http.StatusForbidden, body: `{"message":"API rate limit exceeded"}`, wantErr: true},
		{name: "missing tag", status: http.StatusOK, body: `{}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			got, err := latestVersion(srv.Client(), srv.URL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got %q, err %v, wantErr %v", got, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
