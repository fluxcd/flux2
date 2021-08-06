package main

import (
	"testing"
)

func TestLogsNoArgs(t *testing.T) {
	cmd := cmdTestCase{
		args:      "logs",
		wantError: false,
	}
	cmd.runTestCmd(t)
}

func TestLogsAllNamespaces(t *testing.T) {
	cmd := cmdTestCase{
		args:      "logs --all-namespaces",
		wantError: false,
	}
	cmd.runTestCmd(t)
}

func TestLogsSince(t *testing.T) {
	cmd := cmdTestCase{
		args:      "logs --since=2m",
		wantError: false,
	}
	cmd.runTestCmd(t)
}

func TestLogsSinceInvalid(t *testing.T) {
	cmd := cmdTestCase{
		args:        "logs --since=XXX",
		wantError:   true,
		goldenValue: `invalid argument "XXX" for "--since" flag: time: invalid duration "XXX"`,
	}
	cmd.runTestCmd(t)
}

func TestLogsSinceTime(t *testing.T) {
	cmd := cmdTestCase{
		args:      "logs --since-time=2021-08-06T14:26:25.546Z",
		wantError: false,
	}
	cmd.runTestCmd(t)
}

func TestLogsSinceTimeInvalid(t *testing.T) {
	cmd := cmdTestCase{
		args:        "logs --since-time=XXX",
		wantError:   true,
		goldenValue: "XXX is not a valid (RFC3339) time",
	}
	cmd.runTestCmd(t)
}

func TestLogsSinceOnlyOneAllowed(t *testing.T) {
	cmd := cmdTestCase{
		args:        "logs --since=2m --since-time=2021-08-06T14:26:25.546Z",
		wantError:   true,
		goldenValue: "at most one of `sinceTime` or `sinceSeconds` may be specified",
	}
	cmd.runTestCmd(t)
}
