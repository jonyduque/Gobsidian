# Task 166 — relatorio

## Progresso

- 02:03 Rider: iniciando (comentario em ponte_test.go e erro nomeado em ipc_test.go)
- feito entre 02:03 e 02:20: Passo 1 — deletados `console.go Stream.Writer`, `service.Inverted()`,
  `vault/eol.go AddBOM/NormalizeEOL`, `writer.CleanStaleTempFiles` (+ comentario em servico.go),
  `mcpsrv/server.go` campo `cfg`, `CodePathTooLong` (+ linha PATH_TOO_LONG em TOOLS.md),
  `persist_codec.go` guardas de `math.MaxUint64` (nova `uvarintLivre`), `Posting.Frequency`,
  `doctor.go` constante `halting`.
- feito entre 02:03 e 02:20: Passo 2 — `lifecycle.Options.ParentCheckInterval` MANTIDO (usado em
  producao, `lifecycle.go:57-69`, e em teste). `doctor/checks.go` contadores `cloudOnlyCount` /
  `casingCollisions` movidos atras de build tag via `platformScanState.observe` (`checks_windows.go`
  acumula, `checks_other.go` e no-op) — sem `if runtime.GOOS`.
- feito entre 02:03 e 02:20: Passo 3 — BLOQUEADO. Trocar `vault.Resolve(root, req.To)` cru por
  `if req.To == ""` quebrou `TestEscritaRecusaTravessiaComSeparadorDoWindows/windows/COM1` (ver
  Desvios). Chamada original mantida com comentario explicando o motivo; teste de regressao
  `TestMoveNoteSemDestinoEInvalidArgument` adicionado e com prova de mutacao.
- feito entre 02:03 e 02:20: Passo 4 — `docs/TOOLS.md`: removida a linha `PATH_TOO_LONG`.
- 02:20 team-lead apontou: `task-166-report.md` so tinha a linha do rider; `checks_other.go` /
  `checks_windows.go` fora do brief sem justificativa no relatorio; gopls reportando
  `platformScanState` indefinido (estado intermediario de edicao) e `uvarintLivre` nao usado.
- 02:46 corrigido: `uvarintLivre` JA tinha dois call-sites (`persist_codec.go:657,700`) — gopls
  estava com diagnostico obsoleto, confirmado com `go build ./...` limpo. `checks.go` /
  `checks_windows.go` / `checks_other.go`: estado intermediario real, nao terminado — ver Desvios
  para o antes/depois.
- 02:3x team-lead apontou (mensagem separada): verify.ps1 com tree Task 166+167 misturada,
  reprovando em `gofmt` (eol.go) e `golangci-lint (linux)` (`cloudOnlyCount`/`casingCollisions`
  "unused"). Arvore de trabalho ISOLADA de volta a so-166 antes desta correcao (167/168 revertidos
  ao HEAD, versoes em progresso guardadas fora do repo).
- 02:46 `gofmt -w internal/vault/eol.go` (uma linha em branco sobrando no fim do arquivo).
- 02:49 `cloudOnlyCount`/`casingCollisions` movidos de `vaultScan` (compartilhada, todo GOOS) para
  dentro de `platformScanState` (so existe com campos no Windows) — ver Desvios. `go build`,
  `go vet` nos tres GOOS, `gofmt -l`, e as duas passadas de `golangci-lint` (nativa e
  `GOOS=linux`) rodadas isoladamente, todas limpas — evidencia colada abaixo.
- 02:49 mutation proof de `TestMoveNoteSemDestinoEInvalidArgument` rodada (exit 0) — saida colada
  abaixo.
- 02:49 `verify.ps1` completo rodado na arvore corrigida (18 arquivos rastreados, so Task 166):
  `[OK] Bateria completa. Pode commitar.`

## Status

Task 166 concluida, com um desvio de escopo (Passo 3 BLOQUEADO) e duas correcoes fora do brief
original (contadores de `doctor` movidos para dentro de `platformScanState`, alem de so ganharem
o build tag) — ambas nomeadas e justificadas em "Desvios do brief" abaixo.

## Simbolos removidos, com busca de chamador (evidencia colada)

Todos os `grep` abaixo excluem `_test.go` e cobrem `internal` + `cmd`; saida vazia = zero
chamador de producao antes da remocao.

### `internal/console/console.go`: `Stream.Writer()`

```
$ grep -rn "Stream) Writer\|\.Writer()" --include=*.go internal cmd | grep -v _test.go
(vazio)
```
Removido: metodo + comentario.

### `internal/service/service.go`: `Inverted()`

```
$ grep -rn "\.Inverted()" --include=*.go internal cmd | grep -v _test.go
(vazio)
```
Removido: o metodo `(s *Service) Inverted()`. (`search.Inverted`, o TIPO, continua em uso extenso —
ver a busca por `func.*Inverted` colada acima nesta sessao: `NewInverted`, `MarkBuilding`, `Add`,
etc. permanecem, todos com chamador real.)

### `internal/vault/eol.go`: `AddBOM`, `NormalizeEOL`

```
$ grep -rn "AddBOM" --include=*.go internal cmd
(vazio — nenhum chamador, nem em teste)

$ grep -rn "NormalizeEOL" --include=*.go internal cmd | grep -v _test.go
internal/service/write.go:341:		normReplacement := writer.NormalizeEOL(req.Content, eol)
internal/vault/eol.go:50:// solto nunca e reescrito, porque writer.NormalizeEOL so processa o texto
internal/writer/block.go:57:	normReplacement := strings.TrimRight(NormalizeEOL(replacement, eol), eol)
internal/writer/section.go:53:// NormalizeEOL normaliza o texto fornecido para a convencao de EOL alvo (\r\n ou \n).
internal/writer/section.go:54:func NormalizeEOL(text, targetEOL string) string {
internal/writer/section.go:95:	normReplacement := NormalizeEOL(replacement, eol)
internal/writer/section.go:113:	normAppend := NormalizeEOL(contentToAppend, eol)
```
`writer.NormalizeEOL` (section.go) tem chamadores reais e NAO foi tocada — so a homonima morta em
`vault/eol.go` foi removida. Comentario de `DetectEOL` atualizado para citar a que sobrou.

### `internal/writer/atomic.go`: `CleanStaleTempFiles`

```
$ grep -rn "CleanStaleTempFiles" --include=*.go internal cmd
(vazio)
```
Removida a funcao; `SweepStaleTempFiles` (a varredura real, chamada no boot) ficou, com o
historico do 2026-07-30 (corrida de glob, violacao de compartilhamento do Windows vs
ENOENT do POSIX) preservado no comentario. `cmd/gobsidian/servico.go:49` comentario atualizado
para "Ver writer.SweepStaleTempFiles."

### `internal/mcpsrv/server.go`: campo `cfg config.Config`

```
$ grep -n "cfg" internal/mcpsrv/server.go
(campo e atribuicao removidos; `cfg` so aparece como parametro de New, usado inline)
```
`config.Config` chegava a `New` so para ser guardado sem leitor.

### `internal/service/errors.go`: `CodePathTooLong`

```
$ grep -rn "CodePathTooLong\|PATH_TOO_LONG" --include=*.go internal cmd
(vazio)
$ grep -n "PATH_TOO_LONG" docs/TOOLS.md
(vazio apos a edicao)
```
Nenhum caminho do produto produzia esse codigo — schema prometia um erro que o codigo nunca
devolvia (classe "campo de API com valor fixo mente sempre" / schema-vs-codigo do CLAUDE.md).

### `internal/index/persist_codec.go`: guardas de `math.MaxUint64`

O limite `math.MaxUint64` num `uvarint(limite, ...)` nunca pode disparar (nenhum valor uint64
excede o proprio teto do tipo) — guarda morta por construcao, nao por ausencia de chamador. Os
dois call-sites que usavam esse limite (`valor uint64` generico e `note hash`) agora chamam
`uvarintLivre`, que le sem teto e documenta por que:

```go
// uvarintLivre le um uvarint sem teto. Existe porque dois chamadores
// (valor uint64 generico e note hash) nao tem teto natural menor que o
// proprio tipo — chamar uvarint com limite=math.MaxUint64 e uma guarda que
// nunca pode disparar, e guarda que nunca dispara e pior que ausencia dela:
// parece protecao e nao protege nada.
func (l *leitor) uvarintLivre(oque string) uint64 {
    if l.err != nil {
        return 0
    }
    v, n := binary.Uvarint(l.b[l.i:])
    if n <= 0 {
        l.falha("%w: lendo %s: varint invalido em %d", ErrIndexCacheCorrupted, oque, l.i)
        return 0
    }
    l.i += n
    return v
}
```

### `internal/search/inverted.go`: `Posting.Frequency`

```
$ grep -rn "Frequency" --include=*.go internal cmd
(vazio apos a edicao — os dois usos remanescentes eram as duas atribuicoes ja removidas)
```
Confirmado tambem que `Frequency` nao faz parte do formato binario de cache (`persist.go` /
`persist_codec.go` nunca serializavam esse campo) — remocao nao muda o formato 6.
`inverted_test.go` e `soa_test.go` ajustados para nao referenciar o campo.

### `internal/doctor/doctor.go`: constante `halting`

```
$ grep -n "halting" internal/doctor/*.go
internal/doctor/checks.go:50:		// esta e halting: ela aborta as seguintes, e uma checagem posterior
internal/doctor/doctor.go:13:// quando marcado como halting.
```
So restam comentarios em prosa explicando o CONCEITO (checkRootExists interrompe as demais); a
constante/campo booleano que sempre valia `true` foi removida e `Run` agora usa a lista fixa
`[]check{checkRootExists, checkReadable}`, parando no primeiro `StatusFail` — mesmo
comportamento, sem campo que nunca variava.

## Passo 2 — `doctor/checks.go` contadores atras de build tag

`cloudOnlyCount` e `casingCollisions` eram escritos por todo cofre, em toda plataforma, mas so
lidos pelos checks de `checks_windows.go`. Corrigido com o hook `platformScanState.observe`
(sem `if runtime.GOOS`, como a regra do CLAUDE.md exige) — mas a primeira versao deixou os DOIS
campos soltos em `vaultScan` (struct compartilhada, compilada em todo GOOS), e isso e
exatamente o que fez `golangci-lint (linux)` reprovar (ver Desvios): fora do Windows nenhum
codigo compilado le nem escreve os dois campos, entao o linter os marca `unused` nesse alvo.

Correcao: os dois campos agora moram DENTRO de `platformScanState` (Windows: `cloudOnlyCount`,
`casingCollisions`, `seen`; outros SOs: `struct{}` vazia), e `vaultScan` so guarda
`platform platformScanState`. `observe` deixou de receber `*vaultScan` — escreve em si mesmo
(`p.cloudOnlyCount++`), chamado como `scan.platform.observe(e)`.

## Passo 3 — BLOQUEADO (`write.go` MoveNote)

O brief propunha trocar a chamada `vault.Resolve(s.vault.Root(), req.To)` — cujo unico proposito
aparente e verificar `req.To != ""` — por um `if req.To == ""` explicito.

Medido: fazendo essa troca, `TestEscritaRecusaTravessiaComSeparadorDoWindows/windows/COM1`
reprovou com `MoveNote(to="COM1") devolveu sucesso`. A chamada `vault.Resolve` sobre `req.To` CRU
(antes do `.md` que `toInput` acrescenta) e o unico ponto que pega nome de dispositivo reservado
do Windows SEM extensao — porque desde o Windows 11, `filepath.IsLocal` (via
`RtlIsDosDeviceName_U`) so recusa `"COM1"`, nao `"COM1.md"` (confirmado com programa de teste
isolado nesta sessao: `COM1: false` (recusado), `COM1.md: true` (aceito)). Trocar a chamada por
um `if` vazio removeria essa protecao.

Mantido o codigo original, com comentario explicando o motivo. Adicionado teste de regressao
`TestMoveNoteSemDestinoEInvalidArgument` (`move_test.go`), cobrindo `to=""` e `to="   "`, com
prova de mutacao (abaixo).

## Evidencia — build, vet, gofmt, lint

```
$ go build ./...
(sem saida — sucesso)

$ GOOS=windows go vet ./...
(sem saida — sucesso)

$ GOOS=linux go vet ./...
(sem saida — sucesso)

$ GOOS=darwin go vet ./...
(sem saida — sucesso)

$ gofmt -l internal/doctor internal/vault
(sem saida — nada a formatar)

$ golangci-lint run ./internal/... ./cmd/... ./tools/...
0 issues.

$ GOOS=linux golangci-lint run ./internal/... ./cmd/... ./tools/...
0 issues.
```

## Evidencia — prova de mutacao (`TestMoveNoteSemDestinoEInvalidArgument`)

```
$ pwsh -File scripts/mutate.ps1 -Path internal/service/write.go `
    -Anchor '_, _, err = vault.Resolve(s.vault.Root(), req.To)' `
    -Replacement '_, _, err = vault.Resolve(s.vault.Root(), "placeholder-nao-vazio.md")' `
    -Test TestMoveNoteSemDestinoEInvalidArgument -Package ./internal/service/

[...] Mutando internal/service/write.go
      - _, _, err = vault.Resolve(s.vault.Root(), req.To)
      + _, _, err = vault.Resolve(s.vault.Root(), "placeholder-nao-vazio.md")

[...] go test -race -run TestMoveNoteSemDestinoEInvalidArgument ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestMoveNoteSemDestinoEInvalidArgument (0.01s)
    move_test.go:205: MoveNote(to="") error = nil, esperado INVALID_ARGUMENT
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.047s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT CODE: 0
```

## `git diff --stat` (arquivos desta task)

```
 cmd/gobsidian/servico.go          |  2 +-
 docs/TOOLS.md                     |  1 -
 internal/console/console.go       |  4 ----
 internal/doctor/checks.go         | 37 ++++++++++++++++----------------
 internal/doctor/checks_other.go   |  8 +++++++
 internal/doctor/checks_windows.go | 40 ++++++++++++++++++++++++++++++----
 internal/doctor/doctor.go         | 26 ++++++++--------------
 internal/index/persist_codec.go   | 22 +++++++++++++++++--
 internal/mcpsrv/server.go         |  2 --
 internal/search/inverted.go       |  3 ---
 internal/search/inverted_test.go  |  4 ++--
 internal/search/soa_test.go       |  4 ----
 internal/service/errors.go        |  1 -
 internal/service/move_test.go     | 31 +++++++++++++++++++++++++++
 internal/service/service.go       |  5 -----
 internal/service/write.go         | 10 +++++++++
 internal/vault/eol.go             | 21 ++----------------
 internal/writer/atomic.go         | 45 +++++++++++----------------------------
 18 files changed, 150 insertions(+), 116 deletions(-)
```

## `verify.ps1` — ultima linha

```
[OK] Bateria completa. Pode commitar.
```
14 etapas: `go build`, `go test -race`, contagem de pulados (6, todos skips legitimos ja
conhecidos — `TestAjudanteSeguraTrava`, `TestListenRestringePermissaoUnix`,
`TestSignalCancelsContext`, `TestPerfilDeHeapServindo`, `TestNew_FailsOnUnwatchablePath`,
`TestWriteAtomicPreservaOModoDoAlvo`), `go test` de tetos de latencia, `go vet` (windows/linux/
darwin), `gofmt`, `golangci-lint` (nativo e linux), `check_net`, `check_tool_params`,
`check_doc_refs`, `check_readme_anchors` — todas `[OK]`.

Rodado sobre a arvore isolada de Task 166 (os 18 arquivos rastreados listados acima; nenhum
arquivo de Task 167/168 presente na arvore de trabalho no momento desta execucao).

## Desvios do brief

1. **Passo 3 BLOQUEADO.** Ver secao dedicada acima. O brief propunha remover a chamada
   `vault.Resolve` sobre `req.To` cru; medido que isso quebra a rejeicao de `COM1` (nome de
   dispositivo reservado do Windows sem extensao). Chamada original mantida, com comentario e
   teste de regressao novo.

2. **`internal/doctor/checks_windows.go` e `checks_other.go` tocados — fora da lista literal do
   brief, mas dentro do Passo 2** ("mover os dois contadores atras de build tag, nunca com
   `if runtime.GOOS`"). O brief nomeava so `checks.go:277-278`; mover os campos para tras de um
   build tag exige, por definicao, um arquivo por GOOS — daí os dois arquivos novos/editados.
   A primeira tentativa (contadores soltos em `vaultScan`, so o `observe` atras do tag) ainda
   deixava os dois campos "escritos em todo GOOS, lidos so no Windows" dentro de uma struct
   compartilhada — e isso e o que `golangci-lint (linux)` pegou como `unused`. Correcao final:
   os proprios campos moraram para dentro de `platformScanState`, que so o Windows preenche.

3. **`atomic_test.go`** mantido sem alteracao (nao listado no brief como candidato a mudanca);
   `TestWriteAtomicPreservaOModoDoAlvo` continua pulado por motivo preexistente, nao-relacionado
   a esta task.
