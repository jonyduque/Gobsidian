# Task 182 — Link de ancora nao e alvo ausente

## Progresso

20:21 - inicio - brief lido; ast.go, ext_wikilink.go, types.go, resolve.go lidos
20:23 - Step 1-2 - links_anchor_test.go escrito; FAIL nos 4 casos com ancora, como esperado
20:24 - Step 3 - splitAnchor em ast.go, splitWikilink passou a chama-la, comentario de Link.Anchor; PASS
20:24 - Step 4-5 - resolve_anchor_test.go escrito; FAIL nos 4 links de ancora (target_missing)
20:24 - Step 6 - resolveTarget passou a receber anchor (3 chamadores), IndexCacheParserVersion 1->2; PASS
20:25 - Step 7 - anchorMarkdown no writer; teste falhou antes, passa depois
20:25 - Step 8 - suite ampla com -race: parser/index/writer/service ok; nenhum golden mudou
20:29 - Step 9 - quatro provas de mutacao, todas exit 0 (teste reprovou sob mutacao)
20:31 - Step 10 - verify.ps1 completo: 14 etapas verdes, EXIT=0
20:36 - Step 10 - relatorio escrito; commit a seguir
20:43 - Step 10 - commit 8e684d0 (10 arquivos, +240 -17); ledger anexado
20:46 - fim - audit_reports.ps1 182 sem achado no relatorio; git status limpo em internal/ e testdata/

## Fix round 1

### Progresso

20:55 - inicio - review-182.md lido; F1, F2, F3 + nits F6, F7 e o texto do F4
20:57 - F1/F3 RED - anchor_selfref_test.go: move com ENOENT, delete acusando a propria nota
20:57 - F2 RED - TestMoveNote_LinkSoDeAncoraSegueANota: Resolved="a.md" apos mover para b/a.md
20:58 - fixes - write.go (F1 e F3), update.go (F2), ast.go strings.Cut, nits F6/F7 no teste do indice
20:58 - sonda - medido que "[[a]]" dentro de a.md ja derrubava note_move ANTES desta task; teste do F1 ajustado e o achado registrado
20:59 - GREEN - os tres testes novos passam; gofmt e go vet limpos
21:03 - mutacao - tres provas, todas exit 0
21:04 - gate - verify.ps1 completo: 14 etapas verdes, EXIT=0
21:07 - F4 - a frase "parity_test verde => nao ha divergencia" corrigida na secao Concerns
21:08 - commit - c987ff1 (6 arquivos, +226 -10); git status limpo em internal/

### Status desta rodada

DONE_WITH_CONCERNS — F1, F2 e F3 corrigidos com teste e prova de mutacao; os
nits F6, F7 e o `strings.Cut` do F8/diagnostico pegaram carona; o texto do F4
foi corrigido na secao Concerns. Uma ressalva nova, medida: ver "Achado
pre-existente".

### Commit

`c987ff165c2455d8c8b3fcf67212c7bd5fe85384` (`c987ff1`) —
`fix(links): keep note_move and note_delete correct for anchor-only self-links`

6 arquivos, +226 -10. Novo commit sobre 8e684d0; nada foi emendado nem resetado.

### Files changed

- `C:\Users\jonyd\Projetos\Gobsidian\internal\service\write.go` — F1: guarda
  `rl.Target == ""` no laco que colhe replacements de `MoveNote`; F3:
  `bl.From == canonical` pula a propria nota no relatorio de `DeleteNote`
- `C:\Users\jonyd\Projetos\Gobsidian\internal\index\update.go` — F2: logo apos
  `n := &movida`, todo `Resolved == oldPath` vira `newPath`
- `C:\Users\jonyd\Projetos\Gobsidian\internal\parser\ast.go` — `splitAnchor` usa
  `strings.Cut`
- `C:\Users\jonyd\Projetos\Gobsidian\internal\service\anchor_selfref_test.go`
  (novo) — os testes de F1 e F3
- `C:\Users\jonyd\Projetos\Gobsidian\internal\index\move_test.go` — o teste de F2
- `C:\Users\jonyd\Projetos\Gobsidian\internal\index\resolve_anchor_test.go` —
  nits F6 (`vault.New` com erro conferido) e F7 (`![[#Topo]]` na tabela, agora
  sete links)

### RED — antes das correcoes

F1 e F3, `go test ./internal/service -run '...' -v`:

```
=== RUN   TestMoveNote_LinkSoDeAncoraNaoQuebraOMove
    anchor_selfref_test.go:33: MoveNote: lendo nota "a.md": open C:\Users\...\a.md: The system cannot find the file specified.
--- FAIL: TestMoveNote_LinkSoDeAncoraNaoQuebraOMove (0.02s)
=== RUN   TestDeleteNote_NaoAcusaAPropriaNotaApagada
    anchor_selfref_test.go:86: BrokenLinks = [a.md ref.md]; a propria nota apagada nao pode estar na lista
    anchor_selfref_test.go:90: BrokenLinks = [a.md ref.md]; quer [ref.md]
    anchor_selfref_test.go:95: BrokenAnchors tem {From:a.md To:a.md Anchor:Topo}; a nota apagada nao aponta mais para nada
    anchor_selfref_test.go:95: BrokenAnchors tem {From:a.md To:a.md Anchor:Nada}; a nota apagada nao aponta mais para nada
    anchor_selfref_test.go:95: BrokenAnchors tem {From:a.md To:a.md Anchor:Topo}; a nota apagada nao aponta mais para nada
    anchor_selfref_test.go:95: BrokenAnchors tem {From:a.md To:a.md Anchor:Nada}; a nota apagada nao aponta mais para nada
--- FAIL: TestDeleteNote_NaoAcusaAPropriaNotaApagada (0.02s)
FAIL	github.com/jonyd/gobsidian/internal/service	0.722s
```

F2, `go test ./internal/index -run TestMoveNote_LinkSoDeAncoraSegueANota -v`:

```
=== RUN   TestMoveNote_LinkSoDeAncoraSegueANota
    move_test.go:300: Resolved="a.md" State=ok, quer "b/a.md"/ok
    move_test.go:305: backlinks de b/a.md ainda tem From="a.md", uma nota que nao existe mais
--- FAIL: TestMoveNote_LinkSoDeAncoraSegueANota (0.02s)
FAIL	github.com/jonyd/gobsidian/internal/index	0.665s
```

**Honestidade sobre este RED de F1:** ele foi colhido de uma PRIMEIRA versao do
teste, que tambem punha `[[a]]` dentro da nota movida. Essa versao continuou
falhando depois da correcao, e o motivo esta em "Achado pre-existente" abaixo. O
teste foi reescrito sem o `[[a]]`, e o RED da versao final e a prova de mutacao
1, que remove exatamente o guarda e produz o mesmo ENOENT.

### GREEN — depois

```
=== RUN   TestMoveNote_LinkSoDeAncoraNaoQuebraOMove
--- PASS: TestMoveNote_LinkSoDeAncoraNaoQuebraOMove (0.03s)
=== RUN   TestDeleteNote_NaoAcusaAPropriaNotaApagada
--- PASS: TestDeleteNote_NaoAcusaAPropriaNotaApagada (0.01s)
ok  	github.com/jonyd/gobsidian/internal/service	0.893s
```

```
=== RUN   TestMoveNote_UpdatesIncomingBacklinks
--- PASS: TestMoveNote_UpdatesIncomingBacklinks (0.01s)
=== RUN   TestMoveNote_UpdatesOutgoingBacklinks
--- PASS: TestMoveNote_UpdatesOutgoingBacklinks (0.01s)
=== RUN   TestMoveNote_LinkSoDeAncoraSegueANota
--- PASS: TestMoveNote_LinkSoDeAncoraSegueANota (0.01s)
=== RUN   TestMoveNote_ReprocessesBrokenLinks
--- PASS: TestMoveNote_ReprocessesBrokenLinks (0.01s)
=== RUN   TestAncoraNaMesmaNota
--- PASS: TestAncoraNaMesmaNota (0.01s)
ok  	github.com/jonyd/gobsidian/internal/index	0.797s
```

`TestAncoraNaMesmaNota` acima ja e a versao com os sete links, incluindo o
`![[#Topo]]` do nit F7.

### Achado pre-existente, medido nesta rodada

**Uma nota que cita a si mesma com alvo ESCRITO derruba `note_move` pelo mesmo
ENOENT do F1, e isso NAO e da Task 182.** A revisao ja dizia que o mecanismo era
pre-existente; eu medi.

Sonda: uma nota cujo unico conteudo era `# A\n\nVeja [[a]].\n`, movida de `a.md`
para `sub/a.md`, sem nenhuma ancora envolvida:

```
=== RUN   TestSondaAutoLinkComAlvoEscrito
    zz_sonda_test.go:16: MoveNote: lendo nota "a.md": open C:\Users\...\a.md: The system cannot find the file specified.
--- FAIL: TestSondaAutoLinkComAlvoEscrito (0.02s)
```

A sonda foi apagada depois de medida; nao esta no commit. O caminho e o mesmo
para antes e depois de 8e684d0, porque nada na Task 182 toca link com alvo
escrito.

**Consequencia para o teste do F1:** a revisao pede que ele fixe as DUAS metades
do discriminante — que a ancora seja pulada e que `[[a]]` continue sendo
reescrita. A segunda metade **nao e testavel aqui** sem consertar tambem o
defeito pre-existente, que o brief nao autoriza. O teste prova a primeira metade
e, com `ref.md`, que o guarda nao desligou a reescrita de citante nenhum; o
limite esta escrito no comentario do proprio teste. **Isso merece task propria.**

### Mutation proofs desta rodada

Tres, `scripts/mutate.ps1`, todas `EXIT=0` (o script sai 0 quando o teste
REPROVA sob mutacao).

#### 1. Removi o guarda `rl.Target == ""` de `MoveNote` (F1)

```
[...] Mutando internal/service/write.go
      - 					if rl.Target == "" {\n						continue\n					}\n\n
      +

[...] go test -race -run TestMoveNote_LinkSoDeAncoraNaoQuebraOMove ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestMoveNote_LinkSoDeAncoraNaoQuebraOMove (0.01s)
    anchor_selfref_test.go:45: MoveNote: lendo nota "a.md": open C:\Users\...\a.md: The system cannot find the file specified.
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.062s
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

#### 2. Removi a reescrita de `Resolved` em `index.MoveNote` (F2)

```
[...] Mutando internal/index/update.go
      - 	for i := range n.Links {\n		if n.Links[i].Resolved == oldPath {\n			n.Links[i].Resolved = newPath\n		}\n	}\n
      +

[...] go test -race -run TestMoveNote_LinkSoDeAncoraSegueANota ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestMoveNote_LinkSoDeAncoraSegueANota (0.01s)
    move_test.go:300: Resolved="a.md" State=ok, quer "b/a.md"/ok
    move_test.go:305: backlinks de b/a.md ainda tem From="a.md", uma nota que nao existe mais
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.964s
----------------------------------------------------------------------
[OK] internal/index/update.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

#### 3. Removi o pulo da propria nota em `DeleteNote` (F3)

```
[...] Mutando internal/service/write.go
      - 			if bl.From == canonical {\n				continue\n			}\n\n
      +

[...] go test -race -run TestDeleteNote_NaoAcusaAPropriaNotaApagada ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestDeleteNote_NaoAcusaAPropriaNotaApagada (0.02s)
    anchor_selfref_test.go:101: BrokenLinks = [a.md ref.md]; a propria nota apagada nao pode estar na lista
    anchor_selfref_test.go:105: BrokenLinks = [a.md ref.md]; quer [ref.md]
    anchor_selfref_test.go:110: BrokenAnchors tem {From:a.md To:a.md Anchor:Topo}; a nota apagada nao aponta mais para nada
    anchor_selfref_test.go:110: BrokenAnchors tem {From:a.md To:a.md Anchor:Nada}; a nota apagada nao aponta mais para nada
    anchor_selfref_test.go:110: BrokenAnchors tem {From:a.md To:a.md Anchor:Topo}; a nota apagada nao aponta mais para nada
    anchor_selfref_test.go:110: BrokenAnchors tem {From:a.md To:a.md Anchor:Nada}; a nota apagada nao aponta mais para nada
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.196s
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

`strings.Cut` nao ganhou prova propria: e refatoracao de equivalencia, e as
quatro provas da rodada anterior — inclusive a da ORDEM
(`TestMarkdownLinkNaoSeparaNoPercent23`) — continuam cobrindo `splitAnchor`, e
passaram depois da troca.

### verify.ps1 (tail) desta rodada

```
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
EXIT=0
```

A etapa 3 seguiu informando os mesmos 6 testes pulados da rodada anterior;
nenhum dos seis toca parser, indice, writer ou service.

### Achados da revisao nao endereçados, e por que

- **F5** (o hook `pre_commit_docs.ps1` se satisfaz com um comentario de shell): a
  propria revisao diz que merece ticket proprio, fora desta tarefa. Nao toquei
  no hook. Esta rodada usou a mesma escotilha, declarada do mesmo jeito.
- **F8** (`[x](b.md#)` perde o `#` final): a revisao nao propoe mudar, so
  registra. Nao mudei.
- A linha em `docs/TOOLS.md` sobre aresta com `source == target` no `link_graph`:
  a revisao a enderecou a Task 183/184. Esta rodada e `[sem-doc]`.

---

## Status

DONE_WITH_CONCERNS — o codigo esta pronto e o gate verde; as ressalvas
(desvio de desenho justificado, gopls indisponivel, ledger) estao ao final.

## Commits

`8e684d0f5be54d7a157da4d9ead64484718ee2c9` (`8e684d0`) —
`fix(links): anchor links resolve to their note instead of counting as missing targets`

10 arquivos, +240 -17. Staged um a um por caminho explicito; nenhum `git add -A`.

O hook `scripts/pre_commit_docs.ps1` barrou a primeira tentativa: ele procura
`[sem-doc]` no **comando**, nao no arquivo passado a `git commit -F`. A segunda
tentativa declarou a escotilha tambem no comando, num comentario de shell; a
mensagem gravada continua com o `[sem-doc]` no corpo, como o brief pede.

## Files changed

Producao:
- `C:\Users\jonyd\Projetos\Gobsidian\internal\parser\ast.go` — `splitAnchor` nova; ramos `*gast.Link` e `*gast.Image` de `collect` separam a ancora ANTES de `PercentDecode` e decodificam as duas partes
- `C:\Users\jonyd\Projetos\Gobsidian\internal\parser\ext_wikilink.go` — `splitWikilink` passou a chamar `splitAnchor`; ficou so com o `|` e o aparo de espaco
- `C:\Users\jonyd\Projetos\Gobsidian\internal\parser\types.go` — comentario de `Link.Anchor` reescrito (vale para os tres `Kind`)
- `C:\Users\jonyd\Projetos\Gobsidian\internal\index\resolve.go` — `resolveTarget` recebe `anchor`; `target==""` com ancora resolve para a origem, `ViaPath`, `LinkOK`
- `C:\Users\jonyd\Projetos\Gobsidian\internal\index\update.go` — os dois outros chamadores de `resolveTarget` passam `Anchor`
- `C:\Users\jonyd\Projetos\Gobsidian\internal\index\persist.go` — `IndexCacheParserVersion` 1 -> 2, com o motivo
- `C:\Users\jonyd\Projetos\Gobsidian\internal\writer\linkrewrite.go` — `anchorMarkdown` nova; os dois ramos Markdown de `BuildLinkText` reemitem `#ancora`

Teste:
- `C:\Users\jonyd\Projetos\Gobsidian\internal\parser\links_anchor_test.go` (novo)
- `C:\Users\jonyd\Projetos\Gobsidian\internal\index\resolve_anchor_test.go` (novo)
- `C:\Users\jonyd\Projetos\Gobsidian\internal\writer\linkrewrite_test.go` (caso novo)

Golden: **nenhum mudou**, e isso foi medido, nao suposto. `TestGolden` passou sem
`-update`, e `grep -rn ']([^)]*#' testdata/parser --include=*.md` nao devolve
linha nenhuma — nenhuma fixture tem link Markdown com ancora, entao a mudanca
nao tinha por onde aparecer ali. `git status --porcelain testdata/` saiu vazio.

## Desvio de desenho, com o motivo

O brief dizia: "Como `resolveTarget` nao recebe a ancora, a decisao mora no
chamador (a funcao que percorre os links e chama `resolveAnchor`)".

**Sao TRES essas funcoes**, nao uma — `resolveAllLinks` (`resolve.go:97`),
`resolveLinksForNoteLocked` (`update.go:271`) e `reprocessNoteLinksLocked`
(`update.go:431`). Por a decisao no chamador escreveria a mesma condicao tres
vezes, que e exatamente o que "uma conta por regra" proibe — e a Task 182 existe
por causa de uma separacao de `#` que existia em um lugar so e faltava no outro.

Entao `resolveTarget` ganhou o parametro `anchor` e a decisao ficou onde o brief
manda no primeiro periodo do Step 6 ("o ramo `target == ""` passa a distinguir").
Os tres chamadores so passam `Links[i].Anchor`. `ViaPath` foi a constante
escolhida, como o brief sugere, e o comentario diz por que: a nota de origem ja
E um caminho do cofre. Nenhum estado novo foi inventado.

## Test output

Cada uma das tres regras foi escrita RED antes de existir codigo que a
satisfizesse, e so entao GREEN. As saidas abaixo sao as reais, nesta ordem.

### RED — parser (Passo 2), antes da correcao

```
=== RUN   TestMarkdownLinkSeparaAncora
=== RUN   TestMarkdownLinkSeparaAncora/nota_e_heading
    links_anchor_test.go:28: Target="b.md#Sec" Anchor="", queria "b.md"/"Sec"
=== RUN   TestMarkdownLinkSeparaAncora/so_ancora
    links_anchor_test.go:28: Target="#Topo" Anchor="", queria ""/"Topo"
=== RUN   TestMarkdownLinkSeparaAncora/ancora_percent-encoded
    links_anchor_test.go:28: Target="b.md#Seção" Anchor="", queria "b.md"/"Seção"
=== RUN   TestMarkdownLinkSeparaAncora/embed_markdown
    links_anchor_test.go:28: Target="b.md#Sec" Anchor="", queria "b.md"/"Sec"
=== RUN   TestMarkdownLinkSeparaAncora/sem_ancora_fica_igual
=== RUN   TestMarkdownLinkSeparaAncora/URL_com_fragmento:_Target_sem_fragmento,_esquema_intacto
    links_anchor_test.go:28: Target="https://ex.com/p#f" Anchor="", queria "https://ex.com/p"/"f"
--- FAIL: TestMarkdownLinkSeparaAncora (0.00s)
FAIL	github.com/jonyd/gobsidian/internal/parser	1.047s
```

### GREEN — parser (Passo 3), depois

```
--- PASS: TestMarkdownLinkSeparaAncora (0.00s)
    --- PASS: TestMarkdownLinkSeparaAncora/nota_e_heading (0.00s)
    --- PASS: TestMarkdownLinkSeparaAncora/so_ancora (0.00s)
    --- PASS: TestMarkdownLinkSeparaAncora/ancora_percent-encoded (0.00s)
    --- PASS: TestMarkdownLinkSeparaAncora/embed_markdown (0.00s)
    --- PASS: TestMarkdownLinkSeparaAncora/sem_ancora_fica_igual (0.00s)
    --- PASS: TestMarkdownLinkSeparaAncora/URL_com_fragmento:_Target_sem_fragmento,_esquema_intacto (0.00s)
=== RUN   TestMarkdownLinkNaoSeparaNoPercent23
--- PASS: TestMarkdownLinkNaoSeparaNoPercent23 (0.00s)
ok  	github.com/jonyd/gobsidian/internal/parser	0.651s
```

### RED — indice (Passo 5), antes

```
=== RUN   TestAncoraNaMesmaNota
    resolve_anchor_test.go:60: link 1 (""#"Topo"): Resolved="" State=target_missing, quer "a.md"/ok
    resolve_anchor_test.go:60: link 2 (""#"Topo"): Resolved="" State=target_missing, quer "a.md"/ok
    resolve_anchor_test.go:60: link 3 (""#"Nada"): Resolved="" State=target_missing, quer "a.md"/anchor_missing
    resolve_anchor_test.go:60: link 4 (""#"Nada"): Resolved="" State=target_missing, quer "a.md"/anchor_missing
--- FAIL: TestAncoraNaMesmaNota (0.02s)
FAIL	github.com/jonyd/gobsidian/internal/index	0.746s
```

O link 0 (`[x](b.md#Sec)`) ja passava aqui: quem o consertou foi o Step 3, no
parser. Os que restam sao os de alvo vazio, que e o que o Step 6 conserta.

### GREEN — indice (Passo 6), depois

```
=== RUN   TestAncoraNaMesmaNota
--- PASS: TestAncoraNaMesmaNota (0.01s)
ok  	github.com/jonyd/gobsidian/internal/index	0.779s
```

### RED e GREEN — writer (Passo 7)

```
=== RUN   TestRewriteLinks_PreservaAncoraEmLinkMarkdown
    linkrewrite_test.go:98: obtido "Link: [x](c.md)\nEmbed: ![x](c.md)\nEspaco: [x](c.md)", quer "Link: [x](c.md#Sec)\nEmbed: ![x](c.md#Sec)\nEspaco: [x](c.md#Com%20Espaco)"
--- FAIL: TestRewriteLinks_PreservaAncoraEmLinkMarkdown (0.00s)
FAIL	github.com/jonyd/gobsidian/internal/writer	0.711s
```

```
=== RUN   TestRewriteLinks_PreservesAliasAndAnchor
--- PASS: TestRewriteLinks_PreservesAliasAndAnchor (0.00s)
=== RUN   TestRewriteLinks_PreservesSyntaxAndEmbed
--- PASS: TestRewriteLinks_PreservesSyntaxAndEmbed (0.00s)
=== RUN   TestRewriteLinks_PreservaAncoraEmLinkMarkdown
--- PASS: TestRewriteLinks_PreservaAncoraEmLinkMarkdown (0.00s)
=== RUN   TestRewriteLinks_MultipleOccurrencesInSameNote
--- PASS: TestRewriteLinks_MultipleOccurrencesInSameNote (0.00s)
=== RUN   TestRewriteLinks_LinkNoInicioENoFim
--- PASS: TestRewriteLinks_LinkNoInicioENoFim (0.00s)
=== RUN   TestRewriteLinks_RejectsInvalidOffsets
--- PASS: TestRewriteLinks_RejectsInvalidOffsets (0.00s)
=== RUN   TestRewriteLinks_PreservesBOMAndEOL
--- PASS: TestRewriteLinks_PreservesBOMAndEOL (0.00s)
ok  	github.com/jonyd/gobsidian/internal/writer	0.629s
```

Passo 8 — `go test ./internal/parser/... ./internal/index/... ./internal/writer/... ./internal/service/... -race`:

```
ok  	github.com/jonyd/gobsidian/internal/parser	2.168s
ok  	github.com/jonyd/gobsidian/internal/index	4.111s
ok  	github.com/jonyd/gobsidian/internal/writer	2.143s
ok  	github.com/jonyd/gobsidian/internal/service	45.992s
```

## Mutation proofs

Todas com `scripts/mutate.ps1`, no passado, com a saida real. As quatro sairam
`EXIT=0` — o script sai 0 quando o teste REPROVA sob mutacao, que e a prova.

### 1. Removi o split em `ast.go` (corpo de `splitAnchor`)

```
[...] Mutando internal/parser/ast.go
      - 	if i := strings.IndexByte(s, '#'); i >= 0 {\n		return s[:i], s[i+1:]\n	}\n
      +

[...] go test -race -run TestMarkdownLinkSeparaAncora ./internal/parser/
----------------------------------------------------------------------
--- FAIL: TestMarkdownLinkSeparaAncora (0.00s)
    --- FAIL: TestMarkdownLinkSeparaAncora/nota_e_heading (0.00s)
        links_anchor_test.go:28: Target="b.md#Sec" Anchor="", queria "b.md"/"Sec"
    --- FAIL: TestMarkdownLinkSeparaAncora/so_ancora (0.00s)
        links_anchor_test.go:28: Target="#Topo" Anchor="", queria ""/"Topo"
    --- FAIL: TestMarkdownLinkSeparaAncora/ancora_percent-encoded (0.00s)
        links_anchor_test.go:28: Target="b.md#Seção" Anchor="", queria "b.md"/"Seção"
    --- FAIL: TestMarkdownLinkSeparaAncora/embed_markdown (0.00s)
        links_anchor_test.go:28: Target="b.md#Sec" Anchor="", queria "b.md"/"Sec"
    --- FAIL: TestMarkdownLinkSeparaAncora/URL_com_fragmento:_Target_sem_fragmento,_esquema_intacto (0.00s)
        links_anchor_test.go:28: Target="https://ex.com/p#f" Anchor="", queria "https://ex.com/p"/"f"
FAIL
FAIL	github.com/jonyd/gobsidian/internal/parser	0.679s
----------------------------------------------------------------------
[OK] internal/parser/ast.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### 2. Removi o ramo `Target == "" && Anchor != ""` do indice

```
[...] Mutando internal/index/resolve.go
      - 		if anchor != "" {\n			return origin, ViaPath, LinkOK\n		}\n
      +

[...] go test -race -run TestAncoraNaMesmaNota ./internal/index/
----------------------------------------------------------------------
--- FAIL: TestAncoraNaMesmaNota (0.01s)
    resolve_anchor_test.go:60: link 1 (""#"Topo"): Resolved="" State=target_missing, quer "a.md"/ok
    resolve_anchor_test.go:60: link 2 (""#"Topo"): Resolved="" State=target_missing, quer "a.md"/ok
    resolve_anchor_test.go:60: link 3 (""#"Nada"): Resolved="" State=target_missing, quer "a.md"/anchor_missing
    resolve_anchor_test.go:60: link 4 (""#"Nada"): Resolved="" State=target_missing, quer "a.md"/anchor_missing
FAIL
FAIL	github.com/jonyd/gobsidian/internal/index	0.959s
----------------------------------------------------------------------
[OK] internal/index/resolve.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

O link 2 e `[[#Topo]]`, que e o caso que o brief manda ver falhar.

### 3. Removi a ancora de `BuildLinkText` (corpo de `anchorMarkdown`)

```
[...] Mutando internal/writer/linkrewrite.go
      - 	return "#" + encodeMarkdownTarget(anchor, origRaw)
      + 	return ""

[...] go test -race -run TestRewriteLinks_PreservaAncoraEmLinkMarkdown ./internal/writer/
----------------------------------------------------------------------
--- FAIL: TestRewriteLinks_PreservaAncoraEmLinkMarkdown (0.00s)
    linkrewrite_test.go:98: obtido "Link: [x](c.md)\nEmbed: ![x](c.md)\nEspaco: [x](c.md)", quer "Link: [x](c.md#Sec)\nEmbed: ![x](c.md#Sec)\nEspaco: [x](c.md#Com%20Espaco)"
FAIL
FAIL	github.com/jonyd/gobsidian/internal/writer	0.759s
----------------------------------------------------------------------
[OK] internal/writer/linkrewrite.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### 4. (extra) Inverti a ORDEM: decodifiquei antes de separar

O brief pede tres. Esta quarta existe porque a ordem "separar antes de
`PercentDecode`" e uma regra por si — o brief a escreve em maiusculas — e as
tres primeiras nao a cobrem: qualquer uma delas fica verde com a ordem trocada.

```
[...] Mutando internal/parser/ast.go
      - 	if i := strings.IndexByte(s, '#'); i >= 0 {
      + 	s = PercentDecode(s)\n	if i := strings.IndexByte(s, '#'); i >= 0 {

[...] go test -race -run TestMarkdownLinkNaoSeparaNoPercent23 ./internal/parser/
----------------------------------------------------------------------
--- FAIL: TestMarkdownLinkNaoSeparaNoPercent23 (0.00s)
    links_anchor_test.go:44: Target="C" Anchor=".md", queria "C#.md"/""
FAIL
FAIL	github.com/jonyd/gobsidian/internal/parser	0.689s
----------------------------------------------------------------------
[OK] internal/parser/ast.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

## verify.ps1 (tail)

`pwsh -File scripts/verify.ps1`, completo, foreground, `EXIT=0`:

```
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
EXIT=0
```

A etapa 3 informou 6 testes pulados (`TestAjudanteSeguraTrava`,
`TestListenRestringePermissaoUnix`, `TestSignalCancelsContext`,
`TestPerfilDeHeapServindo`, `TestWriteAtomicPreservaOModoDoAlvo`,
`TestNew_FailsOnUnwatchablePath`). Ela informa e nao reprova, e **nao medi** se
essa contagem mudou por causa desta tarefa — nenhum dos seis toca parser,
indice ou writer.

## Concerns

1. **Backlink de si mesma.** Com `[[#Topo]]` resolvendo para a propria nota, o
   link passa a existir como referencia resolvida da nota para ela mesma, e
   `note_backlinks` de `a.md` passa a lista-la.

   **Correcao de uma frase errada desta secao** (achado F4 da revisao): a versao
   anterior dizia que a suite verde, "incluindo `parity_test.go`", mostrava que
   **nao ha divergencia medida**. A frase e literalmente verdadeira e induz ao
   contrario do que quer dizer. `assertGraphMatches`
   (`internal/index/parity_test.go:101-129`) itera `ref.ResolvedLinks` do
   Obsidian e exige `nossos[alvo] >= querAoMenos`: a comparacao e **assimetrica
   de proposito**, e uma aresta A MAIS do nosso lado — que e exatamente o que o
   self-backlink cria — nao tem como reprovar esse teste. O instrumento nao mediu
   nada nesta direcao e nao podia.

   O que vale entao: **nao medido** contra cofre real. A suite verde nao e
   evidencia aqui, nem a favor nem contra. Se a Task 183 for contar ou listar
   links, isso precisa de medicao propria. Nao mudei o comportamento porque
   estaria fora do escopo do brief.

2. **`gopls` indisponivel nesta sessao.** O `implementador.md` manda usar
   `go_symbol_references` antes de mudar assinatura, e `resolveTarget` mudou de
   assinatura. As ferramentas `mcp__gopls__*` nao estao expostas neste agente —
   `ToolSearch` por "gopls" nao devolve nada. Substitui por `grep` sobre
   `internal/` (achou os tres chamadores e as duas mencoes em comentario) mais o
   compilador: em Go, chamador esquecido de uma assinatura mudada nao compila, e
   `go build ./...` faz parte do gate, que ficou verde. Registro porque e um
   desvio de processo, ainda que com garantia mais forte no lugar.

3. **Ledger.** Anexei a linha da Task 182 em
   `.superpowers/sdd/2026-09-06-broken-links/progress.md` **sem commita-la**: o
   arquivo e do orquestrador e ele escreve nele durante a tarefa; commitar por
   cima arriscaria conflito com uma linha que ele estivesse gravando. O commit
   leva so codigo, teste e o proprio relatorio nao vai nele.

4. **Escopo.** Nao toquei em `docs/` — dai o `[sem-doc]` no corpo da mensagem.
   `IndexCacheParserVersion = 2` e um fato que `docs/ESTADO.md` registra segundo
   o brief da Task 184, que e quem faz isso.
