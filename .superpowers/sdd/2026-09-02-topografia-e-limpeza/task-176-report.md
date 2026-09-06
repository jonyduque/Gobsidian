# Task 176 -- relatorio

## Progresso

- 10:46 -- Base commit confirmado: `6b70721` (`docs(sdd): record the Task 175 review and its package`), HEAD ja estava la. Lido o brief `task-176-brief.md` e os arquivos-alvo (`index.go`, `inspect.go`, `search.go`, `serve.go`, `daemon.go`, `doctor.go`, `cli_log.go`, `internal/boot/indice.go`, `internal/config/config.go`, `cli_subcommands_test.go`).
- 10:46 -- Step 1: criado `cmd/gobsidian/index_origem_test.go` com `TestIndexCmdJSONTrazOrigem` (texto literal do brief). Rodado -- FAIL confirmado (ver evidencia abaixo).
- 10:47-10:48 -- Step 2: `index.go` e `inspect.go` passaram a abrir o indice via `boot.AbrirIndice` em vez de `index.New()+Build`; `indexSummaryJSON` ganhou `Origin string \`json:"origin"\``; console de `index` ganhou `con.Item("Origem: %s", origem)`. Para o teste rodar sem escrever no cache real do usuario, os dois arquivos ganharam `--cache-dir`/`--log-level` registrados a mao (mesmo texto de `search.go`), a serem substituidos no commit 2.
- 10:48 -- Step 4: `--cache-dir", t.TempDir()` acrescentado aos quatro `SetArgs` de `TestIndexCmd_StdoutAndJSON` e `TestInspectCmd_StdoutAndJSON` em `cli_subcommands_test.go`.
- 10:48 -- Step 1 re-rodado -- PASS confirmado.
- 10:48 -- `go build ./...`, `go vet ./...`, `go test -race ./cmd/gobsidian/...` -- todos limpos.
- 10:48 -- Step 5: prova de mutacao rodada contra `Origin:     origem,` (texto real pos-gofmt, ancora do brief `Origin: origem,` sem o alinhamento nao batia) -- exit 0, teste reprovou com a mutacao, arquivo restaurado byte a byte (confirmado por `grep` apos a rodada).
- 10:49 -- Step 6: `README.md` (tabela de flags, secao "Command line") e `docs/OPERACAO.md` (nova secao) documentam `--cache-dir`/`--log-level` em `index`/`inspect` e o campo `origin`. UTF-8 validado nos dois com `python -c "open(...,encoding='utf-8').read()"`.
- 10:49 -- `pwsh -File scripts/verify.ps1` (14 etapas, completo) -- verde. Commit 1 criado: `022d2ef` (`feat(cli): index and inspect load from the cache`), so os 6 arquivos do escopo do commit.
- 10:52-10:56 -- Step 3: criado `cmd/gobsidian/flags.go` com `flagsDeCofre`/`flagsDeCache`; `search.go`, `index.go`, `inspect.go`, `serve.go`, `daemon.go` passaram a chamar as duas; `doctor.go` so `flagsDeCofre` (nao le cache). `go build`, `go vet` limpos; `grep -rn '"vault", ""' cmd/gobsidian/*.go` -- so `flags.go`. Testes-alvo e suite completa de `cmd/gobsidian` com `-race` -- PASS.
- 10:57 -- Commit 2 tentado e bloqueado pelo hook `pre_commit_docs.ps1` (7 .go de producao em stage, nenhum doc -- correto, e um commit estrutural puro). Ajustada a mensagem para `refactor(cli): register shared vault flags once [sem-doc]` e o proprio comando `git commit` para carregar `[sem-doc]` no texto que o hook le (o hook inspeciona o texto do comando bash, nao o arquivo de `-F`). Commit 2 criado: `8a11cec`.
- 10:57-10:59 -- `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` (segunda rodada, pos-commit 2) -- verde.
- 11:01 -- `git status --porcelain` conferido: so os arquivos preexistentes do dono (test-vault/, .claude/skills/troglodita*, artefatos de outras tasks em `.superpowers/sdd/`) e os dois `commit-176-*.txt` que eu proprio criei permanecem fora do historico -- nada meu vazou para dentro dos dois commits.

## Status

DONE

## Commits

- `022d2ef` -- `feat(cli): index and inspect load from the cache`
- `8a11cec` -- `refactor(cli): register shared vault flags once [sem-doc]`

`git log --oneline -2`:
```
8a11cec refactor(cli): register shared vault flags once [sem-doc]
022d2ef feat(cli): index and inspect load from the cache
```

## Step 1 -- FAIL -> PASS

RED (`go test ./cmd/gobsidian/ -run TestIndexCmdJSONTrazOrigem -v`), antes de qualquer mudanca de producao:
```
=== RUN   TestIndexCmdJSONTrazOrigem
    index_origem_test.go:24: index: unknown flag: --cache-dir
        stderr:
        Error: unknown flag: --cache-dir
--- FAIL: TestIndexCmdJSONTrazOrigem (0.04s)
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	0.186s
FAIL
```
Nota: a falha imediata foi "unknown flag: --cache-dir" (a flag ainda nao existia em `index.go`), nao "origin vazio" como o brief antecipava -- porque `index`/`inspect` nem aceitavam `--cache-dir` antes desta task. E a mesma causa raiz (nada de Task 176 implementado ainda) e o teste falha exatamente onde deveria: sem o Step 2, `index` nao tem como devolver `origin`.

GREEN, apos Step 2 e Step 4:
```
=== RUN   TestIndexCmdJSONTrazOrigem
--- PASS: TestIndexCmdJSONTrazOrigem (0.10s)
PASS
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	0.242s
```

## Step 3 -- grep

```
$ grep -rn '"vault", ""' cmd/gobsidian/*.go
cmd/gobsidian/flags.go:13:	cmd.Flags().StringVar(&f.VaultPath, "vault", "", "caminho da raiz do cofre (obrigatorio)")
```
So `flags.go` registra `--vault` a mao; os seis subcomandos (`search`, `index`, `inspect`, `serve`, `daemon`, `doctor`) chamam `flagsDeCofre`.

## Step 5 -- prova de mutacao

Ancora ajustada ao texto real (gofmt alinhou o campo `Origin` com espacos extras: `Origin:     origem,`, nao `Origin: origem,` como o brief escreveu):

```
$ pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/index.go -Anchor 'Origin:     origem,' -Replacement 'Origin:     "build",' -Test TestIndexCmdJSONTrazOrigem -Package ./cmd/gobsidian/
Carregado em 448ms
[...] Mutando cmd/gobsidian/index.go
      - Origin:     origem,
      + Origin:     "build",

[...] go test -race -run TestIndexCmdJSONTrazOrigem ./cmd/gobsidian/
----------------------------------------------------------------------
--- FAIL: TestIndexCmdJSONTrazOrigem (0.11s)
    index_origem_test.go:42: segunda execucao: origin = "build", quero "cache"
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	1.600s
FAIL
----------------------------------------------------------------------
[OK] cmd/gobsidian/index.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada -- a regra esta verificada.
EXIT=0
```
Restauracao confirmada por `grep -n "Origin:" cmd/gobsidian/index.go` apos a rodada: `63:					Origin:     origem,` (identico ao original).

## verify.ps1 -- ultimas linhas

Primeira rodada (completa, 14 etapas, antes do commit 1):
```
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```
6 testes pulados reportados na etapa 3 (skips de plataforma ja conhecidos: `TestAjudanteSeguraTrava`, `TestListenRestringePermissaoUnix`, `TestSignalCancelsContext`, `TestPerfilDeHeapServindo`, `TestWriteAtomicPreservaOModoDoAlvo`, `TestNew_FailsOnUnwatchablePath` -- nenhum deles em `cmd/gobsidian`, nao relacionados a esta task).

Segunda rodada (`-SkipCross -SkipNet`, apos commit 2, pura estrutura):
```
[...] 11. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```
Mesmos 6 skips reportados.

## Textos de ajuda que divergiam

Nenhum. `--vault`, `--follow-symlinks`, `--cache-dir` e `--log-level` ja tinham o mesmo texto em todos os arquivos que os registravam a mao (`search.go`, `serve.go`, `daemon.go`, `doctor.go` para `--vault`/`--follow-symlinks`; `search.go` para `--cache-dir`/`--log-level`) -- conferido por leitura direta de cada arquivo antes de escrever `flags.go`. (`doctor.go` tem texto diferente para `--read-only`, mas essa flag nao faz parte das quatro que migraram para `flagsDeCofre`/`flagsDeCache`.)

## Escotilha [sem-doc] no commit 2

O hook `pre_commit_docs.ps1` bloqueou o commit 2 (7 `.go` de producao em stage, nenhum doc) -- comportamento correto, porque o commit 2 e puramente estrutural (o comportamento e a doc ja foram no commit 1). O hook le o TEXTO DO COMANDO bash, nao o arquivo passado a `-F`, entao a mensagem de commit sozinha (com `[sem-doc]` no assunto, em `commit-176-2.txt`) nao bastava -- foi preciso que o proprio comando `git commit -F ... # [sem-doc]` contivesse a string. O commit final tem `[sem-doc]` no assunto, visivel no `git log`.

## Concerns

- Nenhum. Escopo entregue integralmente: os dois commits, na ordem, `verify.ps1` verde duas vezes, prova de mutacao com saida colada, grep do Step 3 colado, documentacao (README + OPERACAO) atualizada e validada como UTF-8, nada do trabalho nao commitado do dono tocado.
- `TestSubcommands_FlagsSetPopulated` citado no brief nao existe com esse nome -- e `TestIndexEInspectNaoAceitamFlagsQueIgnoram` (linha 205), como a correcao do orquestrador ja apontava; rodado e continua verde.

## `scripts/audit_reports.ps1 176`

Rodado; `=== Relatorios (1) ===` volta vazio -- nenhum achado dentro deste
`task-176-report.md`. Os 14 achados que o script lista sao todos em
`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` (o ledger do plano
M0-M6, arquivo distinto do ledger `2026-09-02-topografia-e-limpeza` desta
task), e datam de tasks 4, 6, 94-103 e 153 -- muito antes da Task 176 e do
milestone atual. Nao toquei nesse ledger e nao investiguei o merito de cada
achado (SHA-fantasma `deadbee`, relatorios ausentes de 94-103, descricoes que
nao batem com o range de commit citado): sao divida preexistente de um
milestone ja fechado, fora do escopo desta task, e ficam registrados aqui
para quem decidir se vale abrir uma task de limpeza do ledger antigo.
