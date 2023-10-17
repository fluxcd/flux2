//go:build e2e
// +build e2e

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
	"testing"
)

func TestLogsNoArgs(t *testing.T) {
	cmd := cmdTestCase{
		args:   "logs",
		assert: assertSuccess(),
	}
	cmd.runTestCmd(t)
}

func TestLogsWrongNamespace(t *testing.T) {
	cmd := cmdTestCase{
		args:   "logs --flux-namespace=default",
		assert: assertError(`no Flux pods found in namespace "default"`),
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
		assert: assertError("at most one of `sinceTime` or `sinceDuration` may be specified"),
	}
	cmd.runTestCmd(t)
}
