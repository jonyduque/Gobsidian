package boot_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyduque/Gobsidian/internal/boot"
	"github.com/jonyduque/Gobsidian/internal/config"
	"github.com/jonyduque/Gobsidian/internal/index"
	"github.com/jonyduque/Gobsidian/internal/vault"
)

func cofreDeTeste(t *testing.T) (*vault.Vault, config.Config) {
	t.Helper()
	raiz := t.TempDir()
	if err := os.WriteFile(filepath.Join(raiz, "a.md"), []byte("# A\n\nliga [[b]] #tag\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "b.md"), []byte("# B\n\ntexto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := vault.New(raiz)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{VaultPath: raiz, CacheDir: t.TempDir()}
	return v, cfg
}

func logSilencioso() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestAbrirIndiceSemCacheConstroiEGrava(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	idx, origem, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if origem != "build" {
		t.Fatalf("origem = %q, quero \"build\"", origem)
	}
	if idx.NoteCount() != 2 {
		t.Fatalf("NoteCount = %d, quero 2", idx.NoteCount())
	}
	if _, _, err := index.LoadIndexCache(context.Background(), cfg.CacheDir, cfg.VaultPath); err != nil {
		t.Fatalf("AbrirIndice devia ter gravado o cache: %v", err)
	}
}

func TestAbrirIndiceComCacheFrescoCarrega(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	if _, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso()); err != nil {
		t.Fatal(err)
	}
	idx, origem, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if origem != "cache" {
		t.Fatalf("origem = %q, quero \"cache\"", origem)
	}
	if idx.NoteCount() != 2 {
		t.Fatalf("NoteCount = %d, quero 2", idx.NoteCount())
	}
}

func TestAbrirIndiceComCacheVelhoReconstroi(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	if _, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso()); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(cfg.VaultPath, "a.md")
	if err := os.WriteFile(p, []byte("# A mudou\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// mtime explicito 2 s a frente: VerifyFreshness compara mtime e tamanho,
	// e dois writes no mesmo tick de relogio podem empatar.
	if err := os.Chtimes(p, time.Now().Add(2*time.Second), time.Now().Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	idx, origem, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if origem != "build" {
		t.Fatalf("origem = %q, quero \"build\" (cache velho)", origem)
	}
	_, cp, err := vault.Resolve(v.Root(), "a.md")
	if err != nil {
		t.Fatal(err)
	}
	n, ok := idx.Get(cp)
	if !ok || n.Title != "A mudou" {
		t.Fatalf("indice reconstruido nao viu a edicao: ok=%v n=%+v", ok, n)
	}
}

func TestAbrirIndiceComCacheCorrompidoReconstroi(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	if _, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso()); err != nil {
		t.Fatal(err)
	}
	entradas, err := os.ReadDir(cfg.CacheDir)
	if err != nil || len(entradas) != 1 {
		t.Fatalf("esperava um unico arquivo no cache dir, tenho %d (err=%v)", len(entradas), err)
	}
	p := filepath.Join(cfg.CacheDir, entradas[0].Name())
	if err := os.Truncate(p, 16); err != nil {
		t.Fatal(err)
	}
	_, origem, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if origem != "build" {
		t.Fatalf("origem = %q, quero \"build\" (cache corrompido)", origem)
	}
}
