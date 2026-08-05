# Relatório Task 79: checador de artefato citado na doc que não existe no código

**Data:** 2026-08-05
**Base:** `d29d7dd`
**Commit:** `cfb3d60c440115563e3603bc8d25b856610aa19a`

---

## O que foi entregue

`scripts/check_doc_refs.ps1`. Varre `docs/*.md` (não recursivo — `docs/superpowers/plans/`
fica de fora) e `README.md`, procurando token entre crases que casa com uma de
três formas de identificador de código:

- **ARQUIVO** — termina em `.go` ou `.gob`.
- **SNAKE_CASE** — `[a-z][a-z0-9]*(_[a-z0-9]+)+`, minúsculo puro.
- **CAMEL_CALL** — `[A-Z][A-Za-z0-9]*(...)`, identificador com maiúscula
  inicial seguido de parênteses.

Duas checagens, conforme a classe:

- `.go` é arquivo-fonte: existe no repositório, ou não existe — confere
  presença em disco pelo nome-base (`Get-ChildItem -Recurse -Filter *.go`,
  comparado por `HashSet` de nomes).
- `.gob`, `SNAKE_CASE` e `CAMEL_CALL` não são arquivo-fonte; o teste é se o
  token aparece como substring em algum `.go` do repositório (produção e
  teste juntos), `-cnotmatch` (case-sensitive).

Blocos cercados por ` ``` ` ficam fora da varredura (contador de estado por
linha, sem `DOTALL`).

## Por que ARQUIVO usa duas mecânicas

A primeira versão testava TUDO por substring-no-corpus, igual às outras duas
classes. Isso deu 21 achados de `ARQUIVO`, quase todos falso-positivo: Go não
se autorreferencia por nome de arquivo, então `persist.go`, `analyzer.go`,
`bm25.go` etc. — arquivos reais, verificados com `find` — nunca aparecem como
texto literal em nenhum `.go`. Substring-no-corpus é o teste certo para
`.gob` (arquivo de dado gerado em runtime, fora do repositório, cuja única
pista é o código que o cria citando o nome), errado para `.go` (arquivo
versionado, cuja existência se confere no disco). Comentado na função de
classificação do script.

## Bug de parsing encontrado e corrigido durante o desenvolvimento

Escrita inicial: `Write-Output ("... [{2}] `{3}`" -f ...)` — o backtick
imediatamente antes do `"` de fechamento formava a sequência de escape
`` `" `` (aspas dupla literal), então a string nunca fechava ali; o parser
consumia o resto do arquivo tentando achar o próximo `"`, e o erro reportado
apontava para uma linha 20+ adiante da causa real. Corrigido usando escape de
crase dupla (` `` ` = uma crase literal): `` "... ``{3}``" ``. Bisseção com
`[System.Management.Automation.Language.Parser]::ParseFile` (`head -n K`
progressivo) localizou a linha exata (97) antes do fix.

Também corrigido: `-match`/`-notmatch` do PowerShell são **insensíveis a
maiúsculas por padrão**. Sem `(?-i)` nos três padrões e sem `-cnotmatch` na
busca no corpus, `SNAKE_CASE` casava com `ERROR_SHARING_VIOLATION` (API do
Windows) e `FILE_ATTRIBUTE_RECALL_ON_OPEN`, e `CAMEL_CALL` casaria com
`close()` (inicial minúscula). Achado ao contar o volume da primeira rodada
(40 achados) e investigar cada classe antes de aceitar.

## Volume de achados no repositório hoje: 14

```
[i] corpus: 201 arquivos .go.
  docs\ARCHITECTURE.md:358: [ARQUIVO] `backend_inotify.go`
  docs\ARCHITECTURE.md:358: [ARQUIVO] `backend_windows.go`
  docs\ARCHITECTURE.md:474: [SNAKE_CASE] `format_version`
  docs\ARCHITECTURE.md:475: [SNAKE_CASE] `parser_version`
  docs\ARCHITECTURE.md:478: [SNAKE_CASE] `parser_version`
  docs\ESTRUTURA.md:186: [ARQUIVO] `interfaces.go`
  docs\ESTRUTURA.md:186: [ARQUIVO] `helpers.go`
  docs\ESTRUTURA.md:188: [ARQUIVO] `helpers.go`
  docs\ESTRUTURA.md:188: [ARQUIVO] `utils.go`
  docs\ESTRUTURA.md:234: [ARQUIVO] `snake_case.go`
  docs\PRD.md:378: [ARQUIVO] `index_cache.gob`
  docs\PRD.md:382: [ARQUIVO] `index_cache.gob`
  docs\TOOLS.md:89: [SNAKE_CASE] `total_bytes`
  docs\WINDOWS.md:157: [SNAKE_CASE] `max_user_watches`

[!] 14 achado(s). Nao e veredito -- cada linha e um token que
    precisa de uma pessoa confirmando se e artefato real, flag/nome
    externo (ruido esperado), ou decisao de doc que o codigo ainda
    nao cumpriu.
EXIT=1
```

Abaixo do teto de ~20 do brief; não foi preciso restringir o padrão além do
ajuste de ARQUIVO acima.

### Classificação, um por linha

- **`backend_inotify.go`, `backend_windows.go`** (ARCHITECTURE.md:358) —
  ruído aceito. São arquivos internos da dependência `fsnotify` v1.10.1
  (vendorizada fora deste repositório), citados em prosa que descreve o
  comportamento da lib, não artefato do gobsidian.
- **`format_version`, `parser_version`** (ARCHITECTURE.md:474/475/478) —
  ruído aceito. Tabela descrevendo os campos `FormatVersion`/`ParserVersion`
  de `internal/search/persist_codec.go` (confirmado: `h.FormatVersion`,
  `h.ParserVersion` existem e são usados nas linhas 122-123, 343-344,
  357-358) em estilo minúsculo por legibilidade da tabela, não os nomes Go
  reais.
- **`interfaces.go`, `helpers.go` (x2), `utils.go`, `snake_case.go`**
  (ESTRUTURA.md:186/188/234) — ruído aceito. Nomes de exemplo numa seção de
  convenção de nomenclatura — `helpers.go`/`utils.go` são exatamente os
  nomes que o `CLAUDE.md` deste projeto proíbe ("Sem `helpers.go`,
  `utils.go`, `common.go`"), citados como exemplo do que não fazer, não como
  arquivo que deveria existir.
- **`index_cache.gob` (x2)** (PRD.md:378/382) — **achado real, o mesmo que a
  evidência do brief nomeia.** `docs/PRD.md` Q3 fechou persistir dois caches;
  só `inverted_cache.gob` existe (`internal/search/persist.go:100,118`, mais
  `persist_test.go`). `index_cache.gob` não aparece em nenhum `.go`.
- **`max_user_watches`** (WINDOWS.md:157) — ruído aceito. Nome do sysctl do
  kernel Linux, citado ao comparar o comportamento do Windows (que não tem
  limite de watches) contra o Linux. Não é artefato deste repositório.
- **`total_bytes`** (TOOLS.md:89) — **achado real, não estava na evidência do
  brief; apareceu na varredura.** TOOLS.md promete que o retorno de
  `note_read` traz `content, path, hash, truncated, total_bytes`, mas
  `service.ReadResult` (`internal/service/read.go:28-33`) só tem `Content`,
  `Hash`, `Section`, `Truncated` — nem `path` nem `total_bytes` existem no
  struct que o SDK serializa. `grep -rn "total_bytes\|TotalBytes"
  --include=*.go .` não acha nada.

## Confirmação: `index_cache.gob` aparece

Sim — as duas ocorrências (PRD.md:378 e 382) estão na lista acima, classe
`ARQUIVO`.

## Prova de disparo (verificação obrigatória do brief)

**Passo 1-2 — inserido `` `create_dirs` `` em `docs/TOOLS.md` linha 9, rodado:**

```
  docs\ARCHITECTURE.md:358: [ARQUIVO] `backend_inotify.go`
  docs\ARCHITECTURE.md:358: [ARQUIVO] `backend_windows.go`
  docs\ARCHITECTURE.md:474: [SNAKE_CASE] `format_version`
  docs\ARCHITECTURE.md:475: [SNAKE_CASE] `parser_version`
  docs\ARCHITECTURE.md:478: [SNAKE_CASE] `parser_version`
  docs\ESTRUTURA.md:186: [ARQUIVO] `interfaces.go`
  docs\ESTRUTURA.md:186: [ARQUIVO] `helpers.go`
  docs\ESTRUTURA.md:188: [ARQUIVO] `helpers.go`
  docs\ESTRUTURA.md:188: [ARQUIVO] `utils.go`
  docs\ESTRUTURA.md:234: [ARQUIVO] `snake_case.go`
  docs\PRD.md:378: [ARQUIVO] `index_cache.gob`
  docs\PRD.md:382: [ARQUIVO] `index_cache.gob`
  docs\TOOLS.md:9: [SNAKE_CASE] `create_dirs`
  docs\TOOLS.md:89: [SNAKE_CASE] `total_bytes`
  docs\WINDOWS.md:157: [SNAKE_CASE] `max_user_watches`

[!] 15 achado(s). ...
EXIT=1
```

`docs\TOOLS.md:9: [SNAKE_CASE] `create_dirs`` apareceu — 14 → 15 achados.

**Passo 3-4 — removida a linha (Edit, sem script), rodado de novo:**

```
[i] corpus: 201 arquivos .go.
  docs\ARCHITECTURE.md:358: [ARQUIVO] `backend_inotify.go`
  docs\ARCHITECTURE.md:358: [ARQUIVO] `backend_windows.go`
  docs\ARCHITECTURE.md:474: [SNAKE_CASE] `format_version`
  docs\ARCHITECTURE.md:475: [SNAKE_CASE] `parser_version`
  docs\ARCHITECTURE.md:478: [SNAKE_CASE] `parser_version`
  docs\ESTRUTURA.md:186: [ARQUIVO] `interfaces.go`
  docs\ESTRUTURA.md:186: [ARQUIVO] `helpers.go`
  docs\ESTRUTURA.md:188: [ARQUIVO] `helpers.go`
  docs\ESTRUTURA.md:188: [ARQUIVO] `utils.go`
  docs\ESTRUTURA.md:234: [ARQUIVO] `snake_case.go`
  docs\PRD.md:378: [ARQUIVO] `index_cache.gob`
  docs\PRD.md:382: [ARQUIVO] `index_cache.gob`
  docs\TOOLS.md:89: [SNAKE_CASE] `total_bytes`
  docs\WINDOWS.md:157: [SNAKE_CASE] `max_user_watches`

[!] 14 achado(s). ...
EXIT=1
```

`create_dirs` sumiu — voltou a 14. `git status --porcelain docs/TOOLS.md`
saiu vazio (arquivo idêntico ao original).

## Sem prova de mutação

Esta tarefa **não tem prova de mutação** — o entregável é PowerShell, e
`mutate.ps1` roda teste Go. A prova equivalente é o disparo controlado acima:
inserir o token conhecido, confirmar que aparece, remover, confirmar que
some.

## Bateria

```
$ pwsh -File scripts/verify.ps1 -SkipCross -SkipNet
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 4. go vet (windows)
[OK] go vet (windows)
[i] vet cruzado pulado (-SkipCross)
[...] 5. gofmt
[OK] gofmt
[...] 6. golangci-lint
[OK] golangci-lint
[i] check_net pulado (-SkipNet)
[...] 7. check_tool_params
[OK] check_tool_params

[OK] Bateria completa. Pode commitar.
EXIT=0
```

## SHA do commit

```
$ git cat-file -t cfb3d60c440115563e3603bc8d25b856610aa19a
commit
```

## O que ficou de fora

Nada do escopo pedido. Um achado real (`total_bytes`) surgiu da varredura sem
estar na evidência do brief — fica registrado acima, não corrigido: corrigir
`ReadResult`/TOOLS.md é fora do escopo desta tarefa (que é o checador, não a
correção dos gaps que ele acha).
