//go:build unit
// +build unit

/*
Copyright 2021 The Flux authors

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
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func TestLogsNoArgs(t *testing.T) {
	cmd := cmdTestCase{
		args:   "logs",
		assert: assertSuccess(),
	}
	cmd.runTestCmd(t)
}

func TestLogsAllNamespaces(t *testing.T) {
	cmd := cmdTestCase{
		args:   "logs --all-namespaces",
		assert: assertSuccess(),
	}
	cmd.runTestCmd(t)
}

func TestLogsSince(t *testing.T) {
	cmd := cmdTestCase{
		args:   "logs --since=2m",
		assert: assertSuccess(),
	}
	cmd.runTestCmd(t)
}

func TestLogsSinceInvalid(t *testing.T) {
	cmd := cmdTestCase{
		args:   "logs --since=XXX",
		assert: assertError(`invalid argument "XXX" for "--since" flag: time: invalid duration "XXX"`),
	}
	cmd.runTestCmd(t)
}

func TestLogsSinceTime(t *testing.T) {
	cmd := cmdTestCase{
		args:   "logs --since-time=2021-08-06T14:26:25.546Z",
		assert: assertSuccess(),
	}
	cmd.runTestCmd(t)
}

func TestLogsSinceTimeInvalid(t *testing.T) {
	cmd := cmdTestCase{
		args:   "logs --since-time=XXX",
		assert: assertError("XXX is not a valid (RFC3339) time"),
	}
	cmd.runTestCmd(t)
}

func TestLogsSinceOnlyOneAllowed(t *testing.T) {
	cmd := cmdTestCase{
		args:   "logs --since=2m --since-time=2021-08-06T14:26:25.546Z",
		assert: assertError("at most one of `sinceTime` or `sinceSeconds` may be specified"),
	}
	cmd.runTestCmd(t)
}

var testPodLogs = `{"level":"info","ts":"2022-08-02T12:55:34.419Z","logger":"controller.gitrepository","msg":"no changes since last reconcilation: observed revision","reconciler group":"source.toolkit.fluxcd.io","reconciler kind":"GitRepository","name":"podinfo","namespace":"default"}
{"level":"error","ts":"2022-08-02T12:56:04.679Z","logger":"controller.gitrepository","msg":"no changes since last reconcilation: observed revision","reconciler group":"source.toolkit.fluxcd.io","reconciler kind":"GitRepository","name":"flux-system","namespace":"flux-system"}
{"level":"error","ts":"2022-08-02T12:56:34.961Z","logger":"controller.kustomization","msg":"no changes since last reconcilation: observed revision","reconciler group":"kustomize.toolkit.fluxcd.io","reconciler kind":"Kustomization","name":"flux-system","namespace":"flux-system"}
{"level":"info","ts":"2022-08-02T12:56:34.961Z","logger":"controller.kustomization","msg":"no changes since last reconcilation: observed revision","reconciler group":"kustomize.toolkit.fluxcd.io","reconciler kind":"Kustomization","name":"podinfo","namespace":"default"}
{"level":"info","ts":"2022-08-02T12:56:34.961Z","logger":"controller.gitrepository","msg":"no changes since last reconcilation: observed revision","reconciler group":"source.toolkit.fluxcd.io","reconciler kind":"GitRepository","name":"podinfo","namespace":"default"}
{"level":"error","ts":"2022-08-02T12:56:34.961Z","logger":"controller.kustomization","msg":"no changes since last reconcilation: observed revision","reconciler group":"kustomize.toolkit.fluxcd.io","reconciler kind":"Kustomization","name":"podinfo","namespace":"flux-system"}`

type testResponseMapper struct {
}

func (t *testResponseMapper) DoRaw(_ context.Context) ([]byte, error) {
	return nil, nil
}

func (t *testResponseMapper) Stream(_ context.Context) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(testPodLogs)), nil
}

func TestLogRequest(t *testing.T) {
	mapper := &testResponseMapper{}
	tests := []struct {
		name       string
		namespace  string
		flags      *logsFlags
		assertFile string
	}{
		{
			name: "all logs",
			flags: &logsFlags{
				tail:          -1,
				allNamespaces: true,
			},
			assertFile: "testdata/logs/all-logs.txt",
		},
		{
			name:      "filter by namespace",
			namespace: "default",
			flags: &logsFlags{
				tail: -1,
			},
			assertFile: "testdata/logs/namespace.txt",
		},
		{
			name: "filter by kind and namespace",
			flags: &logsFlags{
				tail: -1,
				kind: "Kustomization",
			},
			assertFile: "testdata/logs/kind.txt",
		},
		{
			name: "filter by loglevel",
			flags: &logsFlags{
				tail:          -1,
				logLevel:      "error",
				allNamespaces: true,
			},
			assertFile: "testdata/logs/log-level.txt",
		},
		{
			name:      "filter by namespace, name, loglevel and kind",
			namespace: "flux-system",
			flags: &logsFlags{
				tail:     -1,
				logLevel: "error",
				kind:     "Kustomization",
				name:     "podinfo",
			},
			assertFile: "testdata/logs/multiple-filters.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			logsArgs = tt.flags
			if tt.namespace != "" {
				*kubeconfigArgs.Namespace = tt.namespace
			}
			w := bytes.NewBuffer([]byte{})
			err := logRequest(context.Background(), mapper, w)
			g.Expect(err).To(BeNil())

			got := make([]byte, w.Len())
			_, err = w.Read(got)
			g.Expect(err).To(BeNil())

			expected, err := os.ReadFile(tt.assertFile)
			g.Expect(err).To(BeNil())

			fmt.Printf("'%s'\n", got)
			g.Expect(string(got)).To(Equal(string(expected)))

			// reset flags to default
			*kubeconfigArgs.Namespace = rootArgs.defaults.Namespace
			logsArgs = &logsFlags{
				tail: -1,
			}
		})
	}
}
