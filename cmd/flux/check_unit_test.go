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
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

func Test_checkListError(t *testing.T) {
	tests := []struct {
		name    string
		check   func(context.Context, client.Client) bool
		wantLog string
	}{
		{
			name:    "controllers",
			check:   componentsCheck,
			wantLog: "failed to list controllers",
		},
		{
			name:    "crds",
			check:   crdsCheck,
			wantLog: "failed to list crds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			var testLogger bytes.Buffer
			oldLogger := logger
			logger = stderrLogger{&testLogger}
			t.Cleanup(func() { logger = oldLogger })

			kubeClient := fake.NewClientBuilder().
				WithScheme(utils.NewScheme()).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
						return errors.New("forbidden")
					},
				}).
				Build()

			g.Expect(tt.check(context.Background(), kubeClient)).To(BeFalse())
			g.Expect(testLogger.String()).To(ContainSubstring(tt.wantLog))
			g.Expect(testLogger.String()).To(ContainSubstring("forbidden"))
		})
	}
}
