// +build unit

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
