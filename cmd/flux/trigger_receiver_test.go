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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
)

// resetTriggerReceiverArgs restores the package-global flags to their defaults
// so tests do not leak state into each other.
func resetTriggerReceiverArgs(t *testing.T) {
	t.Helper()
	prev := triggerReceiverArgs
	prevNS := kubeconfigArgs.Namespace
	prevTimeout := rootArgs.timeout

	triggerReceiverArgs = triggerReceiverFlags{
		receiverType: notificationv1.GenericReceiver,
		payload:      "{}",
	}
	ns := "default"
	kubeconfigArgs.Namespace = &ns
	rootArgs.timeout = time.Minute

	t.Cleanup(func() {
		triggerReceiverArgs = prev
		kubeconfigArgs.Namespace = prevNS
		rootArgs.timeout = prevTimeout
	})
}

func TestValidateTriggerReceiverArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    triggerReceiverFlags
		wantErr string
	}{
		{
			name:    "generic requires token",
			args:    triggerReceiverFlags{receiverType: notificationv1.GenericReceiver},
			wantErr: "--token is required",
		},
		{
			name: "generic with token is valid",
			args: triggerReceiverFlags{receiverType: notificationv1.GenericReceiver, token: "t"},
		},
		{
			name:    "generic rejects oidc flags",
			args:    triggerReceiverFlags{receiverType: notificationv1.GenericReceiver, token: "t", oidcProvider: "github"},
			wantErr: "can only be set for --type=generic-oidc",
		},
		{
			name: "hmac with token is valid",
			args: triggerReceiverFlags{receiverType: notificationv1.GenericHMACReceiver, token: "t"},
		},
		{
			name:    "unknown type",
			args:    triggerReceiverFlags{receiverType: "bogus", token: "t"},
			wantErr: "invalid --type",
		},
		{
			name:    "oidc rejects token",
			args:    triggerReceiverFlags{receiverType: genericOIDCReceiver, token: "t"},
			wantErr: "--token must not be set",
		},
		{
			name:    "oidc provider and token mutually exclusive",
			args:    triggerReceiverFlags{receiverType: genericOIDCReceiver, oidcProvider: "github", oidcToken: "x"},
			wantErr: "mutually exclusive",
		},
		{
			name:    "oidc invalid provider",
			args:    triggerReceiverFlags{receiverType: genericOIDCReceiver, oidcProvider: "gitlab"},
			wantErr: "invalid --oidc-provider",
		},
		{
			name:    "oidc audience requires provider",
			args:    triggerReceiverFlags{receiverType: genericOIDCReceiver, oidcToken: "x", oidcAudience: "aud"},
			wantErr: "--oidc-audience can only be set together with --oidc-provider",
		},
		{
			name: "oidc with provider is valid",
			args: triggerReceiverFlags{receiverType: genericOIDCReceiver, oidcProvider: "forgejo", oidcAudience: "aud"},
		},
		{
			name: "oidc with token is valid",
			args: triggerReceiverFlags{receiverType: genericOIDCReceiver, oidcToken: "x"},
		},
		{
			name: "oidc without provider or token is valid (env fallback)",
			args: triggerReceiverFlags{receiverType: genericOIDCReceiver},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTriggerReceiverArgs(t)
			triggerReceiverArgs = tt.args

			err := validateTriggerReceiverArgs()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestTriggerReceiverRun(t *testing.T) {
	const name = "my-receiver"
	const ns = "default"
	const token = "my-token"

	expectedPath := (&notificationv1.Receiver{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
	}).GetWebhookPath(token)
	expectedOIDCPath := (&notificationv1.Receiver{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
	}).GetWebhookPath("")

	t.Run("generic sends payload with default headers", func(t *testing.T) {
		resetTriggerReceiverArgs(t)
		var got *http.Request
		var gotBody string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = r
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		triggerReceiverArgs.url = srv.URL
		triggerReceiverArgs.token = token
		triggerReceiverArgs.payload = `{"hello":"world"}`

		if err := triggerReceiverCmdRun(nil, []string{name}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", got.URL.Path, expectedPath)
		}
		if got.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", got.Method)
		}
		if ct := got.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if ua := got.Header.Get("User-Agent"); !strings.HasPrefix(ua, "flux/v") {
			t.Errorf("User-Agent = %q, want prefix flux/v", ua)
		}
		if gotBody != `{"hello":"world"}` {
			t.Errorf("body = %q", gotBody)
		}
	})

	t.Run("generic-hmac sets X-Signature", func(t *testing.T) {
		resetTriggerReceiverArgs(t)
		var sig string
		payload := `{"a":1}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sig = r.Header.Get("X-Signature")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		triggerReceiverArgs.url = srv.URL
		triggerReceiverArgs.token = token
		triggerReceiverArgs.receiverType = notificationv1.GenericHMACReceiver
		triggerReceiverArgs.payload = payload

		if err := triggerReceiverCmdRun(nil, []string{name}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mac := hmac.New(sha256.New, []byte(token))
		mac.Write([]byte(payload))
		want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if sig != want {
			t.Errorf("X-Signature = %q, want %q", sig, want)
		}
	})

	t.Run("generic-oidc with --oidc-token sets bearer and empty-token path", func(t *testing.T) {
		resetTriggerReceiverArgs(t)
		var auth, path string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth = r.Header.Get("Authorization")
			path = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		triggerReceiverArgs.url = srv.URL
		triggerReceiverArgs.receiverType = genericOIDCReceiver
		triggerReceiverArgs.oidcToken = "the-oidc-token"

		if err := triggerReceiverCmdRun(nil, []string{name}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if auth != "Bearer the-oidc-token" {
			t.Errorf("Authorization = %q, want Bearer the-oidc-token", auth)
		}
		if path != expectedOIDCPath {
			t.Errorf("path = %q, want %q (empty token salt)", path, expectedOIDCPath)
		}
	})

	t.Run("generic-oidc reads default env var", func(t *testing.T) {
		resetTriggerReceiverArgs(t)
		t.Setenv(defaultOIDCTokenEnvVar, "env-oidc-token")
		var auth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		triggerReceiverArgs.url = srv.URL
		triggerReceiverArgs.receiverType = genericOIDCReceiver

		if err := triggerReceiverCmdRun(nil, []string{name}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if auth != "Bearer env-oidc-token" {
			t.Errorf("Authorization = %q, want Bearer env-oidc-token", auth)
		}
	})

	t.Run("non-2xx response is an error", func(t *testing.T) {
		resetTriggerReceiverArgs(t)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("nope"))
		}))
		defer srv.Close()

		triggerReceiverArgs.url = srv.URL
		triggerReceiverArgs.token = token

		err := triggerReceiverCmdRun(nil, []string{name})
		if err == nil || !strings.Contains(err.Error(), "nope") {
			t.Fatalf("expected error containing response body, got: %v", err)
		}
	})

	t.Run("retries on retryable status then succeeds", func(t *testing.T) {
		resetTriggerReceiverArgs(t)
		var attempts int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt32(&attempts, 1) < 3 {
				w.WriteHeader(http.StatusNotFound) // transient: path not registered yet
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		triggerReceiverArgs.url = srv.URL
		triggerReceiverArgs.token = token
		triggerReceiverArgs.retries = 5
		triggerReceiverArgs.retryDelay = time.Millisecond

		if err := triggerReceiverCmdRun(nil, []string{name}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := atomic.LoadInt32(&attempts); got != 3 {
			t.Errorf("attempts = %d, want 3", got)
		}
	})

	t.Run("does not retry non-retryable status", func(t *testing.T) {
		resetTriggerReceiverArgs(t)
		var attempts int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()

		triggerReceiverArgs.url = srv.URL
		triggerReceiverArgs.token = token
		triggerReceiverArgs.retries = 5
		triggerReceiverArgs.retryDelay = time.Millisecond

		if err := triggerReceiverCmdRun(nil, []string{name}); err == nil {
			t.Fatal("expected error")
		}
		if got := atomic.LoadInt32(&attempts); got != 1 {
			t.Errorf("attempts = %d, want 1 (no retry on 403)", got)
		}
	})

	t.Run("returns error after exhausting retries", func(t *testing.T) {
		resetTriggerReceiverArgs(t)
		var attempts int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		triggerReceiverArgs.url = srv.URL
		triggerReceiverArgs.token = token
		triggerReceiverArgs.retries = 2
		triggerReceiverArgs.retryDelay = time.Millisecond

		if err := triggerReceiverCmdRun(nil, []string{name}); err == nil {
			t.Fatal("expected error after exhausting retries")
		}
		if got := atomic.LoadInt32(&attempts); got != 3 {
			t.Errorf("attempts = %d, want 3 (1 initial + 2 retries)", got)
		}
	})
}
