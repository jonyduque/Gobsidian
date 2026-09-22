# Final fix round — report (Tasks 182–185 branch, HEAD ed24393 -> 92b6af5)

> Editado em 2026-09-22 pelo orquestrador: os dois blocos de diff passaram a
> cerca de quatro crases (o hunk do `CLAUDE.md` traz uma linha " ```" de
> contexto, que fechava a cerca de tres no meio e deixava o Markdown — e o
> auditor — sem os cabecalhos que vinham depois), e as secoes RED/GREEN e
> Mutacao ganharam a resposta que faltava: nesta rodada nao houve teste nem
> regra nova, e o relatorio original nao dizia isso em lugar nenhum. O
> conteudo de 2026-09-06 nao foi alterado.

## Progresso

- 22:41 — HEAD confirmado em ed24393f506d76407890eb68e2dbc53ced3bd9c6, igual ao
  esperado pelo brief. `git status` conferido: nada tocado nos tres arquivos
  alvo ainda.
- 22:42 — F1 aplicado: comentario do `if req.DryRun {` em
  `internal/service/write.go:544` reescrito, separando os dois papeis da
  origem (nota movida vs. citante de si mesma).
- 22:42 — F3 aplicado: uma frase adicionada ao comentario "Ordem
  deterministica" em `internal/service/broken.go:123`, sobre a perna Source
  vir herdada de `index.NotePaths()`. (Uma primeira tentativa usou
  "propósito" com acento — corrigido para "proposito" antes de seguir, para
  bater com o resto do arquivo, sem acentos.)
- 22:42 — F2 aplicado em tres pontos de `CLAUDE.md`: frase de abertura do
  bloco do grafo (de "Duas linhas mudaram" para "Tres linhas mudaram",
  citando a Task 183), a linha do `service` no grafo (+`text`), e o paragrafo
  de justificativa novo logo apos o paragrafo do `writer → text`.
- 22:42 — F2 aplicado em `docs/superpowers/plans/2026-09-06-broken-links.md:15`
  (arquivo UNTRACKED, editado mas NAO adicionado ao stage — o orquestrador
  comita).
- 22:42 — `gofmt -l internal/service` sem saida (limpo).
- 22:42 — `go build ./...` OK, sem saida.
- 22:42 — `go vet ./internal/service/` OK, sem saida.
- 22:42 — `go test ./internal/service/ -count=1` -> `ok  ...  16.978s`.
- 22:43 — inicio de `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet`, em
  foreground.
- 22:49 — `verify.ps1` terminou: `[OK] Bateria completa. Pode commitar.`
  EXIT=0.
- 22:49 — validacao de encoding UTF-8 dos dois `.md` editados: ambos `[OK]`.
- 22:49 — `git add internal/service/write.go internal/service/broken.go
  CLAUDE.md` (so os tres caminhos explicitos do brief; `git status`
  conferido depois — nenhum outro arquivo staged).
- 22:49 — mensagem de commit escrita em
  `.superpowers/sdd/2026-09-06-broken-links/final-fix-commit-msg.txt`.
- 22:49 — `git commit -F .superpowers/sdd/2026-09-06-broken-links/final-fix-commit-msg.txt`
  -> commit `92b6af5db7b9858b214b97981c5840eb60e118bd` criado, sem hook
  bloqueando. `3 files changed, 30 insertions(+), 10 deletions(-)`.

## Diff (`git show --stat HEAD`)

````
commit 92b6af5db7b9858b214b97981c5840eb60e118bd
Author: jonyduque <jonyduque@hotmail.com>
Date:   Sun Sep 6 22:49:15 2026 -0300

    docs: the dry-run comment and the graph catch up with the code

    note_move's dry-run comment said the origin never enters diffs; since a
    moved note that cites itself is rewritten, it does, under the old path.
    The dependency graph in CLAUDE.md claimed to be re-extracted today and
    missed service -> text, which vault_broken_links' prefix filter added.
    The order comparator in broken.go now says its Source leg is inherited
    from index.NotePaths and repeated on purpose.

    Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
    Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj

 CLAUDE.md                  | 15 +++++++++++----
 internal/service/broken.go |  5 ++++-
 internal/service/write.go  | 20 +++++++++++++++-----
 3 files changed, 30 insertions(+), 10 deletions(-)
````

## Diff (`git show HEAD -- internal/ CLAUDE.md`)

````diff
diff --git a/CLAUDE.md b/CLAUDE.md
index 2125e22..46dd014 100644
--- a/CLAUDE.md
+++ b/CLAUDE.md
@@ -108,9 +108,10 @@ scripts/           gates e utilitários PowerShell — ver Comandos
 
 Grafo de dependências, acíclico e **re-extraído dos imports de produção em
 2026-09-06** — `GOOS=windows go list -f '{{.Imports}}'` pacote a pacote, que NÃO
-enxerga arquivo `_test.go`. Duas linhas mudaram desde 2026-09-02: a do `writer`
-e a do `boot`, que é nova — e a do `boot` ganhou `lifecycle` no mesmo dia
-(Task 177). As justificativas estão logo abaixo do bloco:
+enxerga arquivo `_test.go`. Três linhas mudaram desde 2026-09-02: a do `writer`,
+a do `boot`, que é nova — e a do `boot` ganhou `lifecycle` no mesmo dia
+(Task 177) —, e a do `service`, que ganhou `text` em 2026-09-06 (Task 183).
+As justificativas estão logo abaixo do bloco:
 
 ```
 text  vault  config  console  lifecycle      folhas
@@ -120,7 +121,7 @@ writer   → parser, text, vault
 index    → parser, text, vault
 search   → index, parser, text, vault
 watcher  → index, search, vault
-service  → index, parser, search, vault, writer
+service  → index, parser, search, text, vault, writer
 mcpsrv   → config, index, parser, service, vault
 boot     → config, index, lifecycle, search, service, vault, watcher
 daemon   → config, ipc, mcpsrv
@@ -152,6 +153,12 @@ como dois arquivos. A conta mudou-se para `text.ChaveDeCaminho` porque `writer`
 não importa `index` e não vai importar. `text` continua folha: ganhou
 `path/filepath`, que é stdlib.
 
+`service → text` é de 2026-09-06 (Task 183): o filtro `prefix` de
+`vault_broken_links` compara caminho de origem com o prefixo pedido, e
+comparar caminho é `text.ChaveDeCaminho` — a mesma conta do `writer`
+(Task 169) e do `index`. Uma comparação local faria `Sub/` e `sub/` serem
+duas pastas aqui e uma lá. `text` continua folha; o grafo continua acíclico.
+
 Quatro arestas existem **só em teste**, e ficam fora do grafo acima de
 propósito — teste pode montar o mundo inteiro sem que isso vire acoplamento do
 produto:
diff --git a/internal/service/broken.go b/internal/service/broken.go
index 1c4fad2..a8a3e05 100644
--- a/internal/service/broken.go
+++ b/internal/service/broken.go
@@ -122,7 +122,10 @@ func (s *Service) BrokenLinks(_ context.Context, req BrokenLinksRequest) (Broken
 
 	// Ordem deterministica: a origem e depois a posicao no corpo. Uma lista
 	// paginada cuja ordem varia entre chamadas repete item numa pagina e some
-	// com outro na seguinte.
+	// com outro na seguinte. A perna Source ja vem ordenada de
+	// index.NotePaths(); o comparador a repete de proposito, para que a
+	// garantia seja local a esta funcao e nao dependa do invariante de outro
+	// pacote.
 	slices.SortFunc(achados, func(a, b achadoQuebrado) int {
 		if c := cmp.Compare(a.link.Source, b.link.Source); c != 0 {
 			return c
diff --git a/internal/service/write.go b/internal/service/write.go
index a2a4fe1..2a34716 100644
--- a/internal/service/write.go
+++ b/internal/service/write.go
@@ -542,11 +542,21 @@ func (s *Service) MoveNote(ctx context.Context, req MoveNoteRequest) (MoveNoteRe
 	}
 
 	if req.DryRun {
-		// A origem nao entra em diffs: mover nao altera o conteudo dela, e
-		// UnifiedDiff de um texto contra ele mesmo e "" — um item vazio que
-		// dizia "esta nota nao muda" sobre a nota que muda de lugar. A
-		// leitura continua para que uma origem ilegivel falhe aqui, e nao so
-		// na execucao real.
+		// A origem tem dois papeis aqui, e sao diferentes:
+		//
+		// Como nota MOVIDA, ela nao ganha entrada propria em diffs: mover
+		// nao reescreve o corpo dela por si so, e um item vazio diria "esta
+		// nota nao muda" sobre a nota que muda de lugar. A leitura abaixo
+		// continua sendo so a checagem "origem ilegivel falha aqui, e nao
+		// so na execucao real" — nao alimenta diffs.
+		//
+		// Como CITANTE de si mesma com alvo escrito (ex.: "[[a]]" ou
+		// "[x](a.md)" dentro de a.md), ela entra em diffs normalmente, sob
+		// o caminho ANTIGO — o que existe em disco durante o dry-run —, e
+		// nao sob o novo, como a execucao real reporta em rewritten. Conta
+		// em links_updated nos dois modos. Auto-referencia so de ancora
+		// ("[[#h]]", "[x](#h)") nao entra em nenhum dos dois: o guarda
+		// Target == "" no topo do loop de citantes barra antes.
 		absFrom := s.vault.Abs(canonicalFrom)
 		if _, err := os.ReadFile(absFrom); err != nil {
 			return MoveNoteResult{}, Errorf(CodeInternal,
````

## RED / GREEN — sem ciclo de teste nesta rodada

Nenhum teste novo, nenhum teste alterado: os tres achados (F1, F2, F3) eram
de COMENTARIO e de DOCUMENTACAO — um comentario em `write.go`, uma frase em
`broken.go`, o grafo e uma justificativa em `CLAUDE.md`. Nao ha comportamento
que um teste pudesse ficar vermelho antes e verde depois. A suite existente
rodou inteira (abaixo, em Verificacao) para provar que a mudanca de texto nao
quebrou compilacao nem teste.

## Mutacao — sem regra nova a mutar

Prova de mutacao e para regra reivindicada por teste; esta rodada nao
reivindicou nenhuma. O que se verificou foi que o texto novo diz a verdade
sobre o codigo: o comentario de dry-run foi conferido contra o laco de
citantes em `write.go` (o guarda `Target == ""` existe e barra a
auto-referencia so de ancora), e a aresta `service -> text` do grafo foi
conferida por `GOOS=windows go list -f '{{.Imports}}' ./internal/service`.

## Verificacao

`gofmt -l internal/service` — sem saida (arquivos ja formatados).

`go build ./...` — sem saida, exit 0.

`go vet ./internal/service/` — sem saida, exit 0.

`go test ./internal/service/ -count=1`:
```
ok  	github.com/jonyd/gobsidian/internal/service	16.978s
```

`pwsh -File scripts/verify.ps1 -SkipCross -SkipNet` (ultimas linhas):
```
[...] 9. check_tool_params
Carregamento: 1007,8 ms

[OK] check_tool_params
[...] 10. check_doc_refs
[OK] check_doc_refs
[...] 11. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
EXIT=0
```
(Etapa 3, contagem de testes pulados, reportou os 6 skips ja conhecidos do
projeto — `TestAjudanteSeguraTrava`, `TestListenRestringePermissaoUnix`,
`TestSignalCancelsContext`, `TestPerfilDeHeapServindo`,
`TestWriteAtomicPreservaOModoDoAlvo`, `TestNew_FailsOnUnwatchablePath` — a
etapa so informa, nao reprova, como o CLAUDE.md documenta.)

Validacao de encoding:
```
[OK] CLAUDE.md UTF-8 valido
[OK] plan UTF-8 valido
```

## Commit

SHA: `92b6af5db7b9858b214b97981c5840eb60e118bd`

Staged por caminho explicito (`git add internal/service/write.go
internal/service/broken.go CLAUDE.md`) — confirmado via `git status --short`
antes do commit que nenhum outro arquivo entrou no stage. O plano
(`docs/superpowers/plans/2026-09-06-broken-links.md`) permanece untracked,
editado mas nao adicionado, conforme o brief.

Nenhum hook bloqueou o commit.

## Status

DONE. Commit `92b6af5db7b9858b214b97981c5840eb60e118bd`. Tres correcoes de
texto (comentario de dry-run em write.go, grafo+justificativa em CLAUDE.md,
frase no comparador de broken.go) mais a atualizacao do plano untracked;
verify.ps1 -SkipCross -SkipNet terminou EXIT=0 e nenhum comportamento mudou.
