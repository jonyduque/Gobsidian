# Task 160 — Apagar ou consertar os cinco testes que nao podiam falhar

Status: DONE
SHA: `ef2bbb4a89b3b8e339260476ccab7a7483dfecd7`

## Progresso

- 02:28 li os dois briefs, `CLAUDE.md`, `papeis/implementador.md`, `papeis/testador.md`
- 02:34 li os cinco sitios e confirmei os cinco mecanismos; relatorios criados
- 02:35 Step 1 escrito (`TestReconcileCtxCanceladoParaAntesDeProcessarTudo`), verde isolado
- 02:36 Step 1 provado por mutacao em `internal/vault/walk.go` — `mutate.ps1` exit 0
- 02:36 Steps 2, 3, 5 apagados (`build_test.go`, `git rm normalizacao_equivalente_test.go`, `read_test.go`)
- 02:36 Step 4 aplicado nos tres sitios de `schema_params_test.go`
- 02:36 Step 4 provado: `{}` derruba a guarda de tamanho, texto invalido derruba a de Unmarshal; restaurado
- 02:37 verificacoes do brief (`no tests to run` nos dois pacotes)
- 02:42 `verify.ps1` verde, 13/13 etapas
- 02:44 relatorio final e commit `ef2bbb4` — Task 160 encerrada

## Os cinco mecanismos, confirmados antes de editar

1. **`overflow_test.go:219` `TestReconcile_CtxCancelStopsEarly` — mecanismo
   confirmado: sim.** Evidencia: entre `idx.Build` (linha 234) e `Reconcile`
   (linha 246) o teste nao toca em arquivo nenhum, e o atalho de
   `overflow.go:58-61` (`if n, ok := idx.Get(e.Path); ok && (inv == nil ||
   inv.HasDoc(...))` seguido de `if n.ModTime.Equal(info.ModTime()) && n.Size ==
   info.Size() { return nil }`) devolve `nil` sem incrementar `updated` para as
   200 notas. Logo `updated` vale 0 com ou sem cancelamento e a assercao
   `updated >= 200` era inalcancavel. As outras duas assercoes
   (`idx.NoteCount() != 200`, `removed != 0`) tambem valiam nos dois mundos.

2. **`build_test.go:143` `TestBuildSkipsUnreadableFile` — mecanismo confirmado:
   sim.** Evidencia: o teste cria um DIRETORIO `unreadable.md`
   (`os.Mkdir(unreadablePath, 0o755)`, linha 156) esperando erro de leitura, mas
   `internal/vault/walk.go:172-180` trata `d.IsDir()` e devolve antes de
   `Canonicalize`/`Classify` (linhas 182-188): um diretorio nunca chega a ser
   classificado como nota e nunca e lido. `NoteCount() == 2` seria verdade
   mesmo se o `Mkdir` nao existisse. A cobertura real de "arquivo ilegivel e
   pulado" esta em `internal/index/build_descarte_test.go`, com
   `vaulttest.TravarExclusivo`, que confere que a trava trava.

3. **`normalizacao_equivalente_test.go` — mecanismo confirmado: sim.**
   Evidencia: `internal/index/query.go:39-41` e literalmente
   `func normalizeString(s string) string { return text.Normalize(s) }`. O teste
   compara `normalizeString(e)` com `text.Normalize(e)` — a mesma chamada dos
   dois lados. Tautologia: nenhuma entrada pode faze-lo falhar sem que o
   compilador reclame antes. `normalizeString` continua com chamadores de
   producao (`query.go:89`, `:92`, `:96`), entao apagar o teste nao deixa a
   funcao orfa.

4. **`schema_params_test.go:169,:202,:235` — mecanismo confirmado: sim.**
   Evidencia: os tres sitios eram `_ = json.Unmarshal(b, &data)` seguidos de
   `for _, e := range data.Edges { ... }`. Erro de Unmarshal descartado e
   `data.Edges` vazio produzem laco de zero iteracoes, e o teste passa sem
   comparar nada. (O brief citava 171/204/237 e o nome `got`; no arquivo de hoje
   sao 169/202/235 e a struct chama-se `data`. O campo iterado e `Edges` nos
   TRES — confirmado por leitura: `:162-167`, `:196-200`, `:229-233`.)

5. **`read_test.go:115` `TestReadNoteCloudOnlyFails` — mecanismo confirmado:
   sim.** Evidencia: o corpo inteiro era
   `t.Skip("requer arquivo com FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS")` — skip
   incondicional, cobertura zero reportada como teste. A cobertura pretendida
   existe em `internal/service/cloudonly_replace_windows_test.go`, que monta o
   placeholder com `vaulttest.MarcarSomenteNuvem`.

## Desvio do desenho do brief no Step 1, e por que

O brief mandava afirmar `updated < n` depois de modificar as 300 notas. **Essa
assercao nao mata a mutacao exigida.** O motivo esta medido, nao raciocinado: o
ctx e checado em mais de um lugar no caminho. Desligado o check da varredura
(`internal/vault/walk.go:166`), `Reconcile` chega a `idx.Replace`, que chama
`v.ReadAll` — e `internal/vault/vault.go:233` tambem faz `if err := ctx.Err();
err != nil { return nil, err }`. As 300 notas falham a leitura, caem no ramo
`skipped++` de `overflow.go:65`, e `updated` continua **0**. Com `updated < 300`
o teste passaria sob a mutacao (exit 1: regra escrita, nao verificada).

O observavel honesto de "parou cedo" e "nao processou entrada nenhuma": com o
ctx ja cancelado, `Reconcile` tem de devolver `0, 0, 0`. A saida do `mutate.ps1`
abaixo mostra `skipped=300` sob a mutacao — isto e, a prova de que o desvio era
necessario esta na propria prova.

## Prova de mutacao do Step 1

Comando, como rodou:

```
pwsh -File scripts/mutate.ps1 -Path internal/vault/walk.go \
  -Anchor "if ctxErr := ctx.Err(); ctxErr != nil {" \
  -Replacement "if ctxErr := ctx.Err(); ctxErr != nil && false {" \
  -Test TestReconcileCtxCanceladoParaAntesDeProcessarTudo \
  -Package ./internal/watcher/
```

Saida:

```
[...] Mutando internal/vault/walk.go
      - if ctxErr := ctx.Err(); ctxErr != nil {
      + if ctxErr := ctx.Err(); ctxErr != nil && false {

[...] go test -race -run TestReconcileCtxCanceladoParaAntesDeProcessarTudo ./internal/watcher/
----------------------------------------------------------------------
--- FAIL: TestReconcileCtxCanceladoParaAntesDeProcessarTudo (0.86s)
    overflow_test.go:276: Reconcile processou entradas com o ctx ja cancelado antes de comecar: updated=0 removed=0 skipped=300 (quer 0, 0, 0 nas tres); o cancelamento nao e respeitado
FAIL
FAIL	github.com/jonyd/gobsidian/internal/watcher	1.959s
FAIL
----------------------------------------------------------------------
[OK] internal/vault/walk.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

Exit code: **0** (o esperado — o teste reprovou sob mutacao).

Nota do brief acatada: `internal/watcher/overflow.go` **nao** contem check de
ctx proprio; `Reconcile` depende do check de `v.Walk`. Por isso o `-Path` e
`internal/vault/walk.go` e nao `overflow.go`.

## Prova do Step 4 (as duas guardas novas)

Guarda de tamanho — troquei `b` por `[]byte("{}")` em
`TestLinkGraph_DirectionParameter`:

```
=== RUN   TestLinkGraph_DirectionParameter
    schema_params_test.go:176: resposta sem arestas; o teste nao exercitou nada
--- FAIL: TestLinkGraph_DirectionParameter (0.03s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.145s
FAIL
```

Guarda de Unmarshal — troquei `b` por `[]byte("isto nao e json")` no mesmo
sitio:

```
=== RUN   TestLinkGraph_DirectionParameter
    schema_params_test.go:171: StructuredContent nao e o JSON esperado: invalid character 'i' looking for beginning of value
        isto nao e json
--- FAIL: TestLinkGraph_DirectionParameter (0.02s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.482s
FAIL
```

Restaurado (as duas edicoes temporarias eram uma linha `b = ...` acrescentada e
removida):

```
$ git diff --stat internal/mcpsrv/schema_params_test.go
 internal/mcpsrv/schema_params_test.go | 27 ++++++++++++++++++++++++---
 1 file changed, 24 insertions(+), 3 deletions(-)
$ go test ./internal/mcpsrv/ -run 'TestLinkGraph_'
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	1.181s
```

24 insercoes / 3 delecoes = tres sitios, cada um trocando uma linha
(`_ = json.Unmarshal(...)`) por oito. Nenhum residuo da prova.

## Verificacoes do brief

```
$ go test ./internal/index/ -run 'TestBuildSkipsUnreadableFile|TestNormaliza' -v
testing: warning: no tests to run
PASS
ok  	github.com/jonyd/gobsidian/internal/index	0.856s [no tests to run]

$ go test ./internal/service/ -run TestReadNoteCloudOnly -v
testing: warning: no tests to run
PASS
ok  	github.com/jonyd/gobsidian/internal/service	0.848s [no tests to run]
```

O segundo nao imprime `SKIP`: a funcao foi apagada, nao pulada.

## `verify.ps1`

Rodado inteiro (sem `-SkipCross`/`-SkipNet`/`-SkipLint`), 02:37 -> 02:42.
13 de 13 etapas `[OK]`. Ultimas linhas:

```
[...] 13. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Exit code 0.

Ressalva de ambiente, para nao inventar limpeza que nao houve: o gate de orfaos
do orquestrador (`scripts/test_orphans.ps1`, cinco processos) estava rodando
durante esta bateria. A etapa 3 (tetos de latencia) passou **sob** essa
contencao — um teto que passa com a maquina carregada e uma evidencia mais forte,
nao mais fraca. Nenhum numero foi medido nesta tarefa.

## Arquivos tocados

- `internal/watcher/overflow_test.go` — `TestReconcile_CtxCancelStopsEarly`
  substituido por `TestReconcileCtxCanceladoParaAntesDeProcessarTudo`
- `internal/index/build_test.go` — `TestBuildSkipsUnreadableFile` apagado,
  comentario no lugar dizendo por que e onde esta a cobertura real
- `internal/index/normalizacao_equivalente_test.go` — `git rm`
- `internal/mcpsrv/schema_params_test.go` — tres sitios ganham checagem de erro
  de `Unmarshal` e guarda de lista vazia
- `internal/service/read_test.go` — `TestReadNoteCloudOnlyFails` apagado,
  comentario no lugar

## Fora do escopo

- `schema_params_test.go:101` e `:128` tambem descartam o erro de
  `json.Unmarshal` (`_ = json.Unmarshal(bDefault, &dataDefault)` e
  `_ = json.Unmarshal(bTags, &dataTags)`), mas ja afirmam sobre `len(...)` nos
  dois sentidos (`== 0` e `!= 0`), entao nao sao testes que nao podem falhar. O
  brief lista tres sitios; deixei estes dois como estao. Um `Unmarshal` que
  falha ali produziria `len(...) == 0` e o teste reprovaria — pela mensagem
  errada, mas reprovaria.
- `b, _ := json.Marshal(res.StructuredContent)` continua descartando o erro nos
  cinco sitios. `json.Marshal` de um `map[string]any` vindo do SDK so falha com
  tipo nao serializavel; nao e o defeito que a tarefa nomeia.
- Nenhuma mudanca em codigo de producao. `internal/vault/walk.go` foi mutado e
  restaurado pelo proprio `mutate.ps1`, que confere SHA-256.
