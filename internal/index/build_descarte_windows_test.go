//go:build windows

package index_test

import (
	"syscall"
	"testing"
)

func lockFileForTest(t *testing.T, path string) func() {
	t.Helper()
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	return func() {
		_ = syscall.CloseHandle(h)
	}
}
