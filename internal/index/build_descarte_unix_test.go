//go:build !windows

package index_test

import (
	"os"
	"testing"
)

func lockFileForTest(t *testing.T, path string) func() {
	t.Helper()
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	return func() {
		_ = os.Chmod(path, 0644)
	}
}
