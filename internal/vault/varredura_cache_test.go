package vault_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/vault"
)

// TestVarrerTemporariosAntigosSoApagaOVelho: no diretorio de cache gravam ao
// mesmo tempo o daemon, a ponte em processo e a CLI, e o boot de um deles nao e
// momento sem escrita em voo para os outros -- por isso a varredura do cofre
// (SweepStaleTempFiles) nunca entrou ali. A regra aqui e a idade: uma gravacao
// de cache leva segundos, entao um temporario de mais de uma hora nao esta em
// voo. Medido em 2026-10-01 no cache de Estudo: 230 MB em tres temporarios de
// 2026-09-04 e 2026-09-21 que nada apagava.
func TestVarrerTemporariosAntigosSoApagaOVelho(t *testing.T) {
	dir := t.TempDir()
	escreve := func(nome string, idade time.Duration) string {
		p := filepath.Join(dir, nome)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		quando := time.Now().Add(-idade)
		if err := os.Chtimes(p, quando, quando); err != nil {
			t.Fatal(err)
		}
		return p
	}
	velho := escreve(vault.TempFilePrefix+"123", 2*time.Hour)
	velhoGob := escreve(vault.TempFilePrefix+"cache-456.gob", 30*24*time.Hour)
	emVoo := escreve(vault.TempFilePrefix+"789", time.Minute)
	cache := escreve("index_cache.gob", 30*24*time.Hour)
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	fundo := filepath.Join(sub, vault.TempFilePrefix+"999")
	if err := os.WriteFile(fundo, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	antigo := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(fundo, antigo, antigo); err != nil {
		t.Fatal(err)
	}

	res, err := vault.VarrerTemporariosAntigos(context.Background(), dir, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if res.Removidos != 2 {
		t.Errorf("Removidos = %d, quer 2 (%+v)", res.Removidos, res)
	}
	for _, p := range []string{velho, velhoGob} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s ficou: temporario velho tinha de sair", filepath.Base(p))
		}
	}
	for _, p := range []string{emVoo, cache, fundo} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s sumiu (%v): so o temporario velho da raiz do diretorio sai", filepath.Base(p), err)
		}
	}
}

// TestVarrerTemporariosAntigosSemDiretorio: cofre que nunca gravou cache nao
// tem o diretorio, e isso nao e erro.
func TestVarrerTemporariosAntigosSemDiretorio(t *testing.T) {
	res, err := vault.VarrerTemporariosAntigos(context.Background(), filepath.Join(t.TempDir(), "nao-existe"), time.Hour)
	if err != nil || res.Removidos != 0 {
		t.Errorf("res = %+v, err = %v; quer zero e nil", res, err)
	}
}
