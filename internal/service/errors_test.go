package service

import (
	"errors"
	"testing"
)

// TestErrorIsTypedNilTarget trava a regressao do achado 4: um alvo
// error-interface nao-nulo que carrega um *Error nulo nao pode fazer Is
// desreferenciar target.Code. Antes da guarda, errors.Is(err, (*Error)(nil))
// entrava em panic.
func TestErrorIsTypedNilTarget(t *testing.T) {
	err := Errorf(CodeNoteNotFound, "nota nao encontrada")

	var nilTarget *Error
	if got := errors.Is(err, nilTarget); got {
		t.Fatal("Is deveria devolver false para um alvo *Error nulo, nao panicar nem devolver true")
	}
}
