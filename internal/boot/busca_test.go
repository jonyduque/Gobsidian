package boot_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/search"
)

func TestPrepararBuscaSemCacheConstroiEMarcaPronta(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	idx, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	inv := search.NewInverted()
	inv.MarkBuilding()
	boot.PrepararBusca(context.Background(), v, idx, inv, cfg, logSilencioso())
	defer func() { _ = inv.Close() }()
	if inv.Building() {
		t.Fatal("PrepararBusca devia ter marcado Ready")
	}
	for _, p := range idx.NotePaths() {
		if !inv.HasDoc(string(p)) {
			t.Fatalf("nota %q fora do indice de busca", p)
		}
	}
	doCache, _, err := search.LoadInvertedCache(context.Background(), cfg.CacheDir, cfg.VaultPath)
	if err != nil {
		t.Fatalf("PrepararBusca devia ter gravado o cache de busca: %v", err)
	}
	// O carregamento com sucesso pode deixar a arena de posicoes mapeada; sem
	// isto o mapeamento fica aberto ate o processo de teste terminar.
	_ = doCache.Close()
}

func TestPrepararBuscaComCacheCompletoAdota(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	idx, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	primeiro := search.NewInverted()
	primeiro.MarkBuilding()
	boot.PrepararBusca(context.Background(), v, idx, primeiro, cfg, logSilencioso())
	_ = primeiro.Close()

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	segundo := search.NewInverted()
	segundo.MarkBuilding()
	boot.PrepararBusca(context.Background(), v, idx, segundo, cfg, log)
	defer func() { _ = segundo.Close() }()
	if segundo.Building() {
		t.Fatal("devia estar pronto")
	}
	if !strings.Contains(buf.String(), "origem=cache") {
		t.Fatalf("log nao diz origem=cache:\n%s", buf.String())
	}
}

func TestPrepararBuscaCtxCanceladoNaoMarcaPronta(t *testing.T) {
	v, cfg := cofreDeTeste(t)
	idx, _, err := boot.AbrirIndice(context.Background(), v, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	inv := search.NewInverted()
	inv.MarkBuilding()
	boot.PrepararBusca(ctx, v, idx, inv, cfg, logSilencioso())
	defer func() { _ = inv.Close() }()
	if !inv.Building() {
		t.Fatal("ctx cancelado antes de construir: o indice de busca nao pode ser marcado Ready")
	}
	if _, _, err := search.LoadInvertedCache(context.Background(), cfg.CacheDir, cfg.VaultPath); err == nil {
		t.Fatal("ctx cancelado: nenhum cache de busca devia ter sido gravado")
	}
}
