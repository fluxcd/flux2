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

package provider

import (
	"net/http"
	"testing"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func TestWithCustomCATransportHookPreservesProxy(t *testing.T) {
	opts, err := gitprovider.MakeClientOptions(withCustomCATransportHook(testCAPEM))
	if err != nil {
		t.Fatal(err)
	}

	client, err := gitprovider.BuildClientFromTransportChain(opts.GetTransportChain())
	if err != nil {
		t.Fatal(err)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("got transport type %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatal("expected proxy configuration to be preserved")
	}
	if transport.TLSClientConfig == nil || transport.TLSClientConfig.RootCAs == nil {
		t.Fatal("expected custom CA bundle to be configured")
	}
}

var testCAPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJALRiMLAh4HMHMA0GCSqGSIb3DQEBCwUAMBExDzANBgNVBAMMBnRl
c3RjYTAeFw0yNDA0MDQwMDAwMDBaFw0zNDA0MDIwMDAwMDBaMBExDzANBgNVBAMM
BnRlc3RjYTBcMA0GCSqGSIb3DQEBAQUAA0sAMEgCQQC7o96+IG5sKBe0QKbsBigc
GsR8cKQuDfhCFqzWn7zr4aqHsLQiKEJsClMDGnNHEFGDFpXuIFxnGOTPYFOYIuDH
AgMBAAGjUzBRMB0GA1UdDgQWBBQgTxe0MCRKYB0ILQM0L7V/lMjxNjAfBgNVHSME
GDAWgBQgTxe0MCRKYB0ILQM0L7V/lMjxNjAPBgNVHRMBAf8EBTADAQH/MA0GCSqG
SIb3DQEBCwUAA0EAh/8fnFa6VW1cB8QJWIM4KpCmpY9R1YMaqGCbDjM0FZmE+dqA
NsaKMCSE1YOIMBN6mBUX3iTmy/sCTIYMBbFPgQ==
-----END CERTIFICATE-----
`)
