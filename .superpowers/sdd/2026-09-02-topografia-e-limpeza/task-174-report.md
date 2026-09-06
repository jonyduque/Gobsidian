# Task 174 — `internal/boot`: abrir índice, preparar busca, montar componentes

**Status:** DONE_WITH_CONCERNS
**SHA:** `eac13c92891159e91159f99d19d04097c136c87e`

## Progresso

- 09:09 — brief, `CLAUDE.md`, `docs/papeis/implementador.md`, `servico.go`, `serve.go`, `daemon.go` e os dois testes a mover lidos; assinaturas de `index`, `search`, `service`, `vaulttest` conferidas.
- 09:10 — binário "antes" construído da árvore pré-edição em `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\antes174_gobsidian.exe` (Step 7 exige o par antes/depois).
- 09:12 — Step 1: os três arquivos de teste novos escritos; `go test ./internal/boot/` reprova por compilação (saída colada abaixo).
- 09:14 — Step 2: pacote criado (`doc.go`, `indice.go`, `busca.go`, `montar.go`), corpos movidos.
- 09:15 — Step 3: `git mv` dos dois testes, `git rm servico.go`, `serve.go` e `daemon.go` chamando `boot.Montar`; build, vet, gofmt limpos; 11 testes PASS sob `-race` em `./internal/boot/`; `./cmd/... ./internal/daemon/ ./internal/mcpsrv/` PASS.
- 09:16 — Step 4: prova de mutação exit 0 (âncora única em `busca.go:201`).
- 09:17 — Step 5: `go list` dá exatamente as seis arestas; nenhum pacote de domínio importa `boot`.
- 09:18 — Step 6: `CLAUDE.md` (árvore + grafo + lista de importadores de `vaulttest`), `docs/ESTRUTURA.md`, `docs/ARCHITECTURE.md` §2.1 e §2.14; os três validados como UTF-8.
- 09:19 — Step 7: sete execuções de `measure.ps1` (4 antes / 3 depois, intercaladas).
- 09:20 — linha `servidor pronto` real capturada de `serve`.
- 09:24 — Step 8: `verify.ps1` reprovou em `check_doc_refs` (`docs/OPERACAO.md:1525` cita `cmd/gobsidian/servico.go`, que a Task apagou). Dispensa por linha acrescentada com o motivo histórico.
- 09:28 — Step 8: `verify.ps1` verde, 14 etapas.
- 09:30 — commit `eac13c9`.
- 09:35 — relatório fechado; `audit_reports.ps1 174` rodado e respondido no fim.

---

## Status

**DONE_WITH_CONCERNS** — tudo o que o brief pede está feito e o gate está verde.
As ressalvas são desvios deliberados do **código ilustrativo** do brief, mais
três achados fora da lista de **Files** que não toquei. Estão em `## Concerns`.

**SHA:** `eac13c92891159e91159f99d19d04097c136c87e`

---

## Step 1 (RED) — os testes novos falham antes de o pacote existir

```
$ go test ./internal/boot/
github.com/jonyd/gobsidian/internal/boot: no non-test Go files in C:\Users\jonyd\Projetos\Gobsidian\internal\boot
FAIL	github.com/jonyd/gobsidian/internal/boot [build failed]
FAIL
```

---

## Step 3 (GREEN) — build, vet e testes

```
$ go build ./... && go vet ./...
BUILD+VET OK

$ gofmt -l cmd/gobsidian internal/boot
GOFMT CLEAN
```

`go test -race ./internal/boot/ -v` — os 8 testes novos mais os 3 que vieram
junto com os arquivos movidos:

```
=== RUN   TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem
--- PASS: TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem (0.03s)
=== RUN   TestInvertedCacheState
=== RUN   TestInvertedCacheState/sem_cache_no_disco
=== RUN   TestInvertedCacheState/cache_corrompido
=== RUN   TestInvertedCacheState/versao_divergente_traz_cabecalho,_mas_nao_serve
=== RUN   TestInvertedCacheState/cobertura_exata
=== RUN   TestInvertedCacheState/parcial:_menos_notas_no_cache_do_que_no_cofre
=== RUN   TestInvertedCacheState/parcial_por_uma_nota_so
=== RUN   TestInvertedCacheState/cache_maior_que_o_cofre,_apos_apagar_notas
=== RUN   TestInvertedCacheState/cofre_vazio
--- PASS: TestInvertedCacheState (0.00s)
    --- PASS: TestInvertedCacheState/sem_cache_no_disco (0.00s)
    --- PASS: TestInvertedCacheState/cache_corrompido (0.00s)
    --- PASS: TestInvertedCacheState/versao_divergente_traz_cabecalho,_mas_nao_serve (0.00s)
    --- PASS: TestInvertedCacheState/cobertura_exata (0.00s)
    --- PASS: TestInvertedCacheState/parcial:_menos_notas_no_cache_do_que_no_cofre (0.00s)
    --- PASS: TestInvertedCacheState/parcial_por_uma_nota_so (0.00s)
    --- PASS: TestInvertedCacheState/cache_maior_que_o_cofre,_apos_apagar_notas (0.00s)
    --- PASS: TestInvertedCacheState/cofre_vazio (0.00s)
=== RUN   TestInvertedCacheStateErroNuncaEPronta
--- PASS: TestInvertedCacheStateErroNuncaEPronta (0.00s)
=== RUN   TestPrepararBuscaSemCacheConstroiEMarcaPronta
--- PASS: TestPrepararBuscaSemCacheConstroiEMarcaPronta (0.03s)
=== RUN   TestPrepararBuscaComCacheCompletoAdota
--- PASS: TestPrepararBuscaComCacheCompletoAdota (0.03s)
=== RUN   TestPrepararBuscaCtxCanceladoNaoMarcaPronta
--- PASS: TestPrepararBuscaCtxCanceladoNaoMarcaPronta (0.02s)
=== RUN   TestAbrirIndiceSemCacheConstroiEGrava
--- PASS: TestAbrirIndiceSemCacheConstroiEGrava (0.02s)
=== RUN   TestAbrirIndiceComCacheFrescoCarrega
--- PASS: TestAbrirIndiceComCacheFrescoCarrega (0.03s)
=== RUN   TestAbrirIndiceComCacheVelhoReconstroi
--- PASS: TestAbrirIndiceComCacheVelhoReconstroi (0.03s)
=== RUN   TestAbrirIndiceComCacheCorrompidoReconstroi
--- PASS: TestAbrirIndiceComCacheCorrompidoReconstroi (0.02s)
=== RUN   TestMontarDevolveServicoPronto
--- PASS: TestMontarDevolveServicoPronto (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/boot	1.896s
```

```
$ go test -race ./cmd/... ./internal/daemon/ ./internal/mcpsrv/
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	3.634s
ok  	github.com/jonyd/gobsidian/internal/daemon	(cached)
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	(cached)
```

`(cached)` em `daemon` e `mcpsrv` é legítimo — nenhum dos dois importa `boot` —,
mas "cache válido" não é o mesmo que "rodou". A etapa 2 do `verify.ps1` também é
`go test -race -v` **sem** `-count=1`, então ela não desfaz a dúvida. Rodei de
novo forçando execução crua, e é esta a evidência que vale:

```
$ go test -count=1 -race ./cmd/... ./internal/daemon/ ./internal/mcpsrv/
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	5.165s
ok  	github.com/jonyd/gobsidian/internal/daemon	5.113s
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	6.634s
```

---

## Step 4 — prova de mutação

A âncora ocorre **uma vez só** no arquivo, então não precisou ser ampliada:

```
$ grep -c "if ctx.Err() != nil {" internal/boot/busca.go
1
$ grep -n "if ctx.Err() != nil {" internal/boot/*.go
internal/boot/busca.go:201:		if ctx.Err() != nil {
```

`internal/boot/busca.go:201` é o laço de `construirBusca` — a linha seguinte é o
comentário `// Sai sem gravar, de proposito.`, exatamente o guarda que a Task
manda mutar. `mutate.ps1` recusa âncora ambígua (sai com 2 se a contagem não for
1), então a contagem acima também é a garantia de que a mutação atingiu o alvo
certo. Foi por isso que a âncora curta do brief serviu como está.

```
$ pwsh -File scripts/mutate.ps1 -Path internal/boot/busca.go -Anchor 'if ctx.Err() != nil {' -Replacement 'if false {' -Test TestPrepararBuscaCtxCanceladoNaoMarcaPronta -Package ./internal/boot/
[...] Mutando internal/boot/busca.go
      - if ctx.Err() != nil {
      + if false {

[...] go test -race -run TestPrepararBuscaCtxCanceladoNaoMarcaPronta ./internal/boot/
----------------------------------------------------------------------
--- FAIL: TestPrepararBuscaCtxCanceladoNaoMarcaPronta (0.04s)
    busca_test.go:79: ctx cancelado antes de construir: o indice de busca nao pode ser marcado Ready
FAIL
FAIL	github.com/jonyd/gobsidian/internal/boot	0.817s
FAIL
----------------------------------------------------------------------
[OK] internal/boot/busca.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
[i] Cole a saida acima no relatorio. Prova de mutacao no condicional
    ('se removermos X, o teste falha') nao conta: uma delas ja estava errada.
EXIT=0
```

---

## Step 5 — `go list`

```
$ go list -f '{{.ImportPath}} {{.Imports}}' ./internal/boot/ | tr ' ' '\n' | grep gobsidian
github.com/jonyd/gobsidian/internal/boot
github.com/jonyd/gobsidian/internal/config
github.com/jonyd/gobsidian/internal/index
github.com/jonyd/gobsidian/internal/search
github.com/jonyd/gobsidian/internal/service
github.com/jonyd/gobsidian/internal/vault
github.com/jonyd/gobsidian/internal/watcher
```

Exatamente `config index search service vault watcher`. Nenhum `mcpsrv`,
nenhum `lifecycle`, nenhum `ipc`, nenhum `daemon`, nenhum `doctor`, nenhum `net`
— a etapa 11 do `verify.ps1` (`check_net`, RNF-30) também ficou `[OK]`.

```
$ go list -f '{{.Imports}}' ./internal/index/ ./internal/search/ ./internal/service/ ./internal/watcher/ | grep -c boot
0
```

Em teste, `boot` ganha só `vaulttest`:

```
$ go list -f '{{.TestImports}} {{.XTestImports}}' ./internal/boot/ | tr ' ' '\n' | grep gobsidian | sort -u
github.com/jonyd/gobsidian/internal/boot
github.com/jonyd/gobsidian/internal/config
github.com/jonyd/gobsidian/internal/index
github.com/jonyd/gobsidian/internal/search
github.com/jonyd/gobsidian/internal/vault
github.com/jonyd/gobsidian/internal/vaulttest
```

E quem importa `boot` é um pacote só:

```
$ go list -f '{{.ImportPath}}: {{.Imports}}' ./... | grep "internal/boot" | grep -v "^github.com/jonyd/gobsidian/internal/boot:"
github.com/jonyd/gobsidian/cmd/gobsidian: [... github.com/jonyd/gobsidian/internal/boot ...]
```

**`scripts/test_orphans.ps1` NÃO foi rodado nesta tarefa** — instrução explícita
do orquestrador, que o roda em separado. Não é "não medido por descuido": é
escopo que não me pertence, e por isso não há quatro `[OK]` para colar aqui.

---

## Step 7 — medição

`scripts/measure.ps1 -Vault %TEMP%\vault_5000 -Binary <exe>`, execuções
intercaladas (antes/depois), o `RNF-01 indice` de cada uma. `origem do indice:
cache` nas sete — todas com o cache de índice quente.

| # | binário | `index_ms` |
|---|---|---|
| 1 | antes  | **394** |
| 2 | depois | 89 |
| 3 | antes  | 93 |
| 4 | depois | 97 |
| 5 | antes  | 86 |
| 6 | depois | 91 |
| 7 | antes  | 128 |

A execução 1 é a primeira do dia contra esse cofre e mediu o cache de página do
SO frio, não o código: 394 ms contra 86–128 ms em tudo o que veio depois, com o
mesmo binário. Descartá-la seria conveniente demais para eu fazer em silêncio,
então ela fica na tabela — mas a comparação que responde à pergunta da Task usa
as três de cada braço em estado comparável:

- antes  (execuções 3, 5, 7): 93, 86, 128 ms — mediana **93 ms**
- depois (execuções 2, 4, 6): 89, 97, 91 ms — mediana **91 ms**

As faixas se sobrepõem inteiras (86–128 contra 89–97), e a mediana do "depois"
é 2 ms menor que a do "antes" — diferença sem significado com n=3. Dentro do
ruído de M4 (101–123 ms citado no brief). O boot não ficou mais lento por ter
mudado de pacote, e nada indica que tenha ficado mais rápido.

**Não medi wall clock de `servidor pronto`**: `measure.ps1` não emite esse
número, só o `index_ms` que o próprio servidor loga. Não vou escrever um número
que não medi.

Binário "antes": construído da árvore pré-edição (base `7119645`) em
`%LOCALAPPDATA%\gobsidian-bench\2026-09-02\antes174_gobsidian.exe`, **antes** de
qualquer edição. Binário "depois": `bin\gobsidian.exe` de `scripts/build.ps1`.

---

## Linha `servidor pronto` real

De um `serve` de verdade contra o cofre de 5000 notas. `GOBSIDIAN_NO_DAEMON=1`
porque, sem ela, `serve` encontra o daemon e vira ponte — quem loga a linha é o
outro processo. É o que `measure.ps1` faz, e documenta o motivo.

```
time=2026-09-06T09:20:00.216-03:00 level=INFO msg="servidor pronto" vault=C:\Users\jonyd\AppData\Local\Temp\vault_5000 read_only=false notes=5000 assets=50 index_ms=76 index_origin=cache
```

As seis chaves são as mesmas de antes — `vault`, `read_only`, `notes`,
`assets`, `index_ms`, `index_origin` — e `index_origin` continua saindo com os
literais `"cache"` e `"build"`, agora produzidos por `AbrirIndice`. Conferi no
código antigo antes de mover, como o brief manda: `servico.go` usava exatamente
esses dois, não `"cached"`/`"built"`. A regex de `measure.ps1`
(`index_ms=(\d+)`) continua casando — as sete medições do Step 7 saíram dela.

---

## Step 8 — o gate

Última linha de `pwsh -File scripts/verify.ps1` (14 etapas, primeiro plano):

```
[OK] Bateria completa. Pode commitar.
```

Exit code 0, as 14 etapas em `[OK]`. A etapa 3 (contagem de pulados) informa
`[!] 6 testes pulados` e não reprova — são os mesmos seis do gate da Task 173,
nome a nome (`TestAjudanteSeguraTrava`, `TestListenRestringePermissaoUnix`,
`TestSignalCancelsContext`, `TestPerfilDeHeapServindo`,
`TestWriteAtomicPreservaOModoDoAlvo`, `TestNew_FailsOnUnwatchablePath`).
Nenhum teste novo pula: os 8 do `boot` rodam.

**A primeira rodada do gate reprovou**, e fica registrada aqui porque a correção
saiu da lista de **Files** do brief:

```
[...] 13. check_doc_refs
[!] check_doc_refs
     [i] corpus: 363 arquivos .go.
       docs\OPERACAO.md:1525: [ARQUIVO] `cmd/gobsidian/servico.go`
```

`docs/OPERACAO.md:1525` registra uma medição de 2026-08-28 que editou
`cmd/gobsidian/servico.go` — arquivo que esta Task apagou. O fato é histórico e
verdadeiro, então não reescrevi a história: a frase passou a dizer onde o
arquivo está hoje (`internal/boot/montar.go`) e ganhou a dispensa por linha que
o próprio checador documenta, com o motivo obrigatório. Depois disso:

```
[i] 37 dispensa(s) em uso -- nao contam como achado:
  docs\OPERACAO.md:1525: `cmd/gobsidian/servico.go` -- registro historico de 2026-08-28: o arquivo existia entao, e a Task 174 o moveu para internal/boot/montar.go
```

`docs/OPERACAO.md` entrou no commit por isso, e é o **único** caminho fora da
lista do brief. Nada da árvore não-commitada do dono (`test-vault/`,
`.claude/skills/`, `Resume-Claude.ps1`, o ledger) foi staged: o `git add` foi
caminho a caminho, e o `git show --name-status` do commit lista 15 arquivos,
todos deles.

---

## Nomes antigos → novos

| Antes | Depois |
|---|---|
| `cmd/gobsidian/servico.go:construirServico` | `internal/boot/montar.go:Montar` |
| `cmd/gobsidian/servico.go:servicoMontado` | `internal/boot/montar.go:Componentes` |
| &nbsp;&nbsp;campo `svc` | `Service` |
| &nbsp;&nbsp;campo `w` | `Watcher` |
| &nbsp;&nbsp;campo `wg` (`*sync.WaitGroup`) | `espera` (`sync.WaitGroup`), alcançado por `Esperar()` |
| — (nova) | `internal/boot/indice.go:AbrirIndice` |
| — (novos) | `Componentes.Vault`, `Componentes.Index`, `Componentes.Inverted` |
| `cmd/gobsidian/serve.go:carregarIndiceDoCache` | `internal/boot/indice.go:carregarIndiceDoCache` (nome mantido) |
| `cmd/gobsidian/serve.go:invertedSaveInterval` | `internal/boot/busca.go:intervaloDeGravacao` |
| `cmd/gobsidian/serve.go:invertedCacheState` | `internal/boot/busca.go:estadoDoCache` |
| `cmd/gobsidian/serve.go:devolveMemoriaTransitoria` | `internal/boot/busca.go:devolveMemoriaTransitoria` (nome mantido) |
| `cmd/gobsidian/serve.go:prepararIndiceDeBusca` | `internal/boot/busca.go:PrepararBusca` |
| `cmd/gobsidian/serve.go:buildInvertedIndex` | `internal/boot/busca.go:construirBusca` |
| `cmd/gobsidian/serve.go:watcherStats` | `internal/boot/montar.go:watcherStats` |
| `cmd/gobsidian/inverted_cache_state_test.go` | `internal/boot/estado_do_cache_test.go` |
| `cmd/gobsidian/boot_indice_busca_windows_test.go` | `internal/boot/busca_windows_test.go` |

Os dois arquivos de teste movidos mudaram de `package main` para `package boot`
(caixa-branca, é o que os dois exigem) e a build tag `//go:build windows` de
`busca_windows_test.go` continua na primeira linha.

---

## Concerns

**1. Segui o corpo de `servico.go`, não o código ilustrativo do brief, em dois
pontos de `AbrirIndice`.** O bloco do Step 2 do brief escreve
`fmt.Errorf("construindo indice: %w", err)` e
`log.Warn("nao foi possivel gravar o cache de indice", ...)`. O original tem
`return nil, err` sem embrulho e
`log.Warn("falha ao salvar cache de indice de metadados", ...)`. Mantive os dois
originais. Razão: a regra de execução do próprio brief é "movimento, não
reescrita: corpos verbatim, só nomes e assinaturas mudam", e o mesmo brief, ao
tratar de `index_origin`, manda explicitamente conferir o código e **manter o
antigo** se o literal divergir do que ele escreveu. Texto de log é comportamento
observável, e embrulhar o erro muda a mensagem que
`log.Error("nao foi possivel montar o servico", "err", err)` imprime. Se a
intenção era mesmo trocar os dois, é uma linha em cada ponto — diga e eu troco.

**2. Tirei `(ver Task 177 para a vigilia do host)` do `doc.go`.** O brief traz
essa frase. Task 177 não existe ainda, e quem a procurar não acha nada; "não
deixe sua deliberação no código". O resto do `doc.go` está literal.

**3. Um comentário mudou de texto além do nome, dentro de `construirBusca`.** O
original dizia "`runServe` faz `wg.Wait()` DEPOIS de `lifecycle.Shutdown`". `wg`
não existe mais neste pacote, e `runServe` já era o nome errado antes da Task —
quem espera é `serveEmProcesso`. Ficou: "`serveEmProcesso` faz a espera das
goroutines de fundo DEPOIS de `lifecycle.Shutdown`". Mesmo fato, referência que
resolve.

**4. Não renomeei as duas funções de teste que vieram nos arquivos movidos.**
`TestInvertedCacheState` e `TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem`
nomeiam símbolos que agora se chamam `estadoDoCache` e `construirBusca`. O brief
não pede o rename, e os dois nomes aparecem em `task-128-report.md` e em
`docs/superpowers/plans/2026-08-16-revisao-fixes.md` como registro de provas de
mutação passadas — renomear quebraria a rastreabilidade desses relatórios.
**Sugestão** para uma Task futura: renomear junto com uma nota nos dois
documentos.

**5. Comentários de outros pacotes ainda apontam para `cmd/gobsidian/serve.go`.**
Todos fora da lista de **Files**, todos em comentário (nenhum em código):
`internal/index/persist.go:65`, `internal/search/inverted.go:86,:601,:615`,
`internal/search/mmap.go:158-159`, `internal/search/persist_test.go:54,:99`,
`internal/search/update_bench_test.go:16`,
`internal/search/cloudonly_update_windows_test.go:88,:146,:171,:175` e
`internal/service/search_lazy_test.go:25` citam `invertedCacheState`,
`buildInvertedIndex` ou `prepararIndiceDeBusca` "em `cmd/gobsidian/serve.go`".
Nenhum quebra compilação, e `check_doc_refs` não olha `.go`, então o gate não os
vê. **Não toquei** — escopo não encolhe em silêncio, mas também não cresce em
silêncio. Um `sed` guiado resolve numa Task de limpeza.

**6. `docs/wiki/` ficou desatualizado.** `docs/wiki/Home.md:39`,
`docs/wiki/flows/boot.md:8,:29` e `docs/wiki/decisions/decisoes-fechadas.md:10`
citam `cmd/gobsidian/servico.go` e `construirServico`;
`docs/wiki/_wiki/manifest.json:29` guarda o hash do arquivo apagado. O wiki é
documentação **derivada** e tem mecanismo próprio de `status: stale`; o brief
não o lista e `check_doc_refs` não varre `docs/wiki/`. Fica registrado para a
próxima ingestão.

**7. `§2.14` de `docs/ARCHITECTURE.md` está fora de ordem, de propósito.** A
montagem é camada acima de `service` e abaixo de `cmd`/`mcpsrv`, mas entrou
numerada como 2.14, no fim de §2. Renumerar §2.2–§2.13 quebraria as referências
cruzadas que o resto da documentação faz a esses números. O parágrafo diz isso
explicitamente, para ninguém "consertar" a ordem sem enxergar o custo.

**8. `Componentes` guarda `sync.WaitGroup` por valor.** É o que o brief
especifica, e está correto: `Montar` devolve `*Componentes` e nada copia a
struct. Vale saber que a proteção é automática — `copylocks` faz parte do
`go vet` padrão, e as etapas 5, 6 e 7 do gate (vet nos três GOOS) estão verdes,
então uma cópia futura vira erro de gate, não defeito silencioso.

---

## Resposta a `scripts/audit_reports.ps1 174`

O auditor acusou 16 achados. **Dois são deste relatório; os outros catorze são
do ledger de um marco anterior e não foram tocados por esta Task.**

**Deste relatório (2):**

- `[SECAO-AUSENTE] sem secao de TDD/RED` e `[SECAO-AUSENTE] sem secao de
  TDD/GREEN`. **Procedentes na forma, não no conteúdo.** As duas fases existiam
  e estavam coladas — Step 1 é o RED (o `FAIL [build failed]` com o pacote
  ainda inexistente) e Step 3 é o GREEN (os 11 PASS sob `-race`) —, mas os
  títulos estavam só em português e o auditor procura as palavras `RED` e
  `GREEN` no corpo. Renomeei os dois títulos para `Step 1 (RED)` e
  `Step 3 (GREEN)`. Não acrescentei conteúdo para calar o checador: as saídas
  coladas são as mesmas de antes do rename.

Nenhum achado de `HEDGE`, `NAO-RESPOSTA` ou `MUTACAO-CONDICIONAL` no relatório —
a prova de mutação está no passado, com a saída e o exit code colados, e não no
condicional.

**Do ledger (14), todos anteriores a esta Task:**
`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` acumula dois
`[SHA-NAO-CONFERE]` (Tasks 4 e 6), nove `[RELATORIO-AUSENTE]` (Tasks 94–103),
um `[SHA-FANTASMA]` (`deadbee`, na linha 988) e mais um `[SHA-NAO-CONFERE]`
(Task 153). Nenhum deles cita Task 174, nenhum arquivo do ledger entrou no meu
commit, e corrigir histórico de outras tarefas não é escopo desta. Ficam aqui
registrados para quem cuida do ledger.

---

## Round 1 — as cinco correcoes de documentacao da revisao

### Progresso

- 09:44 — li `.superpowers/sdd/2026-09-02-topografia-e-limpeza/review-174.md` pelo recado do orquestrador e localizei os cinco pontos: `CLAUDE.md:88`, `docs/ESTRUTURA.md:153`, `docs/ESTRUTURA.md:235`, `internal/boot/doc.go:3`, `docs/ARCHITECTURE.md:127`, `:129` e `:568`.
- 09:44 — medi as duas afirmacoes que a revisao me pede para corrigir, em vez de copiar a redacao dela:
  - quem importa `boot`: `grep -rn "index.New()\|boot\." cmd/gobsidian/*.go | grep -v _test` devolve `daemon.go:112`, `daemon.go:159`, `serve.go:103`, `serve.go:107` para `boot.` e `index.go:42`, `inspect.go:47`, `search.go:38` para `index.New()`. Confirma N1: hoje so `serve` e `daemon` chamam `boot`; `index`, `search` e `inspect` montam o indice por conta propria.
  - os passos de §5.1 (N4): a lista em `docs/ARCHITECTURE.md:321-339` tem 8 passos; 1 e 2 sao flags e `lifecycle`, 8 e o servidor MCP em stdio. `Montar` cobre 3 a 7 — cache/varredura (3-4), backlinks (5), indice de busca (6), watcher (7). "3 a 7" confere.
- 09:45 — apliquei as cinco edicoes por script Python com `assert` de ocorrencia unica em cada ancora (nenhuma substituicao cega).
- 09:45 — rodei os checadores permitidos nesta rodada (o gate de orfaos estava rodando na maquina; nao rodei `go test`, `verify.ps1`, `build.ps1` nem nada que suba o gobsidian).
- 09:46 — commit `33cff51f03eb58c51e324011316af5267851450d`, so os quatro arquivos, por caminho explicito.

### SHA

```
33cff51f03eb58c51e324011316af5267851450d
docs(boot): say who calls boot today, qualify the eager-only ordering, name Montar in §7.5
 4 files changed, 11 insertions(+), 10 deletions(-)
```

Arquivos no commit: `CLAUDE.md`, `docs/ARCHITECTURE.md`, `docs/ESTRUTURA.md`, `internal/boot/doc.go`. Nenhum outro — `git status --short` dos quatro, antes do commit, mostrava exatamente `M  CLAUDE.md`, `M  docs/ARCHITECTURE.md`, `M  docs/ESTRUTURA.md`, `M  internal/boot/doc.go`.

### Saida dos checadores

```
$ gofmt -l ./internal/boot/
GOFMT_OK
$ go vet ./internal/boot/
VET_OK
```

(`GOFMT_OK` e `VET_OK` sao os `echo` encadeados com `&&`: `gofmt -l` nao listou arquivo nenhum e `go vet` saiu 0.)

`verify.ps1` etapa 13 chama `scripts/check_doc_refs.ps1` (`scripts/verify.ps1:252`). Rodado direto:

```
$ pwsh -File scripts/check_doc_refs.ps1
[i] corpus: 363 arquivos .go.
[i] 37 dispensa(s) em uso -- nao contam como achado:
  ...
  docs\OPERACAO.md:1525: `cmd/gobsidian/servico.go` -- registro historico de 2026-08-28: o arquivo existia entao, e a Task 174 o moveu para internal/boot/montar.go
  ...
[OK] nenhum token entre crases parece citar artefato ausente do codigo.
EXIT=0
```

UTF-8 dos tres `.md` editados:

```
$ for f in CLAUDE.md docs/ESTRUTURA.md docs/ARCHITECTURE.md; do python -c "open('$f',encoding='utf-8').read()" && echo "[OK] UTF-8 valido: $f"; done
[OK] UTF-8 valido: CLAUDE.md
[OK] UTF-8 valido: docs/ESTRUTURA.md
[OK] UTF-8 valido: docs/ARCHITECTURE.md
```

Nao rodei `verify.ps1` nem `go test`: o gate de orfaos estava medindo ciclos de processo nesta maquina. O orquestrador roda `verify.ps1` sobre este commit depois que o gate terminar.

### O que mudou, achado a achado

**N1 — "serve, daemon e CLI chamam" era falso.** Tres lugares:

- `CLAUDE.md:88` — `serve, daemon` / `e CLI chamam` virou `serve e` / `daemon chamam`.
- `docs/ESTRUTURA.md:153` — `a sequencia de boot que serve, daemon e CLI` virou `a sequencia de boot que serve e daemon` (a linha seguinte, `compartilham; nao importa mcpsrv nem lifecycle`, continua igual).
- `internal/boot/doc.go:3-6` — passou a dizer que `serve` e `daemon` precisam da mesma montagem e que os subcomandos de CLI nao passam por aqui: `index`, `search` e `inspect` montam o indice por conta propria, sem watcher e sem Service. Sem referencia a tarefa futura, como pedido.

**N2 — `construirServico` em §7.5.** `docs/ARCHITECTURE.md:568`: `(construirServico, compartilhada com o boot em processo ...)` virou `(boot.Montar, em internal/boot, compartilhada com o boot em processo ...)`. O resto da frase esta intacto.

**N3 — a ordenacao so vale no modo eager.** `docs/ARCHITECTURE.md:129`. A frase antiga afirmava sem qualificar:

> A ordem nao e livre: `watcher.New` registra os watches **antes** da construcao do indice de busca, e `w.Run` so consome a fila **depois** dela, porque a adocao do cache substitui o conteudo do indice invertido (§6).

A nova separa os dois modos exatamente como `internal/boot/montar.go:170-186` ja separa: `watcher.New` antes da construcao nos dois modos; `w.Run` depois dela so no eager; no modo padrao (lazy) `Montar` parte `w.Run` de imediato e a busca so e carregada na primeira `vault_search` por `opts.CarregarBusca`, e se o watcher ja tiver escrito no indice invertido, `AdotarDe` recusa o cache e `PrepararBusca` cai para a construcao, que conta o que o watcher escreveu via `HasDoc` — mais lento, nunca incorreto. A frase cita o achado A6 e `docs/SUGESTOES.md`, para o proximo leitor nao reintroduzir a versao curta.

**N4 — "o fluxo de §5.1 inteiro".** Mesmo paragrafo, `docs/ARCHITECTURE.md:127`: virou `executa os passos 3 a 7 de §5.1`. Conferido contra a lista de 8 passos em §5.1 (ver Progresso, 09:44).

**N5 — `cmd/gobsidian` nao constroi mais o `Service`.** `docs/ESTRUTURA.md:235`: `Analisa flags, monta configuracao, constroi o Service e delega.` virou `Analisa flags, monta configuracao, pede a montagem a internal/boot e delega.`

### Concerns desta rodada

Nenhum bloqueante. Um registro: a frase nova de N3 e a mais longa de §2.14, e duplica em prosa o que o comentario de `montar.go:170-186` ja explica. Mantive as duas porque a regra da casa e um fato num lugar so **por leitor** — o comentario e para quem edita `Montar`, e o §2.14 e para quem le a arquitetura sem abrir o codigo —, mas se o orquestrador preferir, a versao curta seria a arquitetura remetendo ao comentario em vez de repeti-lo.
