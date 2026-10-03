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
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func withCustomCATransportHook(caBundle []byte) gitprovider.ClientOption {
	return gitprovider.WithPostChainTransportHook(func(_ http.RoundTripper) http.RoundTripper {
		transport := http.DefaultTransport.(*http.Transport).Clone()

		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}
		rootCAs.AppendCertsFromPEM(caBundle)

		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.RootCAs = rootCAs

		return transport
	})
}
