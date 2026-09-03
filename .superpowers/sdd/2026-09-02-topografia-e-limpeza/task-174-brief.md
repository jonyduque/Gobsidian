### Task 174: e PR1 — `internal/boot`: abrir índice, preparar busca, montar componentes

**Files:**
- Create: `internal/boot/doc.go`, `internal/boot/indice.go`, `internal/boot/busca.go`, `internal/boot/montar.go`
- Create: `internal/boot/indice_test.go`, `internal/boot/busca_test.go`, `internal/boot/montar_test.go` (pacote `boot_test`)
- Move: `cmd/gobsidian/inverted_cache_state_test.go` → `internal/boot/estado_do_cache_test.go`; `cmd/gobsidian/boot_indice_busca_windows_test.go` → `internal/boot/busca_windows_test.go` (pacote `boot`, caixa-branca)
- Delete: `cmd/gobsidian/servico.go`
- Modify: `cmd/gobsidian/serve.go` (apagar `carregarIndiceDoCache:99`, `invertedSaveInterval`, `invertedCacheState:155`, `devolveMemoriaTransitoria:198`, `prepararIndiceDeBusca:213`, `buildInvertedIndex:274`, `watcherStats`; `serveEmProcesso:381` chama `boot.Montar`)
- Modify: `cmd/gobsidian/daemon.go:150-184` (chama `boot.Montar`)
- Modify: `CLAUDE.md` (árvore + grafo), `docs/ESTRUTURA.md:12-18,:216`, `docs/ARCHITECTURE.md` (camada de montagem)

**Interfaces:**
- Consumes: `service.New(v, idx *index.Index, inv, w, opts)` (Task 173); `vault.SweepStaleTempFiles` (Task 172); `vaulttest.Prazo` e os helpers de SO de `internal/vaulttest` (Task 159); `index.LoadIndexCache`, `index.VerifyFreshness`, `index.SaveIndexCache`; `search.LoadInvertedCache`, `(*Inverted).AdotarDe/MarkBuilding/MarkReady/Building/HasDoc/DocCount`; `watcher.New(v, idx, inv, debounce, log)`, `(*Watcher).Run(ctx)/Close()/Stats()`.
- Produces (Tasks 175–177 dependem destes nomes exatos):

```go
package boot

// AbrirIndice devolve o indice de metadados: do cache, se existir e estiver
// fresco, ou construido do cofre e gravado. origem e "cache" ou "build".
func AbrirIndice(ctx context.Context, v *vault.Vault, cfg config.Config, log *slog.Logger) (idx *index.Index, origem string, err error)

// PrepararBusca deixa inv pronto: adota o cache de busca completo, retoma um
// parcial, ou constroi do indice. Marca Ready ao fim, salvo ctx cancelado.
func PrepararBusca(ctx context.Context, v *vault.Vault, idx *index.Index, inv *search.Inverted, cfg config.Config, log *slog.Logger)

// Componentes e tudo que serve e daemon compartilham depois de montar.
type Componentes struct {
	Vault    *vault.Vault
	Index    *index.Index
	Inverted *search.Inverted
	Watcher  *watcher.Watcher
	Service  *service.Service
	espera   sync.WaitGroup
}

// Esperar bloqueia ate as goroutines de fundo (busca, watcher) terminarem.
func (c *Componentes) Esperar()

// Montar e o corpo do antigo construirServico: cofre, varredura de temporarios,
// indice, busca (eager ou preguicosa), watcher, Service, e o log
// "servidor pronto" que scripts/measure.ps1 le.
func Montar(ctx context.Context, cfg config.Config, log *slog.Logger) (*Componentes, error)
```

Grafo: `boot → config, index, search, service, vault, watcher` — todas arestas que `cmd/gobsidian` já tinha; nenhum pacote de domínio importa `boot`. Em `_test`: `+ vaulttest`.

- [ ] **Step 1: Testes de caracterização primeiro — o pacote ainda não existe, eles falham por compilação**

`internal/boot/indice_test.go`:

```go
package boot_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/vaulttest"
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
```

Assinaturas reais (conferidas em 2026-09-02): `vault.New(root string, opcoes ...Opcao)`, `vault.Resolve(root, input string) (string, CanonicalPath, error)`, `index.LoadIndexCache(ctx, cacheDir, vaultPath string) (*Index, *CacheHeader, error)`, `index.Note.Title`.

`internal/boot/busca_test.go`:

```go
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
	if inv.Building() {
		t.Fatal("PrepararBusca devia ter marcado Ready")
	}
	for _, p := range idx.NotePaths() {
		if !inv.HasDoc(p) {
			t.Fatalf("nota %q fora do indice de busca", p)
		}
	}
	if _, _, err := search.LoadInvertedCache(context.Background(), cfg.CacheDir, cfg.VaultPath); err != nil {
		t.Fatalf("PrepararBusca devia ter gravado o cache de busca: %v", err)
	}
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
	primeiro.Close()

	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	segundo := search.NewInverted()
	segundo.MarkBuilding()
	boot.PrepararBusca(context.Background(), v, idx, segundo, cfg, log)
	defer segundo.Close()
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
	if !inv.Building() {
		t.Fatal("ctx cancelado antes de construir: o indice de busca nao pode ser marcado Ready")
	}
	if _, _, err := search.LoadInvertedCache(context.Background(), cfg.CacheDir, cfg.VaultPath); err == nil {
		t.Fatal("ctx cancelado: nenhum cache de busca devia ter sido gravado")
	}
}
```

`search.LoadInvertedCache(ctx, cacheDir, vaultPath string) (*Inverted, *CacheHeader, error)` — conferida em 2026-09-02; o teste só usa o `err`. Se o carregamento com sucesso deixar um mapeamento aberto, fechar o `*Inverted` devolvido (`Close`).

`internal/boot/montar_test.go`:

```go
package boot_test

import (
	"context"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/boot"
	"github.com/jonyd/gobsidian/internal/vaulttest"
)

func TestMontarDevolveServicoPronto(t *testing.T) {
	_, cfg := cofreDeTeste(t)
	cfg.EagerSearch = true
	cfg.DebounceMS = 50
	ctx, cancel := context.WithCancel(context.Background())
	c, err := boot.Montar(ctx, cfg, logSilencioso())
	if err != nil {
		t.Fatal(err)
	}
	if c.Service == nil || c.Index == nil || c.Inverted == nil || c.Watcher == nil || c.Vault == nil {
		t.Fatalf("Componentes incompleto: %+v", c)
	}
	if c.Index.NoteCount() != 2 {
		t.Fatalf("NoteCount = %d, quero 2", c.Index.NoteCount())
	}
	cancel()
	_ = c.Watcher.Close()
	feito := make(chan struct{})
	go func() { c.Esperar(); close(feito) }()
	select {
	case <-feito:
	case <-time.After(vaulttest.Prazo):
		t.Fatal("Esperar nao voltou depois de cancelar ctx e fechar o watcher")
	}
}
```

Run: `go test ./internal/boot/` — Expected: FAIL de compilação (`package gobsidian/internal/boot` não existe). Colar a primeira linha.

- [ ] **Step 2: Criar o pacote movendo o código — sem reescrever**

`internal/boot/doc.go`:

```go
// Package boot monta o servidor: cofre, indice de metadados (do cache ou
// construido), indice de busca (adotado do cache, retomado ou construido),
// watcher e Service. Existe porque serve, daemon e os subcomandos de CLI
// precisam da mesma montagem, e ela vivia em cmd/gobsidian, onde nenhum
// teste de pacote a alcancava.
//
// Nao importa mcpsrv nem lifecycle: quem monta nao decide como o host
// conversa nem quando encerra (ver Task 177 para a vigilia do host).
package boot
```

`internal/boot/indice.go` — corpo de `carregarIndiceDoCache` (`serve.go:99`) mais o ramo `else` de `construirServico` (`index.New()+Build+SaveIndexCache`), na forma:

```go
func AbrirIndice(ctx context.Context, v *vault.Vault, cfg config.Config, log *slog.Logger) (*index.Index, string, error) {
	if idx, ok := carregarIndiceDoCache(ctx, v, cfg, log); ok {
		return idx, "cache", nil
	}
	idx := index.New()
	if err := idx.Build(ctx, v); err != nil {
		return nil, "", fmt.Errorf("construindo indice: %w", err)
	}
	if err := index.SaveIndexCache(ctx, cfg.CacheDir, cfg.VaultPath, idx); err != nil {
		// Cache e aceleracao, nao requisito: avisar e seguir com o indice em memoria.
		log.Warn("nao foi possivel gravar o cache de indice", "err", err)
	}
	return idx, "build", nil
}

// carregarIndiceDoCache e o antigo cmd/gobsidian.carregarIndiceDoCache, movido inteiro.
func carregarIndiceDoCache(ctx context.Context, v *vault.Vault, cfg config.Config, log *slog.Logger) (*index.Index, bool) {
	// corpo de serve.go:99-153, verbatim
}
```

As mensagens de log de `carregarIndiceDoCache` (Warn/Info) ficam como estão. Assinaturas reais: `(*Index).Build(ctx, v *vault.Vault) error`, `index.SaveIndexCache(ctx, cacheDir, vaultPath string, ix *Index) error`.

`internal/boot/busca.go` — mover `prepararIndiceDeBusca:213` (renomeada `PrepararBusca`), `buildInvertedIndex:274` (renomeada `construirBusca`, não exportada), `invertedCacheState:155` (renomeada `estadoDoCache`), `invertedSaveInterval` (renomeada `intervaloDeGravacao`) e `devolveMemoriaTransitoria:198`. Corpos verbatim; só o nome muda. O guarda `if ctx.Err() != nil {` dentro de `construirBusca` fica **exatamente** com essa grafia — é a âncora da mutação.

`internal/boot/montar.go` — `Componentes`, `Esperar`, `Montar` com o corpo de `construirServico` (`servico.go:49-247`): `vault.New(cfg.VaultPath, vault.SeguirSymlinks(cfg.FollowSymlinks))`; goroutine de `vault.SweepStaleTempFiles` juntada antes de `watcher.New` com os mesmos `Warn`; `AbrirIndice` no lugar do `if/else` de cache; `indexMS`; `inv := search.NewInverted(); inv.MarkBuilding()`; `watcher.New(v, idx, inv, time.Duration(cfg.DebounceMS)*time.Millisecond, log)` (a conversão que `servico.go` já faz — copiar a expressão real); `service.Options{ReadOnly, MaxResults}` e a closure `CarregarBusca` quando `!cfg.EagerSearch` (a closure chama `PrepararBusca`); `service.New(v, idx, inv, watcherStats{w: w}, opts)`; a goroutine em `c.espera` que, se eager, chama `PrepararBusca` e depois `w.Run(ctx)`; e o log **inalterado**:

```go
	log.Info("servidor pronto",
		"vault", cfg.VaultPath, "read_only", cfg.ReadOnly,
		"notes", idx.NoteCount(), "assets", idx.AssetCount(),
		"index_ms", indexMS, "index_origin", origem)
```

`index_origin` agora vem de `AbrirIndice` ("cache"/"build") — confirmar que os valores anteriores em `servico.go` eram estes dois literais; se eram outros (`"cached"`/`"built"`), manter os antigos: `scripts/measure.ps1` lê esta linha e o texto não muda nesta Task.

`watcherStats` (`serve.go`, fim do arquivo) muda para `montar.go`, não exportado.

- [ ] **Step 3: `serve.go` e `daemon.go` chamam `boot.Montar`**

`serveEmProcesso` (`serve.go:381`): `montado, err := construirServico(ctx, cfg, log)` → `c, err := boot.Montar(ctx, cfg, log)`; `montado.svc` → `c.Service`; `montado.w.Close()` → `c.Watcher.Close()`; `montado.wg.Wait()` → `c.Esperar()`. Idem em `daemon.go:150-184`. `git rm cmd/gobsidian/servico.go`. Apagar de `serve.go` tudo listado em **Files**.

`git mv cmd/gobsidian/inverted_cache_state_test.go internal/boot/estado_do_cache_test.go` (pacote `boot`; `invertedCacheState` → `estadoDoCache`). `git mv cmd/gobsidian/boot_indice_busca_windows_test.go internal/boot/busca_windows_test.go` (pacote `boot`; `buildInvertedIndex` → `construirBusca`; a build tag `//go:build windows` permanece).

Run: `go build ./... && go vet ./...` — Expected: limpo.
Run: `go test -race ./internal/boot/ -v` — Expected: os 8 testes PASS.
Run: `go test -race ./cmd/... ./internal/daemon/ ./internal/mcpsrv/` — Expected: PASS.

- [ ] **Step 4: Prova de mutação**

```powershell
pwsh -File scripts/mutate.ps1 -Path internal/boot/busca.go -Anchor 'if ctx.Err() != nil {' -Replacement 'if false {' -Test TestPrepararBuscaCtxCanceladoNaoMarcaPronta -Package ./internal/boot/
```

Expected: exit 0 (o teste FALHA sob mutação: com o guarda morto, o índice é construído e marcado Ready mesmo com ctx cancelado). Colar a saída.

- [ ] **Step 5: `go list` e órfãos**

Run: `go list -f '{{.ImportPath}} {{.Imports}}' ./internal/boot/ | tr ' ' '\n' | grep gobsidian` — Expected: exatamente `config index search service vault watcher`. Nenhum `mcpsrv`, nenhum `lifecycle`, nenhum `net`.
Run: `go list -f '{{.Imports}}' ./internal/index/ ./internal/search/ ./internal/service/ ./internal/watcher/ | grep -c boot` — Expected: `0`.
Run: `pwsh -File scripts/test_orphans.ps1` — Expected: quatro `[OK]`.

- [ ] **Step 6: Documentação**

- `CLAUDE.md`, árvore: linha `boot/  monta cofre, indice, busca, watcher e Service; serve, daemon e CLI chamam` entre `console/` e `ipc/`; grafo: linha `boot     → config, index, search, service, vault, watcher` depois de `mcpsrv`. Remover `servico.go` da descrição de `cmd/gobsidian/`.
- `docs/ESTRUTURA.md:12-18`: apagar `servico.go`, acrescentar `internal/boot/` com os quatro arquivos; `:216` idem.
- `docs/ARCHITECTURE.md`: um parágrafo na seção de camadas — "montagem" é camada própria acima de `service` e abaixo de `mcpsrv`/`cmd`.

Run: `python -c "open('CLAUDE.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"`.

- [ ] **Step 7: Medição — o boot não pode ficar mais lento por ter mudado de pacote**

`scripts/measure.ps1 -Vault <vault_5000>` antes (binário da Baseline/M4) e depois, 3 execuções cada, `index_ms` e wall de `servidor pronto`. Expected: dentro do ruído de M4 (101–123 ms `index_ms` quente). Colar as seis linhas.

- [ ] **Step 8: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/boot/ cmd/gobsidian/serve.go cmd/gobsidian/daemon.go cmd/gobsidian/servico.go cmd/gobsidian/inverted_cache_state_test.go cmd/gobsidian/boot_indice_busca_windows_test.go CLAUDE.md docs/ESTRUTURA.md docs/ARCHITECTURE.md
git commit -m "refactor(boot): assemble vault, index, search and service in one testable package"
```

#### Verificações

- FAIL de compilação do Step 1 colado; 8 PASS do Step 3 colados.
- `mutate.ps1` do Step 4 com exit 0, saída colada.
- `go list` do Step 5 com a lista exata; `test_orphans.ps1` quatro `[OK]`.
- Linha `servidor pronto` com as mesmas chaves de antes (colar uma do `serve` real).
- Medição do Step 7 colada; `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. `git mv`/`git rm` por caminho são permitidos.
- Movimento, não reescrita: corpos verbatim, só nomes e assinaturas mudam. Melhoria que você enxergar vai para o relatório como sugestão, não para o diff.
- `boot` não importa `mcpsrv`, `lifecycle`, `ipc`, `daemon`, `doctor`. Nenhum pacote de domínio importa `boot`.
- Log só por `log *slog.Logger` recebido; nenhum `fmt.Print*`.
- Testes de caracterização usam cofre real em `t.TempDir()` e `CacheDir` em `t.TempDir()` — nunca o cache padrão do usuário.

#### Comando de mutação

Step 4 (`mutate.ps1`, âncora `if ctx.Err() != nil {` em `internal/boot/busca.go`).

#### Contrato de relatório

`task-174-report.md`: status, SHA, saídas dos Steps 1, 3, 4, 5, 7, linha `servidor pronto` real, última linha do `verify.ps1`, e a lista de nomes antigos → novos.

---
