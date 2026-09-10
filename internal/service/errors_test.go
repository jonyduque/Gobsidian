package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/vault"
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

// TestServicoSemIndiceDevolveVaultUnavailable trava a regressao do achado
// 1.13: sem indice, as tools que leem so do indice devolviam um erro cru sem
// codigo (que o mcpsrv traduzia para INTERNAL) em vez de CodeVaultUnavailable,
// o codigo que a mesma condicao ja usa em outros pontos do servico (ex.:
// read.go). VaultStats NAO entra aqui: sem indice ela cai para uma varredura
// direta do cofre (vault.Walk) e nao e um erro.
func TestServicoSemIndiceDevolveVaultUnavailable(t *testing.T) {
	root := t.TempDir()
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(v, nil, nil, nil, Options{})
	ctx := context.Background()

	chamadas := map[string]func() error{
		"LinkGraph": func() error { _, e := svc.LinkGraph(ctx, GraphRequest{Path: "a.md"}); return e },
		"TagList":   func() error { _, e := svc.TagList(ctx, TagRequest{}); return e },
		"ListNotes": func() error { _, e := svc.ListNotes(ctx, ListRequest{}); return e },
		"Metadata":  func() error { _, e := svc.NoteMetadata(ctx, MetadataRequest{Path: "a.md"}); return e },
	}
	for nome, f := range chamadas {
		err := f()
		if err == nil {
			t.Errorf("%s sem indice devolveu nil", nome)
			continue
		}
		if got := CodeOf(err); got != CodeVaultUnavailable {
			t.Errorf("%s sem indice: codigo = %s, quer %s (%v)", nome, got, CodeVaultUnavailable, err)
		}
	}
}
