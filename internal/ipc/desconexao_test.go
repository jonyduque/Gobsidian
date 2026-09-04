package ipc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestEhDesconexaoLimpaReconheceAsQuatroFormas(t *testing.T) {
	limpos := []error{nil, context.Canceled, io.EOF, io.ErrClosedPipe, os.ErrClosed,
		fmt.Errorf("copiando: %w", os.ErrClosed)}
	for _, e := range limpos {
		if !EhDesconexaoLimpa(e) {
			t.Errorf("EhDesconexaoLimpa(%v) = false", e)
		}
	}
	sujos := []error{errors.New("falha real"), context.DeadlineExceeded, io.ErrUnexpectedEOF}
	for _, e := range sujos {
		if EhDesconexaoLimpa(e) {
			t.Errorf("EhDesconexaoLimpa(%v) = true", e)
		}
	}
}
