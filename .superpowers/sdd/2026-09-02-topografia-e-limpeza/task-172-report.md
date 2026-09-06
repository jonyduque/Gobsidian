# Task 172 — report

Status: DONE
SHA: ae619cffde7d87725afc8f0a048ed080fd518a75
Fix round 1 SHA: 68c0bc550043a2c2ec820e33cf2bf5599e15d26e

## Progresso

- 07:35 — Step 1 feito: `service/write.go` (5 chamadas `writer.WriteAtomic` -> `vault.WriteAtomic`), `cmd/gobsidian/servico.go` (`writer.SweepResult`, `writer.SweepStaleTempFiles`, comentario `:103` e `:68` -> `vault.*`), import `writer` removido de `servico.go` (nao sobrou uso). `go build ./...` limpo.
- 07:35 — Step 2 feito: `internal/index/persist.go` `SaveIndexCache` usa `vault.ReplaceFile`; temporario `.gobsidian-tmp-index-cache-*.gob` e o rename manual saíram. `go build ./...` limpo.
- 07:35 — Step 3 feito: `internal/search/persist.go` `SaveInvertedCache` usa `vault.ReplaceFile`; `promoverArenaSePresente(inv)` movido para ANTES de `inv.ExportForCache()`. Import `vault` adicionado. `go build ./...` limpo.
- 07:36 — Step 3 (testes): `go test -race ./internal/search/ -run 'TestSaveOverwritesMappedCache|TestSaveAndLoadInvertedCache|TestIndiceRecarregadoEIdenticoAoConstruido' -v` → PASS nos tres.
- 07:36 — Step 3 (prova de inversao): movi `promoverArenaSePresente(inv)` para DEPOIS do `ReplaceFile`, rodei `TestSaveOverwritesMappedCache` → **FAIL** confirmado, saida colada abaixo em "Mutação". Restaurei a ordem correta e reconfirmei PASS nos tres testes.
- 07:37 — Step 4 feito: testes de `writer` (`block_test.go`, `section_test.go`) migrados de `writer.WriteAtomic` para `vault.WriteAtomic`; import `vault` adicionado a `block_test.go` (`section_test.go` ja tinha). `git rm internal/writer/atomic.go`. `go build ./...` limpo. Grep de encaminhadores e grep N2 (`.gobsidian-tmp-` literal) confirmados vazios/restritos ao esperado (colados abaixo em "Verificações").
- 07:38 — N5 feito: acrescentada a frase do contrato do callback ao docstring de `vault.ReplaceFile` (não pode fechar/renomear o `*os.File`; buffer precisa ser esvaziado antes do retorno).
- 07:40 — `go vet ./...` limpo; `go test ./internal/writer/... ./internal/index/... ./internal/search/... ./internal/service/... ./cmd/gobsidian/...` → todos PASS (saída colada abaixo).
- 07:40 — `GOOS=windows go list -f '{{.Imports}}' ./internal/index/ ./internal/search/` conferido: nenhum ganhou `writer` (colado abaixo).
- 07:41 — Step 5: baselines `antes172_index.test.exe`/`antes172_search.test.exe` construídos a partir de um `git worktree` detached em `011042f` (nunca toquei a árvore de trabalho para isso — sem checkout/stash/reset), depois removido com `git worktree remove --force`. `depois172_*` construídos da árvore editada. `go test -c -o ...`, sem erro.
- 07:44 — Step 5: benchmarks rodados intercalados (antes,depois x7) contra `%TEMP%\vault_5000` (default de `GOBSIDIAN_BENCH_VAULT`, presente). `benchstat` colado abaixo. `docs/ESTADO.md` linhas 115/120 atualizadas com os números medidos + nota Task 172. `python -c "open('docs/ESTADO.md',encoding='utf-8').read()"` → `[OK] UTF-8 valido`.
- 07:44 a 08:05 — Step 6: `pwsh -File scripts/verify.ps1` (as 14 etapas, sem `-SkipCross`/`-SkipNet`) rodou em background, iniciado as 07:44 e terminado as 08:05 (21 min), com `[OK] Bateria completa. Pode commitar.` (exit code 0). Última linha colada abaixo. `test_orphans.ps1` NÃO rodado por mim, por instrução explícita do orquestrador (roda em separado).
- 08:06 — `pwsh -File scripts/audit_reports.ps1 172` rodado; achados respondidos na seção "Achados do audit_reports.ps1" abaixo. Re-rodado após as correções: zero achados restantes neste relatório.
- 08:09 — Commit criado com `git commit -F commit-172.txt`, staging por caminho explícito (nunca `git add -A`): `cmd/gobsidian/servico.go`, `docs/ESTADO.md`, `internal/index/persist.go`, `internal/search/persist.go`, `internal/service/write.go`, `internal/vault/atomic.go`, `internal/writer/atomic.go` (delete, já estava staged pelo `git rm` do Step 4), `internal/writer/block_test.go`, `internal/writer/section_test.go`. SHA `ae619cffde7d87725afc8f0a048ed080fd518a75`. `git status --short` confirmado depois: trabalho não commitado do dono (`progress.md` do marco anterior, `test-vault/`) permaneceu intocado.

## TDD

Não se aplica no sentido RED/GREEN de teste novo: esta tarefa é migração de
chamador mais troca de rotina de escrita (`os.CreateTemp`+`Rename` manual por
`vault.ReplaceFile`), e os três testes que fixam o comportamento
(`TestSaveOverwritesMappedCache`, `TestSaveAndLoadInvertedCache`,
`TestIndiceRecarregadoEIdenticoAoConstruido`) já existiam antes desta tarefa.
Nenhum teste novo foi escrito. A prova exigida pelo brief é a inversão manual
da ordem `promover → gravar`, documentada na seção "Mutação" abaixo — é o
mecanismo que a Task 172 usa no lugar de RED/GREEN.

## Mutação

O brief não usa `scripts/mutate.ps1` para esta tarefa (ver "Comando de
mutação" no brief): a regra provada é a ordem `promoverArenaSePresente(inv)`
ANTES de `ExportForCache`/`ReplaceFile`, e a prova é mover a chamada para
DEPOIS e observar o teste existente falhar.

Mutação aplicada (temporária, revertida em seguida):

```go
// ANTES (correto, no código commitado):
promoverArenaSePresente(inv)
termos, docLengths := inv.ExportForCache()
finalPath := filepath.Join(cacheDir, "inverted_cache.gob")
if err := vault.ReplaceFile(ctx, finalPath, func(f *os.File) error {
    return escreveCache(f, header, termos, docLengths)
}); err != nil { ... }

// MUTAÇÃO (promoção movida para depois do ReplaceFile):
termos, docLengths := inv.ExportForCache()
finalPath := filepath.Join(cacheDir, "inverted_cache.gob")
if err := vault.ReplaceFile(ctx, finalPath, func(f *os.File) error {
    return escreveCache(f, header, termos, docLengths)
}); err != nil { ... }
promoverArenaSePresente(inv)
```

`go test -race ./internal/search/ -run 'TestSaveOverwritesMappedCache' -v`
com a mutação em vigor:

```
=== RUN   TestSaveOverwritesMappedCache
    persist_test.go:107: SaveInvertedCache por cima do cache mapeado: gravando cache de busca em "C:\\Users\\jonyd\\AppData\\Local\\Temp\\TestSaveOverwritesMappedCache1439835085\\001\\inverted_cache.gob": falha ao renomear "C:\\Users\\jonyd\\AppData\\Local\\Temp\\TestSaveOverwritesMappedCache1439835085\\001\\.gobsidian-tmp-1341878549" para "C:\\Users\\jonyd\\AppData\\Local\\Temp\\TestSaveOverwritesMappedCache1439835085\\001\\inverted_cache.gob" apos 10 tentativas: rename C:\Users\jonyd\AppData\Local\Temp\TestSaveOverwritesMappedCache1439835085\001\.gobsidian-tmp-1341878549 C:\Users\jonyd\AppData\Local\Temp\TestSaveOverwritesMappedCache1439835085\001\inverted_cache.gob: Access is denied.
--- FAIL: TestSaveOverwritesMappedCache (0.14s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/search	0.872s
FAIL
```

FAIL confirmado — o rename falha com "Access is denied" enquanto o arquivo
continua mapeado, exatamente o mecanismo que o comentário do código descreve.
Mutação revertida em seguida; reconfirmado PASS nos três testes:

```
=== RUN   TestSaveAndLoadInvertedCache
--- PASS: TestSaveAndLoadInvertedCache (0.08s)
=== RUN   TestSaveOverwritesMappedCache
--- PASS: TestSaveOverwritesMappedCache (0.03s)
=== RUN   TestIndiceRecarregadoEIdenticoAoConstruido
--- PASS: TestIndiceRecarregadoEIdenticoAoConstruido (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/search	(cached)
```

## Verificações

### Grep do Step 4 (encaminhadores) — esperado vazio

```
$ grep -rn "writer\.\(WriteAtomic\|SweepStaleTempFiles\|TempFilePrefix\|SweepResult\)" --include=*.go .
(sem saída, exit code 1)
```

### Grep N2 (achado da revisão da Task 171) — esperado só `vault/atomic.go` + fixtures de `walk_test.go`

```
$ grep -rn '\.gobsidian-tmp-' --include=*.go internal cmd
internal/vault/atomic.go:14:const TempFilePrefix = ".gobsidian-tmp-"
internal/vault/walk_test.go:57:	writeFile(t, root, ".gobsidian-tmp-abc123.md", "escrita atomica em andamento")
internal/vault/walk_test.go:121:	writeFile(t, root, ".gobsidian-tmp-abc123.md", "escrita atomica interrompida")
```

### go build / go vet / testes dos pacotes tocados

```
$ go build ./...
(limpo)

$ go vet ./...
(limpo)

$ go test ./internal/writer/... ./internal/index/... ./internal/search/... ./internal/service/... ./cmd/gobsidian/...
ok  	github.com/jonyd/gobsidian/internal/writer	1.965s
ok  	github.com/jonyd/gobsidian/internal/index	3.781s
ok  	github.com/jonyd/gobsidian/internal/search	8.227s
ok  	github.com/jonyd/gobsidian/internal/service	21.245s
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	2.872s
```

### go list — nenhuma aresta nova para `writer`

```
$ GOOS=windows go list -f '{{.Imports}}' ./internal/index/ ./internal/search/
[bufio cmp context encoding/binary errors fmt github.com/cespare/xxhash/v2 github.com/jonyd/gobsidian/internal/parser github.com/jonyd/gobsidian/internal/text github.com/jonyd/gobsidian/internal/vault golang.org/x/sync/errgroup hash/crc32 io log/slog math os path path/filepath reflect runtime slices sort strings sync sync/atomic time unicode/utf8]
[bufio container/list context encoding/binary errors fmt github.com/jonyd/gobsidian/internal/index github.com/jonyd/gobsidian/internal/parser github.com/jonyd/gobsidian/internal/text github.com/jonyd/gobsidian/internal/vault io math os path/filepath sort strings sync sync/atomic syscall unicode unicode/utf8 unsafe]
```

Nenhuma das duas linhas contém `writer`.

### Benchstat (Step 5)

Binários `antes172_*` (commit base `011042f`, via `git worktree` detached —
nunca toquei a árvore de trabalho) × `depois172_*` (árvore editada),
intercalados, `-test.count=1` por rodada × 7 rodadas, contra
`%TEMP%\vault_5000`:

```
$ benchstat antes172_index.txt depois172_index.txt
goos: windows
goarch: amd64
pkg: github.com/jonyd/gobsidian/internal/index
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
                      │ antes172_index.txt │         depois172_index.txt         │
                      │       sec/op       │    sec/op     vs base               │
SaveIndexCacheReal-12          17.58m ± 7%   22.23m ± 15%  +26.43% (p=0.001 n=7)

                      │ antes172_index.txt │        depois172_index.txt         │
                      │        B/op        │     B/op      vs base              │
SaveIndexCacheReal-12         1.165Mi ± 0%   1.166Mi ± 0%  +0.04% (p=0.001 n=7)

                      │ antes172_index.txt │        depois172_index.txt        │
                      │     allocs/op      │  allocs/op   vs base              │
SaveIndexCacheReal-12          6.097k ± 0%   6.103k ± 0%  +0.10% (p=0.001 n=7)

$ benchstat antes172_search.txt depois172_search.txt
goos: windows
goarch: amd64
pkg: github.com/jonyd/gobsidian/internal/search
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
                         │ antes172_search.txt │     depois172_search.txt      │
                         │       sec/op        │    sec/op     vs base         │
SaveInvertedCacheReal-12          154.1m ± 30%   163.5m ± 30%  ~ (p=0.097 n=7)

                         │ antes172_search.txt │     depois172_search.txt      │
                         │        B/op         │     B/op      vs base         │
SaveInvertedCacheReal-12          24.95Mi ± 0%   24.95Mi ± 0%  ~ (p=0.053 n=7)

                         │ antes172_search.txt │       depois172_search.txt        │
                         │      allocs/op      │  allocs/op   vs base              │
SaveInvertedCacheReal-12           255.0k ± 0%   255.0k ± 0%  +0.00% (p=0.001 n=7)
```

Leitura honesta: o índice sobe de forma estatisticamente significativa
(+26,43 %, p=0,001), mas fica bem ABAIXO do que a antiga coluna `ComFsync` da
Baseline sugeria (41,87 ms) — aquele benchmark mede um `Sync` manual num
caminho de escrita diferente do de produção (reabre o arquivo final depois do
rename e sincroniza de novo), não é comparável a este número. Não há indício
de custo "além do fsync": a subida fica abaixo da referência, não acima —
condição que dispensaria a investigação extra que o brief pede se ultrapassasse
a referência.

O invertido (`SaveInvertedCacheReal`) NÃO mostra diferença estatisticamente
significativa nesta rodada (p=0,097, n=7): a variância do próprio benchmark
(±30% em ambos os lados) domina o efeito do fsync sobre uma gravação de
~25 MiB. Registrado como tal em `docs/ESTADO.md` — não forcei uma leitura de
"subiu X%" que os números não sustentam.

`docs/ESTADO.md` atualizado (linhas ~115 e ~120, mais uma nota logo depois da
tabela) com estes dois pares de números e a explicação acima.

Binários e saída bruta preservados em
`%LOCALAPPDATA%\gobsidian-bench\2026-09-02\{antes,depois}172_{index,search}.test.exe`
e `.txt`.

### verify.ps1 (Step 6) — última linha

```
[OK] Bateria completa. Pode commitar.
```

Rodou as 14 etapas, sem `-SkipCross`/`-SkipNet`, exit code 0. 6 testes pulados
foram informados (não reprovam o gate): `TestAjudanteSeguraTrava`,
`TestListenRestringePermissaoUnix`, `TestSignalCancelsContext`,
`TestPerfilDeHeapServindo`, `TestWriteAtomicPreservaOModoDoAlvo`,
`TestNew_FailsOnUnwatchablePath` — nenhum deles referencia código tocado por
esta tarefa; são skips conhecidos de ambiente (Windows/Unix-only, perfil de
heap, sinal).

## Achados do audit_reports.ps1

Rodado às 08:06 sobre este relatório e o ledger existente. Achados
específicos deste relatório e resposta:

- `[HEDGE]` na linha do Step 6 original, por causa da aproximação de duração
  sem número exato que eu tinha escrito ali. Corrigido acima: início 07:44,
  fim 08:05, duração 21 minutos — os dois timestamps já estavam no log de
  Progresso, a duração é a subtração deles, não uma estimativa.
- `[SECAO-AUSENTE]` sem seção de TDD/RED, TDD/GREEN e Mutação — corrigido
  acrescentando as seções "TDD" e "Mutação" acima com o texto e a saída que já
  existiam no Progresso, só que sem cabeçalho reconhecível pelo script.

Os demais achados (`SHA-NAO-CONFERE`, `RELATORIO-AUSENTE`, `SHA-FANTASMA`) são
todos em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`, um ledger de
um marco anterior que esta tarefa não toca e não tem autoridade para corrigir
sem contexto de quem escreveu aquelas entradas — fora do escopo da Task 172,
sinalizado aqui só para registro, não resolvido.

## Concerns

- `internal/index/bench_analise_test.go` (`BenchmarkSaveIndexCacheReal`) e
  `internal/search/bench_analise_test.go` (`BenchmarkSaveInvertedCacheReal`)
  têm comentários dizendo "como ela é hoje: CreateTemp + gob + Close + Rename,
  SEM fsync" — isso ficou desatualizado por esta tarefa: as duas funções agora
  gravam COM fsync via `vault.ReplaceFile`. As variantes `...ComFsync` dos
  mesmos arquivos (que reabrem o arquivo final e sincronizam de novo por
  fora) ficaram redundantes — hoje just fazem um segundo `Sync` em cima de um
  arquivo que o `ReplaceFile` já sincronizou. Não toquei esses arquivos porque
  não estão na lista de **Files** do brief nem nas correções do orquestrador;
  registrando para quem revisar decidir se abre uma tarefa de limpeza.
- A comparação do índice contra a antiga referência `ComFsync` (41,87 ms) não
  é maçã-com-maçã, como explicado na seção de benchstat — os dois caminhos de
  código são diferentes. Fica documentado em `docs/ESTADO.md`, mas vale
  reforçar aqui para quem for comparar números entre tarefas.
- O resultado de `SaveInvertedCacheReal` não separa o custo do fsync do ruído
  de medição nesta rodada (p=0,097). Uma rodada com mais amostras poderia
  assentar isso, mas o brief pede `-test.count=7` e foi isso que rodei;
  registrei "sem diferença estatisticamente significativa" em vez de forçar
  uma leitura otimista ou pessimista dos números.

## Fix round 1

Review em `.superpowers/sdd/2026-09-02-topografia-e-limpeza/review-172.md`,
achados N1–N5. Todos os cinco corrigidos.

- 08:18 — N1 (blocking): `docs/ESTRUTURA.md:119-121` — removidas as três
  linhas da árvore que descreviam `writer/atomic.go` (arquivo apagado no
  commit anterior). `docs/ARCHITECTURE.md:107` — parágrafo no presente
  substituído por uma frase no passado: a substituição atômica passou para
  `internal/vault` (§2.5) na Task 171; os encaminhadores transitórios em
  `writer/atomic.go` foram removidos na Task 172, quando o último chamador
  migrou.
- 08:19 — N2: `docs/ESTADO.md` — as duas linhas de fatos medidos ("6 sítios
  de `writer.WriteAtomic`", "4 literais de `.gobsidian-tmp-`") datadas como
  "medido em 2026-09-02, antes das Tasks 171-172", com o fato de hoje
  acrescentado ao lado (zero sítios, um literal). `docs/segregacao.md` seção
  2 ("Troca atômica — três implementações") ganhou uma linha de fecho: extraído
  nas Tasks 171/172, hoje uma implementação só, `vault.ReplaceFile`.
- 08:21 — N3: comentários de `BenchmarkSaveIndexCacheReal`
  (`internal/index/bench_analise_test.go`) e `BenchmarkSaveInvertedCacheReal`
  (`internal/search/bench_analise_test.go`), que diziam "SEM fsync", trocados
  para "via `vault.ReplaceFile`, com fsync do arquivo e do diretorio (Task
  172)". Os pares `...ComFsync` ganharam comentário dizendo que hoje somam um
  SEGUNDO `Sync` redundante e sobrevivem só como referência histórica —
  mantidos, não apagados, como o pedido especificava.
- 08:22 — N4 (ruling do orquestrador, sem mudança de código): acrescentada
  uma frase à seção "## Formato de cache" de `docs/ESTADO.md` registrando que,
  desde a Task 172, os arquivos de cache nascem `0644` via `vault.ReplaceFile`
  — mesma postura do diretório de cache (`0755`, `os.UserCacheDir`) — e que
  isso é aceito, não uma regressão a corrigir.
- 08:23 — N5: `internal/search/persist.go:71-72` — segunda frase do
  comentário da promoção trocada pela redação do revisor: `ExportForCache` já
  copia as posições, então a ordem aqui é sobre o rename, não sobre aliasing
  (a frase antiga insinuava um perigo de aliasing que não existe).
- 08:23 — Verificações: `gofmt -l ./internal` vazio; `go vet ./...` limpo;
  `go test -race -count=1 ./internal/index/ ./internal/search/` → `ok` nos
  dois pacotes (saída colada abaixo); os quatro `.md` tocados
  (`ESTRUTURA.md`, `ARCHITECTURE.md`, `ESTADO.md`, `segregacao.md`) validados
  como UTF-8 com `python -c "open('X',encoding='utf-8').read()"` →
  `[OK]` nos quatro.
- 08:24 — `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` rodado em
  PRIMEIRO PLANO (não em background desta vez) → `[OK] Bateria completa.
  Pode commitar.` Última linha colada abaixo. Os mesmos 6 skips de sempre,
  nenhum deles em código tocado por este round.
- 08:24 — Commit criado com `git commit -F commit-172-fix.txt`, staging por
  caminho explícito: `docs/ARCHITECTURE.md`, `docs/ESTADO.md`,
  `docs/ESTRUTURA.md`, `docs/segregacao.md`,
  `internal/index/bench_analise_test.go`,
  `internal/search/bench_analise_test.go`, `internal/search/persist.go` — 7
  arquivos (a mensagem original do orquestrador dizia "os 6 arquivos acima";
  contei 7 caminhos distintos entre N1–N5, incluindo `persist.go` do N5;
  sinalizando a diferença, não silenciando). SHA
  `68c0bc550043a2c2ec820e33cf2bf5599e15d26e`. Trabalho não commitado do dono
  permaneceu intocado (`git status --short` confirmado depois).

### go test -race (Fix round 1)

```
$ go test -race -count=1 ./internal/index/ ./internal/search/
ok  	github.com/jonyd/gobsidian/internal/index	3.628s
ok  	github.com/jonyd/gobsidian/internal/search	8.249s
```

### verify.ps1 -SkipCross -SkipNet (Fix round 1) — última linha

```
[OK] Bateria completa. Pode commitar.
```

Rodou 11 das 14 etapas normais (vet cruzado e `check_net` pulados de
propósito pela flag), exit code 0, em primeiro plano.
