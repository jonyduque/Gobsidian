package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestShutdownExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"context.Canceled", context.Canceled, 0},
		{"erro embrulhado com context.Canceled", fmt.Errorf("serve: %w", context.Canceled), 0},
		{"io.EOF", io.EOF, 0},
		{"erro embrulhado com io.EOF", fmt.Errorf("sdk: %w", io.EOF), 0},
		{"io.ErrClosedPipe", io.ErrClosedPipe, 0},
		{"erro embrulhado com io.ErrClosedPipe", fmt.Errorf("sdk: %w", io.ErrClosedPipe), 0},
		{"os.ErrClosed", os.ErrClosed, 0},
		{"prazo estourado", context.DeadlineExceeded, 1},
		{"erro real", errors.New("falha real"), 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shutdownExitCode(tc.err); got != tc.want {
				t.Fatalf("shutdownExitCode(%v) = %d, esperado %d", tc.err, got, tc.want)
			}
		})
	}
}
