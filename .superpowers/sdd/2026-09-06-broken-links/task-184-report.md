# Task 184 — Revisao de documentacao e preparo do release

## Progresso

```
22:07 - inicio - relatorio criado; lendo brief, CLAUDE.md, papeis/documentador.md, ARMADILHAS.md
22:07 - Step 1 (inventario) - grep rodado em docs/ e README.md; reviews 182/183/185 lidos; codigo conferido (write.go MoveNote, persist.go, service/*.go para o denominador de os-dois-indices)
22:10 - Step 2 (editar) - comecando as edicoes
22:14 - Step 2 (editar) - 9 arquivos editados; indo para os gates de doc
22:14 - Step 3 (gates) - rodando check_doc_refs.ps1
22:15 - Step 3 (gates) - check_doc_refs EXIT=0; check_readme_anchors EXIT=0; UTF-8 OK nos 10 .md
22:15 - Step 3 (gates) - rodando verify.ps1 -SkipCross -SkipNet (comando longo)
22:21 - Step 3 (gates) - verify.ps1 EXIT=0, 11 etapas verdes, 6 pulados (os de ambiente ja conhecidos)
22:21 - Step 4 (build) - build.ps1 EXIT=0, v1.4.1-111-gb34814a-dirty; version do binario colada
22:21 - Step 5 (commit) - 10 docs em stage por caminho explicito; git commit -F
22:22 - Step 5 (commit) - commit 399d0a6; hook pre_commit_docs nao bloqueou (nenhum .go de producao em stage)
22:22 - Step 4 (build, refeito) - rebuild pos-commit: v1.4.1-112-g399d0a6-dirty
22:22 - ledger - linha DONE escrita em progress.md
22:26 - Step 3 (extra) - audit_reports.ps1 184 rodado; achado NOVO no proprio instrumento (casa palavra, nao evidencia)
22:29 - Step 5 (commit 2) - ESTADO.md ganha a divida do audit_reports; gates de doc re-rodados EXIT=0; commit ed24393
22:29 - Step 4 (build, final) - v1.4.1-113-ged24393-dirty
```

## Status

**DONE.** Os oito achados carregados fechados, mais dois que apareceram na
conferencia contra o codigo. Nenhum BLOCKED. Nao criei tag: a tag e o push sao
do orquestrador.

## Commit

**Dois commits, nao um.** O segundo saiu de um achado que so apareceu quando
rodei o auditor sobre o meu proprio relatorio; a explicacao esta em "Auditoria
do proprio relatorio" e a divida esta em `ESTADO.md`.

1. `399d0a6769b61c6ad1b281fe3ef261e15e31a676` —
   `docs: anchor links, vault_broken_links, and the release workflow in the tree`
   (10 arquivos)
2. `ed24393` — `docs(estado): audit_reports matches the word, not the evidence`
   (so `docs/ESTADO.md`, +12 linhas)

**A tag do orquestrador vai em `ed24393`, nao em `399d0a6`.**

Trailers conferidos com `git log -1 --format='%(trailers)'`:

```
Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj
```

`git diff --cached --name-only` antes do commit — exatamente os dez arquivos que
editei, nenhum `git add -A`, nada sob `.superpowers/`, `test-vault/`, `.claude/`:

```
docs/ARCHITECTURE.md
docs/ARMADILHAS.md
docs/ESTADO.md
docs/ESTRUTURA.md
docs/TOOLS.md
docs/wiki/concepts/os-dois-indices.md
docs/wiki/entities/note-e-caminho.md
docs/wiki/features/escrita.md
docs/wiki/features/parser.md
docs/wiki/risks/armadilhas-pagas.md
```

## Files changed

```
 docs/ARCHITECTURE.md                  |  6 +++
 docs/ARMADILHAS.md                    | 24 ++++++++++++
 docs/ESTADO.md                        | 74 ++++++++++++++++++++++++++++++++++-
 docs/ESTRUTURA.md                     |  3 +-
 docs/TOOLS.md                         | 13 ++++--
 docs/wiki/concepts/os-dois-indices.md | 13 ++++--
 docs/wiki/entities/note-e-caminho.md  | 13 +++++-
 docs/wiki/features/escrita.md         | 14 ++++++-
 docs/wiki/features/parser.md          | 16 ++++++--
 docs/wiki/risks/armadilhas-pagas.md   | 16 +++++++-
 10 files changed, 177 insertions(+), 15 deletions(-)
```

## Step 1 — Inventario

Comando (excluindo `docs/superpowers/`, que e plano e nao normativa):

```bash
grep -rn "broken_links\|anchor_missing\|Anchor\|ancora\|âncora" docs/ README.md
```

O grep devolveu 57 ocorrencias. A tabela abaixo traz **todas**; "certo" quer
dizer que li a linha e ela continua verdadeira depois de `d91b2fb..b34814a`.

| Local | Situacao | Nota |
|---|---|---|
| `docs/TOOLS.md:13` (Limites) | **mudado** | faltava `vault_broken_links` na classe padrao 100 / teto 500 — achado 2 da review-183 |
| `docs/TOOLS.md:245-256` (estados de link) | certo | `ok` / `target_missing` / `anchor_missing` seguem exatos |
| `docs/TOOLS.md:287` (`link_graph`, `source == target`) | certo | escrito pela Task 183; conferido contra `graph.go` |
| `docs/TOOLS.md:293-323` (`vault_broken_links`) | **mudado** | o retorno nao dizia que `target` vem vazio na auto-ancora quebrada — nit 4 da review-183. Acrescentei o paragrafo e uma terceira linha ao exemplo (`total` foi de 2 para 3) |
| `docs/TOOLS.md:319` ("O que NAO entra") | certo | ja cita a auto-ancora e a Task 182 |
| `docs/TOOLS.md:389` (`vault_stats` health) | certo | nomes dos campos batem |
| `docs/TOOLS.md:496` (`note_move`, retorno) | **mudado** | "A origem nao entra: mover nao altera o conteudo dela" era falso — achado 1 da review-185 |
| `docs/TOOLS.md:498, 523` (preservacao de forma; `note_delete`) | certo | — |
| `docs/TOOLS.md:177` (`note_outline`, `#` de ATX) | certo | nao e sobre ancora de link |
| `docs/ARCHITECTURE.md:191-195` (§3.6 Ancoras) | **mudado** | dizia so "ancora que nao existe"; nao dizia que a separacao vale para os tres `LinkKind`, nem que alvo vazio resolve para a origem |
| `docs/ARCHITECTURE.md:241,256,262` (structs) | certo | `Anchor` e `LinkState` conferem com `internal/index` |
| `docs/ARCHITECTURE.md:455-466` (§5.6 `note_move`) | **mudado** | a sequencia estava certa (corpo antes dos citantes) e nao dizia o que isso cobra de quem cita a si mesma |
| `docs/ARCHITECTURE.md:463` ("preservar alias, ancora...") | certo | — |
| `docs/ARCHITECTURE.md:759` (paridade assimetrica) | certo | e o que a review-182 F4 confirma |
| `docs/ARMADILHAS.md` (secao "Acesso a arquivo e confinamento") | **mudado** | entrada nova: lista montada antes do rename, lida depois |
| `docs/ARMADILHAS.md:578` (assert de texto-ancora em script) | certo | outro assunto (edicao de `.md` por script) |
| `docs/ESTADO.md` (Marcos) | **mudado** | marco novo Tasks 182-185 |
| `docs/ESTADO.md` (Formato de cache) | **mudado** | dizia "o formato do cache de metadados e o 3"; `persist.go:38` diz **5**. Ver Concern 1 |
| `docs/ESTADO.md` (Dividas abertas) | **mudado** | duas dividas novas: hook `pre_commit_docs.ps1`; `[x](b.md#)` parqueado |
| `docs/ESTRUTURA.md:75,88` (`slug.go`, `anchors.go`) | certo | os dois arquivos existem e as descricoes batem |
| `docs/ESTRUTURA.md:130` (`broken.go`) | certo | a Task 183 ja o pos na arvore |
| `docs/ESTRUTURA.md:225-227` (`.github/workflows/`) | **mudado** | `release.yml` estava ausente |
| `docs/ESTRUTURA.md` (arquivos `_test.go`) | certo, sem mudanca | a arvore **nao** lista teste por arquivo (unica mencao a `_test.go` e a linha 56, sobre `vaulttest`), entao `anchor_selfref_test.go` nao entra — convencao do proprio arquivo |
| `docs/OPERACAO.md:64,89` (`broken_links`, `broken_anchors`) | certo | as duas descricoes seguem exatas depois da Task 182 |
| `docs/OPERACAO.md:1314, 2173-2176, 2505, 2553` | certo | medicao descartada, ruling M1 e tabela de heap; nada mudou |
| `docs/OPERACAO.md` (tools / versao de cache) | **nada a mudar** | ver Concern 4 para o grep que sustenta |
| `docs/PRD.md:120,168,208,246,468,491` (RF-11, RF-63, D12) | certo | ver Concern 2 |
| `docs/papeis/documentador.md:64,92` | certo | ancora de README e de script |
| `docs/papeis/testador.md:61,74-75` | certo | `-Anchor` do `mutate.ps1` |
| `docs/REVISAO-2026-08-15.md` (9 ocorrencias) | certo | documento historico, congelado por definicao |
| `docs/SUGESTOES.md` (13 ocorrencias) | certo | idem; M1 segue REJEITADO e M16 segue aberto |
| `docs/wiki/concepts/os-dois-indices.md:32-33` | **mudado** | "11 das 14" com a clausula "so `vault_search` precisa do outro" — achado 1 da review-183 |
| `docs/wiki/entities/note-e-caminho.md:137` | **mudado** | a tabela de estados nao dizia que alvo vazio com ancora aponta para a propria nota |
| `docs/wiki/features/parser.md:87,98` | **mudado** | a linha da extensao e a de `Slug` tratavam ancora como coisa de wikilink |
| `docs/wiki/features/escrita.md` (§`note_move`) | **mudado** | nao dizia a ordem corpo-antes-dos-citantes nem o caso da auto-citacao |
| `docs/wiki/features/tools-mcp.md:34,64` | certo | `vault_broken_links` ja esta na tabela com "Toca o disco? nao"; a linha 64 fala de `note_outline` |
| `docs/wiki/features/busca.md:80` | certo | "ancora" ali e a janela do trecho, outro sentido |
| `docs/wiki/reference/gates-e-scripts.md:54,57,74` | certo | ancora do `mutate.ps1` e do README |
| `docs/wiki/notes/incidente-de-campo-2026-08-15.md:50,93` | certo | historico |
| `docs/wiki/risks/armadilhas-pagas.md:148` | **mudado** (secao nova) | a pagina espelha `ARMADILHAS.md` por assunto; ganhou "Lista montada antes do rename, lida depois". Ver Concern 3 |
| `docs/wiki/_wiki/schema.md:99` | certo | "ancorada" no sentido de tipo de pagina |
| `docs/wiki/Home.md:87,127`, `_Sidebar.md:10` | certo | "14 tools" bate com 14 `mcp.AddTool` de producao |
| `README.md:51,237,239` | certo | a Task 183 ja registrou `vault_broken_links` |

### Como cheguei ao 13/14 de `os-dois-indices.md`

Nao contei de memoria. `s.inverted` — o unico ponteiro para o indice invertido
dentro de `internal/service` — aparece em **um** arquivo de producao:

```
$ grep -rn "s\.inv\|\.inv\b" internal/service/*.go | grep -v _test.go
internal/service/search.go:171:  if s.inverted != nil && s.inverted.Building() {
internal/service/search.go:226:  rawHits := search.CalculateBM25(queryTokens, s.inverted, s.index)
internal/service/search.go:311:  snip, errTrecho := search.GenerateSnippet(ctx, s.vault, s.inverted, s.index, ...)
internal/service/search.go:499:  if s.inverted == nil || len(phraseTokens) <= 1 {
internal/service/search.go:505:  positions := s.inverted.Positions(tok.Raw, path)
```

`search.go` serve `vault_search` e mais nada (`grep -n "^func (s \*Service)"`
devolve `Search`, `searchMetadataOnly`, `matchesSearchFilters`,
`casamFrontmatter`, `matchPhraseInNote` — todos internos a ela). O campo so
aparece fora dali em `service.go:60,79`, que sao a declaracao e a injecao. Logo
**13 das 14 tools nao precisam do indice invertido**, e a frase antiga nao
fechava com a propria clausula seguinte — 14 menos uma e 13, nunca 11. Escrevi o
derivado e o metodo junto, para o proximo nao ter de refazer a conta.

### O que verifiquei no codigo antes de escrever cada afirmacao

- `internal/index/resolve.go:114-131` — `resolveTarget` devolve
  `(origin, ViaPath, LinkOK)` para `target == "" && anchor != ""`, **sem olhar o
  `Kind`**. E o que sustenta a frase "vale para as tres formas" em
  `ARCHITECTURE.md` e no wiki; a review-182 F7 tinha notado que o comentario
  afirmava as tres e o teste cobria duas, e o caminho de codigo e de fato um so.
- `internal/parser/ast.go:36,53` e `ext_wikilink.go:125` — `splitAnchor` e
  chamada dos dois ramos Markdown de `collect` (`*gast.Link` e `*gast.Image`) e
  de `splitWikilink`, e nos ramos Markdown **antes** de `PercentDecode`
  (`ast.go:39-40,56-57`). E o que sustenta a frase sobre a ordem.
- `internal/service/write.go:493-535` — o guarda `rl.Target == ""` tira a
  auto-referencia so de ancora de `affectedNotes`.
- `internal/service/write.go:544-570` — no `dry_run`, `diffs` e chaveado por
  `refPath`, que para a origem e o caminho **antigo**.
- `internal/service/write.go:655-681` — na execucao real, `caminhoAtual` manda a
  origem para `canonicalTo`, e e `caminhoAtual` que entra em `rewrittenList`.
  Por isso a frase nova de `TOOLS.md` distingue os dois caminhos: `dry_run`
  reporta o antigo, a execucao reporta o novo. Nao e detalhe: quem le `diffs`
  para prever o resultado veria uma chave que a execucao nao repete.
- `internal/index/persist.go:38,49` — `IndexCacheFormatVersion = 5`,
  `IndexCacheParserVersion = 2`.

## Gate outputs

### `check_doc_refs.ps1`

```
[i] corpus: 383 arquivos .go.
[i] 37 dispensa(s) em uso -- nao contam como achado:
  (as 37 sao as pre-existentes; nao acrescentei dispensa nenhuma)
[OK] nenhum token entre crases parece citar artefato ausente do codigo.
EXIT=0
```

### `check_readme_anchors.ps1`

```
[i] 11 heading(s), 11 link(s) interno(s).
[OK] toda ancora resolve e toda secao H2 e alcancavel pela navegacao.
EXIT=0
```

### UTF-8, um por arquivo editado

```
$ for f in <os dez>; do python -c "open('$f',encoding='utf-8').read()" && echo "[OK] UTF-8 valido: $f"; done
[OK] UTF-8 valido: docs/TOOLS.md
[OK] UTF-8 valido: docs/ARCHITECTURE.md
[OK] UTF-8 valido: docs/ARMADILHAS.md
[OK] UTF-8 valido: docs/ESTADO.md
[OK] UTF-8 valido: docs/ESTRUTURA.md
[OK] UTF-8 valido: docs/wiki/concepts/os-dois-indices.md
[OK] UTF-8 valido: docs/wiki/entities/note-e-caminho.md
[OK] UTF-8 valido: docs/wiki/features/parser.md
[OK] UTF-8 valido: docs/wiki/features/escrita.md
[OK] UTF-8 valido: docs/wiki/risks/armadilhas-pagas.md
```

### `verify.ps1 -SkipCross -SkipNet`

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
     --- SKIP: TestWriteAtomicPreservaOModoDoAlvo (0.00s)
     --- SKIP: TestNew_FailsOnUnwatchablePath (0.00s)
[...] 4. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 5. go vet (windows)
[OK] go vet (windows)
[i] vet cruzado pulado (-SkipCross)
[...] 6. gofmt
[OK] gofmt
[...] 7. golangci-lint
[OK] golangci-lint
[...] 8. golangci-lint (linux)
[OK] golangci-lint (linux)
[i] check_net pulado (-SkipNet)
[...] 9. check_tool_params
[OK] check_tool_params
[...] 10. check_doc_refs
[OK] check_doc_refs
[...] 11. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
EXIT=0
```

Os 6 pulados sao os mesmos que a review-185 nomeou um a um como de ambiente do
Windows; nenhum novo.

## Build output

Antes do commit, sobre `b34814a`:

```
[...] Compilando v1.4.1-111-gb34814a-dirty (b34814a)
[OK] C:\Users\jonyd\Projetos\Gobsidian\bin\gobsidian.exe (10.82 MB)
EXIT=0

$ .\bin\gobsidian.exe version
gobsidian v1.4.1-111-gb34814a-dirty (b34814a) 2026-09-07T01:21:23Z
```

Refeito depois do commit, para o binario carregar o commit de documentacao:

```
[...] Compilando v1.4.1-112-g399d0a6-dirty (399d0a6)
[OK] C:\Users\jonyd\Projetos\Gobsidian\bin\gobsidian.exe (10.82 MB)
EXIT=0

$ .\bin\gobsidian.exe version
gobsidian v1.4.1-112-g399d0a6-dirty (399d0a6) 2026-09-07T01:22:17Z
```

Refeito uma terceira vez depois do commit 2, que e o build que o release deve
usar:

```
[...] Compilando v1.4.1-113-ged24393-dirty (ed24393)
[OK] C:\Users\jonyd\Projetos\Gobsidian\bin\gobsidian.exe (10.82 MB)

$ .\bin\gobsidian.exe version
gobsidian v1.4.1-113-ged24393-dirty (ed24393) 2026-09-07T01:29:45Z
```

`v1.4.1-113-ged24393` e o esperado antes da tag. O sufixo `-dirty` **nao vem de
nada meu**: `git status --porcelain` depois do commit lista so trabalho nao
commitado do dono (`test-vault/`, `.claude/skills/troglodita*`,
`Resume-Claude.ps1`), o ledger e o plano nao versionado. Depois de `git tag
v1.5.0` o binario lera `v1.5.0-dirty` enquanto essa arvore continuar assim — se
o release precisa de versao limpa, o build tem de sair de uma arvore limpa (CI,
que e o que `release.yml` faz, ou `git stash` do dono, que **eu nao rodo**).

## Concerns

1. **`docs/ESTADO.md` afirmava a versao errada do cache de metadados, e eu
   corrigi.** Dizia "o formato do cache de metadados e o 3 (2 ate 2026-08-26)";
   `internal/index/persist.go:38` diz `IndexCacheFormatVersion = 5`. Os bumps
   3→4→5 estao documentados no proprio `persist.go` (`contextoBytes` 80 → 40 →
   80) e nunca chegaram ao `ESTADO.md`. Fora do brief, mas e a classe de defeito
   que o projeto trata como cara — numero errado em documento que se consulta
   para decidir. A tabela de medicao logo abaixo compara formato 2 × 3 e
   continua correta como medicao daquela virada; nao inventei numero para 4 nem
   para 5. **Nao ha medicao de 3→5** e o texto nao finge que ha.

2. **`docs/PRD.md` RF-63 ilustra a validacao de ancora so com wikilink**
   (`[[nota#heading]]`, `[[nota#^bloco]]`) e nao mencionei nada la. Nao esta
   *errado* — o requisito continua sendo o que o codigo faz, e agora vale para
   mais formas do que os exemplos citam. Mas o PRD e normativo e o exemplo agora
   e estreito. **Nao editei** porque mudar redacao de requisito e decisao do
   dono, nao de quem revisa documentacao; deixo o ponteiro. Vale uma linha de
   uma tarefa futura ou um ruling.

3. **Cresci o escopo em um arquivo alem do brief:**
   `docs/wiki/risks/armadilhas-pagas.md`. Ela e a pagina derivada que espelha
   `ARMADILHAS.md` por assunto, e deixa-la sem a armadilha nova produziria
   exatamente a divergencia que `papeis/documentador.md` proibe. A secao nova e
   curta e termina apontando para `docs/ARMADILHAS.md` como a conta completa, em
   vez de recopiar. Se o orquestrador achar que a pagina nao devia crescer, e um
   `git revert` de um hunk.

4. **`docs/OPERACAO.md`: nada a mudar**, e o grep que sustenta:

   ```
   $ grep -nE "IndexCacheParserVersion|IndexCacheFormatVersion|CacheParserVersion|1[0-4] tools|tools MCP|lista de tools|vault_broken_links" docs/OPERACAO.md
   EXIT=1     (zero linhas)
   ```

   Ele nao inventaria tools nem nomeia as constantes de versao de cache. As
   quatro ocorrencias de `broken_links`/`broken_anchors`/ancora que o inventario
   lista (linhas 64, 89, 2173-2176, 2505) foram lidas uma a uma e continuam
   verdadeiras: a descricao de `broken_links` ("links que deveriam resolver para
   uma nota do cofre e nao resolvem") e a de `broken_anchors` ("links que
   resolvem a nota mas nao ao heading ou bloco citado") descrevem exatamente o
   que o codigo conta depois da Task 182.

5. **`docs/superpowers/plans/2026-09-06-broken-links.md` continua UNTRACKED.** O
   plano deste marco nao esta versionado, e eu nao o commitei porque o brief
   limita o commit aos documentos que editei. Se o release deve sair com o plano
   na arvore, e uma decisao do orquestrador — e um `git add` de um caminho.

6. **Um comentario de codigo ficou desatualizado, e nao o toquei.**
   `internal/service/write.go:545` diz "A origem nao entra em diffs: mover nao
   altera o conteudo dela" — e a mesma frase que eu acabei de corrigir no
   `TOOLS.md`, e no `dry_run` ela e falsa pelo mesmo motivo (o laco itera
   `affectedNotes`, que contem `canonicalFrom` quando a nota cita a si mesma com
   alvo escrito). A review-185 achado 1 ja registra a metade do `dry_run` como
   **pre-existente**. Nao mexi porque esta task e de documentacao e mudar
   comentario de `.go` obrigaria `verify.ps1` completo por um texto — mas o
   comentario e agora a unica copia da afirmacao errada que sobrou no repo.
   Merece uma linha numa task de codigo.

7. **Nao re-medi cofre real, e o `ESTADO.md` diz isso.** Os unicos numeros que
   escrevi sao os do paragrafo Spec do plano (quatro cofres, zero alvos
   quebrados com URL, 267 + 10 vazios no Estudo, 372 no Oral) e os dois valores
   de constante lidos de `persist.go`. Que a contagem de `broken_links` dos
   cofres reais tenha caido depois da Task 182 esta escrito como **"nao
   re-medido"**, que e o que e.

## Verificacao

Esta task nao escreve codigo e nao tem teste, entao nao ha ciclo RED/GREEN nem
prova de mutacao para colar — **e essa e a resposta, nao uma omissao**. O que
verifica uma mudanca de documentacao aqui sao tres coisas, todas com saida
colada acima:

1. **Os gates de doc** — `check_doc_refs.ps1` (todo token entre crases que
   parece codigo tem de existir; EXIT=0, e **nao acrescentei dispensa nenhuma**,
   as 37 impressas sao as pre-existentes) e `check_readme_anchors.ps1` (EXIT=0).
2. **`verify.ps1 -SkipCross -SkipNet`** — 11 etapas, EXIT=0. Mudanca so de
   documentacao nao pode quebrar build nem teste, e o gate confirma que nao
   quebrou.
3. **Cada afirmacao nova conferida contra o codigo**, com o sitio nomeado — a
   lista esta em "O que verifiquei no codigo antes de escrever cada afirmacao".
   E o substituto honesto do teste: nao afirmei estado que nao li.

O que **nao** verifiquei, dito sem rodeio: nao rodei o servidor contra cofre
real, nao re-medi contagem de `broken_links` em cofre nenhum, e nao rodei
`test_orphans.ps1` (proibido pelo brief).

## Auditoria do proprio relatorio

`pwsh -File scripts/audit_reports.ps1 184`, saida integral:

```
=== Relatorios (1) ===
  .superpowers\sdd\2026-09-06-broken-links\task-184-report.md:1: [SECAO-AUSENTE] sem secao de TDD/RED
  .superpowers\sdd\2026-09-06-broken-links\task-184-report.md:1: [SECAO-AUSENTE] sem secao de TDD/GREEN
  .superpowers\sdd\2026-09-06-broken-links\task-184-report.md:1: [SECAO-AUSENTE] sem secao de Verificacao

=== Ledger ===
  (14 achados no ledger antigo 2026-07-25-gobsidian-v01: 2 SHA-NAO-CONFERE nas
   Tasks 4 e 6, 9 RELATORIO-AUSENTE nas Tasks 94-103, 1 SHA-FANTASMA "deadbee",
   1 SHA-NAO-CONFERE na Task 153 — todos pre-existentes e conhecidos, nenhum
   desta task)

[!] 17 achado(s). Nenhum e automaticamente um defeito -- cada um e
    uma frase ou um SHA que precisa de uma pessoa confirmando.
[i] Onde nao houve medicao, 'nao medido' e a resposta certa e nao e sinalizada.
EXIT=1
```

**Os tres achados sobre este relatorio, confirmados por mim:** `TDD/RED` e
`TDD/GREEN` sao verdadeiros e **continuam verdadeiros** — nao ha teste nesta
task, que edita dez `.md` e nao toca uma linha de Go. Inventar um par RED/GREEN
para satisfazer o checador seria a evidencia falsa que ele existe para pegar.

**E aqui esta um achado do proprio instrumento, que eu preferia nao ter
produzido.** Rodei o auditor de novo depois de escrever a secao "Verificacao"
acima, e a secao `=== Relatorios (1) ===` voltou **vazia**: os tres achados
sumiram. Nao porque eu tenha colado RED, GREEN ou mutacao nenhuma — sumiram
porque `scripts/audit_reports.ps1:113-118` procura as quatro secoes por
**palavra solta no corpo inteiro**:

```powershell
$Required = @(
    @{ Name = 'TDD/RED';     Pattern = '(?i)(^|\W)red(\W|$)' },
    @{ Name = 'TDD/GREEN';   Pattern = '(?i)(^|\W)green(\W|$)' },
    @{ Name = 'Mutacao';     Pattern = '(?i)muta' },
    @{ Name = 'Verificacao'; Pattern = '(?i)verifica' }
)
```

A frase "**nao** ha ciclo RED/GREEN nem prova de mutacao para colar" contem
`RED`, `GREEN` e `muta`, e satisfaz as tres. Uma negacao explicita passa pelo
mesmo portao que uma evidencia real. **E a mesma classe do achado F5 da
review-182** — o hook `pre_commit_docs.ps1` que grepa `[sem-doc]` do texto do
comando: um checador que casa a palavra em vez do fato nao gateia nada.

Registro isto porque a saida vazia acima, sozinha, seria uma alegacao falsa
sobre este relatorio: **nao ha RED, nao ha GREEN e nao ha prova de mutacao nesta
task**, e o auditor deixou de dizer isso por causa da minha prosa, nao por causa
do meu trabalho. A saida colada e a da **primeira** rodada, que e a informativa.

Os 14 do ledger sao do plano antigo (`2026-07-25-gobsidian-v01`), pre-existentes,
e o brief ja os declara conhecidos. Nenhum e desta task; o ledger deste marco
(`2026-09-06-broken-links/progress.md`) nao produziu achado nenhum.
