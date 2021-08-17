// +build unit

package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Ensure tests print consistent timestamps regardless of timezone
	os.Setenv("TZ", "UTC")
	os.Exit(m.Run())
}
