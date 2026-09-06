# Task 161 — Auditoria dos treze testes reportados pelo sweep

**Status:** DONE_WITH_CONCERNS (ver "Defeitos encontrados — fora de escopo")
**Base:** `50fb0bc` (o brief citava `15c66fb`; o master avancou com dois commits de docs/ledger antes desta tarefa)
**SHA do commit:** `b3bcba0`

## Progresso

- 00:14 — commit `6c1f217` do round de polimento; SHA registrado aqui.
- 00:14 — `verify.ps1` verde, catorze etapas, seis pulados de sempre. Escrevendo o commit.
- 00:08 — fix round 1: `pool_test.go` renomeado, asserção de ordem do bm25 removida com a conta no comentario, frase de alcance da linha 1 no relatorio. Re-mutacao do idf: FAIL (0). Proximo: `verify.ps1`.
- 00:07 — round de polimento (R=1) recebido; HEAD `44f2700`; tres itens nao-bloqueantes.
- 23:57 — commit `b3bcba0`, por caminho explicito, com `git commit -F`. Assunto conferido: sem `@ ` solto.
- 23:54 — `verify.ps1` verde (14/14). Escrevendo relatorio e commit.
- 23:48 — Step 2 verde: `go test -race` nos seis pacotes tocados, todos `ok`. `git diff --stat` mostra 8 arquivos, todos `_test.go`.
- 23:45 — re-mutacao contra os consertados: row1 FAIL(0), row2 FAIL(0), row4 FAIL(0), row12 FAIL(0) depois de exigir o erro de `--vault`, row13 FAIL(0) com a mutacao `filepath.Rel(filepath.Dir(root), abs)` (a mesma mutacao contra o fixture antigo hardcoded: PASS(1)), row14 FAIL(0).
- 23:41 — consertos escritos (tools_read, limites_enums, bm25, persist[apagado], pool[comentario], console_saida, filter, inverted). `gofmt -l` limpo; os pacotes tocados passam.
- 23:35 — rodada de mutacao "como esta" concluida para os treze: row7 FAIL(0), row8 FAIL(0), row9 FAIL(0), row11 confirmado removido em `b8ed7f6`, row12a FAIL(0) / row12b PASS(1) / row12c PASS(1), row13 PASS(1), row14 (inverted:118) PASS(1).
- 23:32 — rodada de mutacao "como esta" (linhas 1..6, 10): row1 PASS(1), row2 PASS(1), row3 FAIL(0), row4 PASS(1), row5 `TestCacheOutsideVault` PASS(1) / `TestLoadCacheDir` FAIL(0), row6 FAIL(0), row10 FAIL(0).
- (hora real nao registrada; o passo ocorreu antes da rodada das 23:32) — treze sitios lidos na integra (tools_read, limites_enums, server, bm25, persist, pool, write x3, slug_persistido, cli_subcommands, console_saida, filter, inverted). Proximo: ler o produto que cada um exercita. O "23:33" que estava aqui foi DIGITADO, nao lido de `date` — e por isso saia fora de ordem e adiantado. A hora verdadeira daquele momento nao e recuperavel, entao nao invento outra. Correcao feita as 23:59 (`date +%H:%M`).
- 23:24 — inicio; brief lido, `docs/papeis/testador.md` lido, HEAD = `50fb0bc`.

---

## A tabela dos treze

Codigo de saida de `scripts/mutate.ps1`: **0 = o teste REPROVOU sob mutacao** (a regra esta verificada); **1 = o teste PASSOU** (a regra estava escrita, nao verificada).

| # | Sitio | Regra de produto que o teste afirma | Mutacao aplicada | Resultado | Veredito | Acao |
|---|---|---|---|---|---|---|
| 1 | `internal/mcpsrv/tools_read_test.go:96-133` | Cada tool de leitura responde com CONTEUDO do cofre, nao so com um envelope nao-nulo | `bm25.go`: `if !math.IsNaN(score) && score > 0 {` → `if false {` | **PASS (1)** → depois do conserto **FAIL (0)** | consertar | `wantIn` por caso; `StructuredContent == nil` virou `t.Fatalf`; caso de sucesso sem `wantIn` reprova |
| 2 | `internal/service/limites_enums_test.go:45` | `note_list` aplica `ComTeto` ao `limit` ANTES de o pedido chegar ao indice | `graph.go`: `q.Limit = ComTeto(q.Limit)` → `q.Limit = req.Query.Limit` | **PASS (1)** → depois do conserto **FAIL (0)** | consertar | cofre com `LimiteTeto+1` notas; afirma `res.Total == 501` (controle) e `len(res.Notes) == 500` |
| 3 | `internal/mcpsrv/server_test.go:116` | `vault_stats` conta as notas do cofre (aqui pelo ramo `s.index == nil`, que varre com `vault.Walk`) | `graph.go`: `out.Notes++` → `_ = e` | **FAIL (0)** | falsifica, mantido | nenhuma |
| 4 | `internal/search/bm25_test.go:213` | Termo presente em TODAS as notas ainda pontua — o `1 +` do idf | `bm25.go`: `if !math.IsNaN(score) && score > 0 {` → `if score > 0 {` | **PASS (1)** → depois do conserto, com a mutacao do idf, **FAIL (0)** | consertar | teste reescrito como `TestBM25TermoEmTodasAsNotasAindaPontua` |
| 5 | `internal/search/persist_test.go:232` | (nenhuma — o teste nao chama codigo de producao) | `config.go`: `defaultCacheDir` passa a devolver um caminho DENTRO do cofre | **PASS (1)**; a mesma mutacao contra `TestLoadCacheDir` da **FAIL (0)** | apagar | `TestCacheOutsideVault` removido; a regra e falsificada por `internal/config/config_test.go` `TestLoadCacheDir/derived_path_is_not_inside_the_vault` |
| 6 | `internal/search/pool_test.go:9` | `Normalize` tira acento e baixa a caixa | `analyzer.go`: `return text.Normalize(s)` → `return s + text.Normalize("")` | **FAIL (0)** | falsifica, mantido | so o comentario, que prometia medir alocacoes e nao mede nada |
| 7 | `internal/service/write_test.go:242` | `note_append` recusa quando `expected_hash` diverge do hash do arquivo | `write.go:185-187`: o guarda de `AppendNote` → `_ = currentHash` | **FAIL (0)** | falsifica, mantido | nenhuma |
| 8 | `internal/service/write_test.go:368` | `note_patch` recusa quando `expected_hash` diverge | `write.go:287-289`: o guarda de `PatchNote` → `_ = currentHash` | **FAIL (0)** | falsifica, mantido | nenhuma |
| 9 | `internal/service/write_test.go:384` | O hash que o INDICE publica (`ListItem.Hash`) e o mesmo que o escritor aceita | `write.go`: `hashDoConteudo` passa a hashear `data` + um byte | **FAIL (0)** | falsifica, mantido | nenhuma |
| 10 | `internal/index/slug_persistido_test.go:84` | O `Slug` do heading sobrevive as tres fontes, inclusive a ida-e-volta pelo cache | `persist_codec.go:195`: `e.str(h.Slug)` → `e.str("")` | **FAIL (0)** | falsifica, mantido | nenhuma |
| 11 | `cmd/gobsidian/cli_subcommands_test.go:168` | — | — | — | ja removido em `b8ed7f6` | nenhuma; a linha 168 hoje pertence a `TestInspectCLI`, que afirma `data.Path` e `data.Backlinks` |
| 12 | `cmd/gobsidian/console_saida_test.go:100` | `serve` e `daemon` nao escrevem no writer que o cobra lhes entrega | (a) `serve.go`: `cmd.Println("diagnostico");` antes de `config.Load` — **FAIL (0)**; (b) `os.Stdout.WriteString` dentro de `runServe` — **PASS (1)**; (c) `main.go`: `newServeCmd()` fora do `AddCommand` — **PASS (1)** | **PASS (1)** nos dois eixos de vacuidade | consertar | exige erro nao-nulo E que a mensagem cite `--vault`; com isso (c) passou a dar **FAIL (0)**. O comentario agora declara o que o teste NAO cobre |
| 13 | `internal/watcher/filter_test.go:117` | Evento fora da raiz do cofre e descartado com `DropOutsideVault` | `path.go`: `rel, err := filepath.Rel(root, abs)` → `filepath.Rel(filepath.Dir(root), abs)` | **PASS (1)** com o fixture antigo; **FAIL (0)** com o consertado | consertar | `root`/`fora` viraram `t.TempDir()` |
| 14 | `internal/search/inverted_test.go:118` | `Inverted.Remove` de fato tira a nota da contagem | `inverted.go`: corpo de `Remove` → `_ = path` | **PASS (1)** → depois do conserto **FAIL (0)** | consertar | `DocCount() < 0` virou `got != 6 && got != 7`, com a derivacao no comentario |

**Alcance da prova da linha 1.** A saida de mutacao colada para a linha 1 exercita `vault_search` e so ele: a mutacao foi em `bm25.go`, entao os outros cinco casos da tabela de `tools_read_test.go` (`note_read`, `note_list`, `note_metadata`, `link_graph`, `tag_list`) nao passam por ela. Esses cinco ganharam `wantIn` e ficam cobertos pelo guarda `len(tt.wantIn) == 0`, que reprova um caso de sucesso sem trecho exigido — mas nenhum deles tem prova de mutacao colada, e este relatorio nao afirma que tem.

A tabela do brief tem treze linhas porque `write_test.go` aparece como uma so; aqui os tres sitios de `write_test.go` estao separados (linhas 7, 8, 9), o que da catorze linhas para os mesmos treze sitios do brief.

---

## Saidas coladas

### Linha 1 — `tools_read_test.go`, antes: PASS (1)

```
[...] Mutando internal/search/bm25.go
      - if !math.IsNaN(score) && score > 0 {
      + if false {

[...] go test -race -run TestReadTools ./internal/mcpsrv/
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	2.925s
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

[!] O teste PASSOU com a regra mutada.
```

### Linha 1 — depois do conserto: FAIL (0)

```
    --- FAIL: TestReadTools/vault_search_valid (0.00s)
        tools_read_test.go:204: StructuredContent nao contem "A.md":
            {"effective_limit":20,"effective_snippet_chars":240,"results":[],"total":0,"truncated":false}
        tools_read_test.go:204: StructuredContent nao contem "B.md":
            {"effective_limit":20,"effective_snippet_chars":240,"results":[],"total":0,"truncated":false}
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.264s
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

O `"results":[]` na saida e a prova direta do que o brief suspeitava: zero acerto passava.

### Linha 2 — `limites_enums_test.go`, antes: PASS (1)

```
      - q.Limit = ComTeto(q.Limit)
      + q.Limit = req.Query.Limit

[...] go test -race -run TestNoteListAplicaOTetoDeLimit ./internal/service/
ok  	github.com/jonyd/gobsidian/internal/service	1.945s
[OK] internal/service/graph.go restaurado byte a byte (SHA-256 confere).

[!] O teste PASSOU com a regra mutada.
```

### Linha 2 — depois do conserto: FAIL (0)

```
--- FAIL: TestNoteListAplicaOTetoDeLimit (0.67s)
    limites_enums_test.go:63: len(res.Notes) = 501, queria 500: o limite absurdo chegou ao índice como veio
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.461s
[OK] internal/service/graph.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### Linha 3 — `server_test.go:116`: FAIL (0), mantido

Primeira mutacao (`Notes: s.index.NoteCount(),` → `Notes: 0,`) saiu **1**, e nao porque o teste seja fraco: `newTestServer` monta `service.New(v, nil, ...)`, entao `VaultStats` cai no ramo `s.index == nil` e conta por `vault.Walk`. Mutado o ramo certo:

```
      - out.Notes++
      + _ = e

[...] go test -race -run TestVaultStatsCountsNotes ./internal/mcpsrv/
--- FAIL: TestVaultStatsCountsNotes (0.13s)
    server_test.go:117: Notes = 0, want 2
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.211s
[OK] internal/service/graph.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### Linha 4 — `bm25_test.go:213`, antes: PASS (1)

```
      - if !math.IsNaN(score) && score > 0 {
      + if score > 0 {

[...] go test -race -run TestBM25FrequentTermNoNaN ./internal/search/
ok  	github.com/jonyd/gobsidian/internal/search	1.704s
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

[!] O teste PASSOU com a regra mutada.
```

O filtro de NaN em `bm25.go:193` roda ANTES de a lista existir, entao o laco do teste nunca podia ver um NaN. Reescrito para a regra que o cenario consegue falsificar — o `1 +` do idf, que e o que faz um termo presente em todas as notas ainda pontuar:

```
      - idfs[i] = math.Log(1.0 + (N-d+0.5)/(d+0.5))
      + idfs[i] = math.Log((N-d+0.5)/(d+0.5))

[...] go test -race -run TestBM25TermoEmTodasAsNotasAindaPontua ./internal/search/
--- FAIL: TestBM25TermoEmTodasAsNotasAindaPontua (0.00s)
    bm25_test.go:232: len(res) = 0, quer 2 — termo presente em todas as notas sumiu do resultado
FAIL
FAIL	github.com/jonyd/gobsidian/internal/search	1.130s
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### Linha 5 — `persist_test.go:232`: apagado

O teste comparava dois `t.TempDir()` entre si. Nao chamava nenhuma funcao de producao. A mesma mutacao mostra os dois lados:

```
      - return filepath.Join(base, "gobsidian", VaultKey(vaultPath))
      + return filepath.Join(vaultPath, ".gobsidian-cache", VaultKey(vaultPath)) + base[:0]

[...] go test -race -run TestCacheOutsideVault ./internal/search/
ok  	github.com/jonyd/gobsidian/internal/search	1.670s
[!] O teste PASSOU com a regra mutada.
```

```
[...] go test -race -run TestLoadCacheDir ./internal/config/
--- FAIL: TestLoadCacheDir (0.00s)
    --- FAIL: TestLoadCacheDir/derived_path_is_not_inside_the_vault (0.00s)
        config_test.go:210: CacheDir "C:\\vault\\three\\.gobsidian-cache\\7912a37a66048a78" esta dentro do cofre "C:\\vault\\three" (rel = ".gobsidian-cache\\7912a37a66048a78")
FAIL
FAIL	github.com/jonyd/gobsidian/internal/config	0.411s
[OK] internal/config/config.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

Outro teste que falsifica a mesma regra: **`internal/config/config_test.go`, `TestLoadCacheDir/derived_path_is_not_inside_the_vault`** (linhas 203–210).

### Linha 6 — `pool_test.go:9`: FAIL (0), mantido

```
      - return text.Normalize(s)
      + return s + text.Normalize("")

[...] go test -race -run TestPoolReuse ./internal/search/
--- FAIL: TestPoolReuse (0.00s)
    pool_test.go:17: Resultados incorretos: "PRESCRIÇÃO", "É", "ÁÉÍÓÚ"
FAIL
FAIL	github.com/jonyd/gobsidian/internal/search	0.657s
[OK] internal/search/analyzer.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

O sweep errou nesta: o teste falsifica `Normalize`. O que estava errado era o **comentario**, que prometia "se o pool estiver funcionando, allocs deve ser menor" e nao mede alocacao nenhuma — cobertura reportada que nao existe, que e a classe de defeito que `docs/papeis/testador.md` abre citando. Comentario reescrito para dizer o que o teste afirma e para apontar quem realmente exercita o pool sob concorrencia (`TestNormalizeNaoVazaEstadoEntreUsos`).

### Linha 7 — `write_test.go:242`: FAIL (0), mantido

```
      - if req.ExpectedHash != "" && currentHash != req.ExpectedHash {
      -     return AppendNoteResult{}, Errorf(CodeHashMismatch, ...)
      - }
      + _ = currentHash

--- FAIL: TestAppendNote_ExpectedHashMatchAndMismatch (0.02s)
    write_test.go:252: esperava CodeHashMismatch, obteve: <nil>
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.754s
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### Linha 8 — `write_test.go:368`: FAIL (0), mantido

```
      - if req.ExpectedHash != "" && currentHash != req.ExpectedHash {
      -     return PatchNoteResult{}, Errorf(CodeHashMismatch, ...)
      - }
      + _ = currentHash

--- FAIL: TestPatchNote_ExpectedHashMatchAndMismatch (0.03s)
    write_test.go:377: esperava CodeHashMismatch, obteve: <nil>
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.716s
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### Linha 9 — `write_test.go:384`: FAIL (0), mantido — **a hipotese de tautologia do brief nao se sustentou**

O brief supunha que `correctHash := fmt.Sprintf("%016x", note.Hash)` fosse tautologia por vir "do mesmo indice que o produto usa". Nao e: sao **dois sitios independentes**. O indice grava `Hash: xxhash.Sum64(data)` (`internal/index/update.go:126`) durante o `Build`; o servico calcula `hashDoConteudo(raw)` (`internal/service/write.go:22-24`) relendo o arquivo no momento da chamada. O teste cruza os dois — e esse cruzamento e exatamente o que o cliente precisa, porque o hash que o cliente recebe e o do indice: `ListItem.Hash = fmt.Sprintf("%016x", n.Hash)` em `internal/service/graph.go:481`.

Mutando so o lado do servico, o teste reprova:

```
      - return fmt.Sprintf("%016x", xxhash.Sum64(data))
      + return fmt.Sprintf("%016x", xxhash.Sum64(append(append([]byte{}, data...), 'x')))

--- FAIL: TestPatchNote_ExpectedHashMatchAndMismatch (0.01s)
    write_test.go:387: PatchNote com hash correto falhou: hash esperado "581a9ec6f8b10d60" nao confere com hash atual "8aed7e9e5ddb6a3d"
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.728s
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

Trocar o valor esperado por um `sha256` do arquivo, como o brief sugeria, **quebraria** o teste: o contrato publicado e xxhash-64 em `%016x`, e o "hash correto" tem de ser o que o cliente realmente recebe.

### Linha 10 — `slug_persistido_test.go:84`: FAIL (0), mantido

```
      - e.str(h.Slug)
      + e.str("")

        slug_persistido_test.go:89: b.md: heading "Notas sobre C#" tem Slug vazio
        slug_persistido_test.go:85: b.md: heading "Seção: Ação & Órgão" tem Slug "", recomputado da "secao acao orgao"
        slug_persistido_test.go:89: b.md: heading "Seção: Ação & Órgão" tem Slug vazio
        slug_persistido_test.go:85: c.md: heading "Titulo Simples" tem Slug "", recomputado da "titulo simples"
        slug_persistido_test.go:89: c.md: heading "Titulo Simples" tem Slug vazio
        slug_persistido_test.go:85: c.md: heading "outro em minuscula" tem Slug "", recomputado da "outro em minuscula"
        slug_persistido_test.go:89: c.md: heading "outro em minuscula" tem Slug vazio
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.962s
[OK] internal/index/persist_codec.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

A comparacao contra `parser.Slug(h.Text)` parece tautologica, mas nao e: a fonte "cache recarregado" faz o slug atravessar o codec, e e la que ele pode se perder. O controle `if vistos != 6` ja estava no teste.

### Linha 11 — `cli_subcommands_test.go:168`: ja removido

```
$ git log --oneline -S"TestSubcommands_FlagsSetPopulated" -- cmd/gobsidian/cli_subcommands_test.go
b8ed7f6 fix(cli): search honours --max-results; index and inspect stop declaring flags they ignore
ac91156 feat(cmd): index, search and inspect subcommands (RF-52)

$ grep -rn "TestSubcommands_FlagsSetPopulated" cmd/
AUSENTE no worktree
```

A linha 168 de hoje esta dentro de `TestInspectCLI...`, que afirma `data.Path == "nota2.md"` e `data.Backlinks == [nota1.md]`. Nenhum trabalho; a linha fica na tabela por exigencia do contrato.

### Linha 12 — `console_saida_test.go:100`

Tres mutacoes, porque o sitio tem dois eixos de vacuidade distintos.

(a) A metade **alcancavel** ja estava coberta:

```
      + cmd.Println("diagnostico"); cfg, err := config.Load(flags)

--- FAIL: TestServeNaoEscreveNoStdout (0.09s)
    --- FAIL: TestServeNaoEscreveNoStdout/serve (0.05s)
        console_saida_test.go:121: serve escreveu em stdout, que pertence ao JSON-RPC: "diagnostico\n"
FAIL
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

(b) A metade que o comentario prometia e o teste **nao** cobre:

```
      + _, _ = os.Stdout.WriteString("diagnostico"); os.Exit(shutdownExitCode(servePonte(parent, cfg, log)))

[...] go test -race -run TestServeNaoEscreveNoStdout ./cmd/gobsidian/
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	3.060s
[!] O teste PASSOU com a regra mutada.
```

Sobrevive por dois motivos somados: `runServe` e inalcancavel sem `--vault`, e termina em `os.Exit`, que nao volta; e a assercao observa o writer do cobra, nao o `os.Stdout` real. Isso **nao tem conserto dentro de `_test.go`** — exigiria injetar o writer em `runServe`, que e mudanca de producao. Registrado abaixo como divida.

(c) A vacuidade que **tem** conserto: um `serve` que nem estivesse na arvore de comandos deixava o teste verde.

```
      + root.AddCommand(newDoctorCmd(), newVersionCmd(), newIndexCmd(), newSearchCmd(), newInspectCmd(), newDaemonCmd())

[...] go test -race -run TestServeNaoEscreveNoStdout ./cmd/gobsidian/
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	2.585s
[!] O teste PASSOU com a regra mutada.
```

Exigir `err != nil` **nao** bastou — `unknown command "serve"` tambem e um erro nao-nulo, e a primeira versao do conserto continuou saindo 1. So exigindo que a mensagem cite `--vault` o teste passou a reprovar:

```
      + root.AddCommand(newDoctorCmd(), newVersionCmd(), newIndexCmd(), newSearchCmd(), newInspectCmd(), newDaemonCmd())

--- FAIL: TestServeNaoEscreveNoStdout (0.09s)
    --- FAIL: TestServeNaoEscreveNoStdout/serve (0.04s)
        console_saida_test.go:139: serve reprovou por outro motivo que nao a falta de --vault, entao nao chegou ao RunE: unknown command "serve" for "gobsidian"
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	2.351s
[OK] cmd/gobsidian/main.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### Linha 13 — `filter_test.go:117`

A primeira mutacao tentada (`path.go:119`, `if slashed == ".." || strings.HasPrefix(slashed, "../")` → `if false {`) sai **1** tanto com o fixture antigo quanto com o consertado, e o motivo e producao, nao teste: `validateLocal` (`path.go:85`) repete a mesma checagem, entao apagar so uma das duas nao muda o comportamento. Anotado abaixo como achado fora de escopo.

A mutacao que isola o confinamento e alargar a raiz contra a qual ele e calculado. Mesma mutacao, os dois fixtures:

**Fixture antigo (`root := "C:\\test\\vault"`, evento em `D:\...`): PASS (1)**

```
      - rel, err := filepath.Rel(root, abs)
      + rel, err := filepath.Rel(filepath.Dir(root), abs)

[...] go test -race -run TestFilter_OutsideVaultIsDropped ./internal/watcher/
ok  	github.com/jonyd/gobsidian/internal/watcher	1.923s
[!] O teste PASSOU com a regra mutada.
```

Com drives diferentes, `filepath.Rel` ja falha por si — a regra de confinamento nunca chega a rodar.

**Fixture consertado (`root`/`fora` = dois `t.TempDir()` irmaos): FAIL (0)**

```
      - rel, err := filepath.Rel(root, abs)
      + rel, err := filepath.Rel(filepath.Dir(root), abs)

--- FAIL: TestFilter_OutsideVaultIsDropped (0.00s)
    filter_test.go:141: emitted = true, want false para evento fora do vault
    filter_test.go:144: reason = "", want "outside_vault"
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	0.805s
[OK] internal/vault/path.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

(A volta ao fixture antigo foi edicao manual, ida e volta; `git diff` depois confirma que so `_test.go` mudou — ver "Commit".)

### Linha 14 — `inverted_test.go:118`, antes: PASS (1)

`DocCount() < 0` sobre um `int` que, sem base carregado, e `len(mapa)`. Mutando o corpo inteiro de `Inverted.Remove`:

```
      - 	ix.geracao++
      - 	ix.sombrearLocked(path)
      - 	ix.removeLocked(path)
      + 	_ = path

[...] go test -race -run TestInvertedConcurrencyRace ./internal/search/
ok  	github.com/jonyd/gobsidian/internal/search	3.807s
[!] O teste PASSOU com a regra mutada.
```

### Linha 14 — depois do conserto: FAIL (0)

```
--- FAIL: TestInvertedConcurrencyRace (2.00s)
    inverted_test.go:130: DocCount = 10 apos concorrencia; quer 6 ou 7
FAIL
FAIL	github.com/jonyd/gobsidian/internal/search	2.828s
[OK] internal/search/inverted.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

O intervalo `{6, 7}` nao e chute: nas dez ultimas voltas o escritor toca cada um dos dez caminhos exatamente uma vez (`i%10`) e remove os que caem em `i%3 == 0`; dez inteiros consecutivos contem tres ou quatro multiplos de tres. Medido com `-count=6` sob `-race` antes de fixar o valor, e depois em `verify.ps1`.

---

## Defeitos encontrados — fora de escopo (nenhuma linha de producao foi tocada)

1. **`internal/vault/path.go`: a regra de travessia esta escrita duas vezes.** `Canonicalize` (linha 119) e `validateLocal` (linha 85) rejeitam ambas `".."` e prefixo `"../"`. Apagar qualquer uma das duas nao muda comportamento nenhum e nenhuma suite reprova — foi o que a mutacao da linha 13 mostrou. Isso contraria "uma conta por regra" do `CLAUDE.md`. Nao consertado aqui porque o diff desta tarefa nao pode tocar producao.
2. **`internal/search/bm25.go:193`: o filtro `!math.IsNaN(score)` e inalcancavel.** Com `idf > 0` garantido pelo `1 +`, `f > 0` e `denom > 0` conferidos logo acima, e `avgdl` protegido contra zero, nao ha entrada que produza NaN. Nenhum teste consegue exercita-lo — e por isso o teste da linha 4 foi reescrito para outra regra em vez de tentar cobrir esta.
3. **`cmd/gobsidian/serve.go`: o stdout de `runServe` nao e testavel.** `runServe` termina em `os.Exit` e escreve em `os.Stdout` diretamente. A regra mais cara deste projeto ("stdout pertence ao JSON-RPC") esta coberta so no trecho anterior a ele. Fecha-la exigiria `runServe` receber o writer como parametro — a mesma correcao que `docs/papeis/testador.md` ja registra ter sido feita em `servePonteRemota`.

---

## Step 2 — `go test -race` dos pacotes tocados

```
$ go test -race ./internal/mcpsrv/ ./internal/service/ ./internal/search/ ./internal/index/ ./internal/watcher/ ./cmd/...
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	12.041s
ok  	github.com/jonyd/gobsidian/internal/service	64.816s
ok  	github.com/jonyd/gobsidian/internal/search	13.375s
ok  	github.com/jonyd/gobsidian/internal/index	4.856s
ok  	github.com/jonyd/gobsidian/internal/watcher	16.183s
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	7.796s
```

## Step 3 — `verify.ps1`

```
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. contagem de testes pulados
[!] 6 testes pulados
     --- SKIP: TestAjudanteSeguraTrava (0.00s)
     --- SKIP: TestListenRestringePermissaoUnix (0.00s)
     --- SKIP: TestSignalCancelsContext (0.10s)
     --- SKIP: TestPerfilDeHeapServindo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.01s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[...] 6. go vet (linux)
[OK] go vet (linux)
[...] 7. go vet (darwin)
[OK] go vet (darwin)
[...] 8. gofmt
[OK] gofmt
[...] 9. golangci-lint
[OK] golangci-lint
[...] 10. golangci-lint (linux)
[OK] golangci-lint (linux)
[...] 11. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 12. check_tool_params
[OK] check_tool_params
[...] 13. check_doc_refs
[OK] check_doc_refs
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Exit code 0. Os seis pulados sao os de sempre (condicoes de ambiente que so existem em outro SO ou com privilegio); nenhum deles esta entre os treze sitios auditados.

## Commit

`b3bcba0` — `test: audit thirteen sweep-flagged tests; fix the ones that could not fail, drop the redundant ones`

```
$ git show --stat --oneline b3bcba0
 .../task-161-report.md                             | 458 +++++++++++++++++++++
 cmd/gobsidian/console_saida_test.go                |  24 +-
 internal/mcpsrv/tools_read_test.go                 |  59 ++-
 internal/search/bm25_test.go                       |  24 +-
 internal/search/inverted_test.go                   |  15 +-
 internal/search/persist_test.go                    |  12 -
 internal/search/pool_test.go                       |  11 +-
 internal/service/limites_enums_test.go             |  33 +-
 internal/watcher/filter_test.go                    |  17 +-
 9 files changed, 606 insertions(+), 47 deletions(-)
```

Oito arquivos de codigo, **todos `_test.go`**; o nono e este relatorio. Nenhuma linha de producao no commit. Staging foi por caminho explicito (nunca `git add -A`), e a mensagem entrou por `git commit -F`.

---

## Fix round 1

Round de polimento pedido apos a revisao (`review-161.md`: Spec APPROVED, Quality
APPROVED, 0 bloqueantes). Tres itens nao-bloqueantes; base `44f2700`.
SHA do round: `6c1f217` — dois `_test.go` e este relatorio, staged por
caminho explicito, mensagem por `git commit -F`.

### 1. `internal/search/pool_test.go` — o nome dizia "reuse" e o teste nao mede reuso

`TestPoolReuse` virou `TestNormalizeAcentosECaixa`. O corpo nao mudou: ele afirma
`Normalize` em tres entradas acentuadas, que e exatamente o que o nome novo diz.
O comentario continua apontando para `TestNormalizeNaoVazaEstadoEntreUsos` em
`analyzer_test.go`, que e quem exercita o `sync.Pool` sob concorrencia, e agora
registra tambem o nome antigo, para quem procurar por ele.

As outras ocorrencias de `TestPoolReuse` no repositorio ficaram como estavam: sao
o relatorio da rodada anterior, o `review-161.md` e o diff da revisao, todos
registros historicos de um nome que naquela hora era o nome real.

### 2. `internal/search/bm25_test.go` — a asserção de ordem saiu

A revisao mediu que `res[0].Path != "a.md"` passava por ~3%, e a margem vinha da
disputa entre frequencia e normalizacao de comprimento, nao da regra do idf que o
teste nomeia. A linha foi **removida**, e o comentario passou a registrar a conta
que a condenava: com `ParamK1 = 1.2` e `ParamB = 0.75` (`bm25.go:19-20`), a fracao
tf/comprimento vale 1.507 para `"de de de"` contra 1.457 para `"de de"` — conta
derivada da formula, nao medida em execucao, e igual a que a revisao fez.

Quem mata a mutacao continua sendo o `Fatalf` de `len(res) != 2`. Re-rodado uma
vez, com o teste ja sem a linha de ordem:

```
[...] Mutando internal/search/bm25.go
      - idfs[i] = math.Log(1.0 + (N-d+0.5)/(d+0.5))
      + idfs[i] = math.Log((N-d+0.5)/(d+0.5))

[...] go test -race -run TestBM25TermoEmTodasAsNotasAindaPontua ./internal/search/
----------------------------------------------------------------------
--- FAIL: TestBM25TermoEmTodasAsNotasAindaPontua (0.00s)
    bm25_test.go:244: len(res) = 0, quer 2 — termo presente em todas as notas sumiu do resultado
FAIL
FAIL	github.com/jonyd/gobsidian/internal/search	0.641s
FAIL
----------------------------------------------------------------------
[OK] internal/search/bm25.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

Exit code 0 — o teste reprovou sob a mutacao, que e o resultado bom.

### 3. Relatorio — o alcance da prova da linha 1

Escrito: a saida colada da linha 1 exercita `vault_search` e mais nada, porque a
mutacao foi em `bm25.go`. A frase esta logo abaixo da tabela.

### `verify.ps1` depois do round

```
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. contagem de testes pulados
[!] 6 testes pulados
     --- SKIP: TestAjudanteSeguraTrava (0.00s)
     --- SKIP: TestListenRestringePermissaoUnix (0.00s)
     --- SKIP: TestSignalCancelsContext (0.10s)
     --- SKIP: TestPerfilDeHeapServindo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.01s)
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[...] 6. go vet (linux)
[OK] go vet (linux)
[...] 7. go vet (darwin)
[OK] go vet (darwin)
[...] 8. gofmt
[OK] gofmt
[...] 9. golangci-lint
[OK] golangci-lint
[...] 10. golangci-lint (linux)
[OK] golangci-lint (linux)
[...] 11. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 12. check_tool_params
[OK] check_tool_params
[...] 13. check_doc_refs
[OK] check_doc_refs
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

As catorze etapas verdes. A linha final so e impressa quando nenhuma etapa
reprova — nao capturei o codigo de saida desta execucao, entao nao escrevo um
numero para ele. Os seis pulados sao os mesmos da rodada anterior.

### Nao mexido, de proposito

`internal/config/config_test.go:203-208` (return silencioso) e o custo dos 501
arquivos de `limites_enums_test.go` ficaram como estao — a revisao os marcou fora
de escopo e aceito, respectivamente.
