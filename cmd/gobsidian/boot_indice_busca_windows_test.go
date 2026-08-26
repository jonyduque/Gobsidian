//go:build windows

package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
	"github.com/jonyd/gobsidian/internal/vault"
)

// sondaDeBoot é um termo que não existe em nenhum outro corpus deste pacote.
// Se ele chegar ao índice, o arquivo foi aberto — não há outra forma.
const sondaDeBoot = "sesquipedaliano"

// marcarOffline põe FILE_ATTRIBUTE_OFFLINE, que vault.IsCloudOnly aceita e que,
// ao contrário de FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS, é gravável por
// SetFileAttributes. É como se monta um placeholder de nuvem em teste.
func marcarOffline(t *testing.T, abs string) {
	t.Helper()
	p, err := windows.UTF16PtrFromString(vault.LongPath(abs))
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
	}
	t.Cleanup(func() { _ = windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_NORMAL) })
}

// TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem exercita o BOOT DE PRODUÇÃO.
//
// Este teste existe porque o que "provava" a regra antes era um dublê: um
// helper chamado construirComoOBoot que chamava Inverted.Update num laço,
// afirmando em comentário que era "exatamente como buildInvertedIndex faz".
// Não era — a produção chamava v.ReadAll (um os.ReadFile puro, sem consulta a
// CloudOnly) e inv.Add direto, escapando da guarda inteira. O teste afirmava
// sobre a reimplementação, não sobre o código que roda.
//
// A consequência em uso: num cofre OneDrive sem cache válido — primeira
// execução, cache corrompido, ou QUALQUER troca de formato — a reconstrução
// baixava todo placeholder do cofre em segundo plano. É a falha que o
// ARMADILHAS.md registra como "trava a máquina do usuário sem dizer por quê".
func TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem(t *testing.T) {
	dir := t.TempDir()

	hidratada := filepath.Join(dir, "hidratada.md")
	if err := os.WriteFile(hidratada, []byte("# Local\n\npalavra comum aqui.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	naNuvem := filepath.Join(dir, "na-nuvem.md")
	if err := os.WriteFile(naNuvem, []byte("# Nuvem\n\n"+sondaDeBoot+" nao deve ser lido.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	anexo := filepath.Join(dir, "imagem.png")
	if err := os.WriteFile(anexo, []byte(sondaDeBoot+" tambem nao.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := vault.New(dir)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	// Marcar DEPOIS do Build: o índice de metadados já tem a entrada, e é
	// justamente esse caminho que NotePaths devolve ao boot da busca.
	marcarOffline(t, naNuvem)

	inv := search.NewInverted()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{VaultPath: dir, CacheDir: t.TempDir()}

	buildInvertedIndex(context.Background(), v, idx, inv, cfg, log)

	// 1. O placeholder NAO pode ter sido aberto.
	if postings := inv.Postings(sondaDeBoot); len(postings) != 0 {
		var onde []string
		for _, p := range postings {
			onde = append(onde, p.Path)
		}
		t.Errorf("o boot ABRIU arquivo que nao devia: %q entrou no indice via %v",
			sondaDeBoot, onde)
	}

	// 2. Mas TEM de contar como coberto, senao todo boot conclui "cache
	//    parcial" e regrava o cache inteiro.
	if !inv.HasDoc("na-nuvem.md") {
		t.Error("o placeholder nao ficou coberto; o cache concluira 'parcial' em todo boot")
	}

	// 3. Contrapeso: a nota local hidratada TEM de ter sido indexada. Sem
	//    isto, "nao abrir nada" passaria neste teste e desligaria a busca.
	if inv.DocLength("hidratada.md") == 0 {
		t.Error("a nota hidratada nao foi indexada; a guarda ficou larga demais")
	}
}
