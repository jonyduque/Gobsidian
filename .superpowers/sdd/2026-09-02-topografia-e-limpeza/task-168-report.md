# Task 168 — relatorio

## Progresso

- 02:59 iniciando: restaurando a versao preparada de `internal/mcpsrv/tools_read.go` (preparada
  antes da correcao da Task 166, guardada fora do repo enquanto a arvore foi isolada).
- 03:04 `go build ./...` limpo. `wc -l internal/mcpsrv/tools_read.go` = 402 (era 405 no HEAD antes
  desta task).
- 03:04 escrevendo `TestVaultSearchDefaultVemDoServico`, `TestNoteListDefaultVemDoServico`,
  `TestLinkGraphDefaultVemDoServico` em `tools_read_test.go`. Reusei o `connectTestSession`
  ja existente no arquivo (nao percebido de saida — criei uma segunda funcao com o mesmo nome e
  assinatura diferente, `go build` reprovou com "redeclared", corrigido chamando a versao
  existente `connectTestSession(ctx, t, srv)`).
- 03:05 os tres testes passam isolados (`go test -run ... -v`) e junto com o resto do pacote
  (`go test -race ./internal/mcpsrv/...`).
- 03:06 mutation proof: o comando EXATO do brief (`-Anchor "opts.Limit = 20"`) FALHOU —
  `[!] Ancora ocorre 2 vez(es)` — porque essa string e substring literal de `opts.Limit = 200`,
  duas linhas abaixo em `search.go`. Ampliei a ancora para o bloco `if opts.Limit <= 0 { ... }`
  inteiro (unico), mantendo a mesma mutacao semantica. Saida real colada abaixo.
- 03:08 rodando `verify.ps1`: reprovou em `gofmt` — `tools_read_test.go` (ver Desvios: editei o
  arquivo com um script Python que gravou CRLF em vez de LF, mudando TODO o arquivo aos olhos do
  gofmt). Corrigido convertendo de volta para LF.
- 03:12 `verify.ps1` de novo: `[OK] Bateria completa. Pode commitar.`
- 03:16 escrevendo este relatorio e preparando o commit.

## Status

Task 168 concluida. `tools_read.go` parou de reaplicar os defaults que o service ja aplica para
`vault_search`, `note_list` e `link_graph`; os tres campos booleanos que o service NAO tem como
default (por serem bool sem ponteiro no dominio) continuam desembrulhados no boundary, e isso
esta documentado inline em cada bloco.

## Passo 1 — tabela de defaults (mcpsrv vs service vs TOOLS.md)

| tool | param | mcpsrv (antes) | service | TOOLS.md | decisao |
|---|---|---|---|---|---|
| vault_search | snippet_chars | 240 | `SnippetChars<=0` -> `search.DefaultSnippetChars` (240) | 240 | **deletar** (os tres concordam) |
| vault_search | limit | 20 | `Limit<=0` -> 20, `>200` -> 200 | 20, max 200 | **deletar** |
| vault_search | offset | 0 | `Offset<0` -> 0 | 0 | **deletar** |
| note_list | limit | 100 | `ComTeto`: `LimitePadrao`=100, teto 500 | 100, max 500 | **deletar** |
| note_list | offset | 0 | (aceita cru, index.Query nao clampa negativo\*) | 0 | **deletar** — ver nota |
| note_list | tag_mode | "all" | `ValidarEnum` default "all" | "all" | **deletar** |
| note_list | sort | "path" | `ValidarEnum` default "path" | "path" | **deletar** |
| note_list | order | "asc" | `ValidarEnum` default "asc" | "asc" | **deletar** |
| note_list | recursive | true | SEM default no service (bool cru) | true | **manter** — service nao distingue "nao informado" de "false" sem ponteiro no dominio |
| link_graph | depth | 1 | `Depth<=0` -> 1, clamp 3 | 1, min 1 max 3 | **deletar** |
| link_graph | limit | 100 | `ComTeto` (mesma funcao de note_list) | 100, max 500 | **deletar** |
| link_graph | direction | "both" | `ValidarEnum` default "both" | "both" | **deletar** |
| link_graph | include_broken | true | SEM default (bool cru) | true | **manter** |
| link_graph | include_embeds | true | SEM default (bool cru) | true | **manter** |

\* `offset` de `note_list` viaja cru para `index.Query.Offset`; nao ha clamp de negativo
documentado no service para este campo especificamente, mas o valor "nao informado" (zero) ja
e o comportamento correto sem reaplicar nada em `tools_read.go` — o schema declara
`"default": 0`, que e o zero-value do proprio tipo Go, entao remover o `if in.Offset != nil`
nao muda nenhum comportamento observavel.

Nenhum default NOVO foi adicionado ao service — todos os campos deletados de `tools_read.go` ja
tinham a mesma conta no service ANTES desta task.

## Passo 3 — `valorOuZero`

```go
// valorOuZero desembrulha um parametro numerico opcional. Zero e "nao
// informado" para o service, que aplica o padrao -- a UNICA conta de cada
// padrao (Task 168). Duplicar o numero aqui, so para desembrulhar o ponteiro,
// era um segundo lugar para o padrao divergir do que o service aplica.
func valorOuZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
```

Usado nos 8 campos numericos deletados da tabela acima (`SnippetChars`, `Limit`, `Offset` em
vault_search; `Limit`, `Offset` em note_list; `Depth`, `Limit` em link_graph). Strings (`TagMode`,
`Sort`, `Order`, `Direction`) passam cruas — zero-value de `string` ja e `""`, que `ValidarEnum`
trata como "nao informado" sem precisar de ponteiro nem de helper.

## Evidencia — `wc -l`

```
tools_read.go antes (HEAD): 405
tools_read.go depois:       402
```

## Evidencia — os tres testes novos

```
$ go test ./internal/mcpsrv/ -run 'TestVaultSearchDefaultVemDoServico|TestNoteListDefaultVemDoServico|TestLinkGraphDefaultVemDoServico' -v
=== RUN   TestVaultSearchDefaultVemDoServico
--- PASS: TestVaultSearchDefaultVemDoServico (0.02s)
=== RUN   TestNoteListDefaultVemDoServico
--- PASS: TestNoteListDefaultVemDoServico (0.03s)
=== RUN   TestLinkGraphDefaultVemDoServico
--- PASS: TestLinkGraphDefaultVemDoServico (0.03s)
PASS
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	1.197s

$ go test -race ./internal/mcpsrv/...
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	7.011s
```

Cada teste chama a tool PELO TRANSPORTE MCP de verdade (`session.CallTool`, nao a funcao Go
direto), porque e so nesse caminho que um "reaplica aqui tambem" a mais voltaria a existir em
silencio:

- `TestVaultSearchDefaultVemDoServico`: busca sem `limit` nem `snippet_chars`; afirma
  `effective_limit == 20` e `effective_snippet_chars == 240` no `StructuredContent`.
- `TestNoteListDefaultVemDoServico`: tres notas (`z.md`, `a.md`, `m.md`) listadas sem `sort` nem
  `order`; afirma ordem `a.md, m.md, z.md` (path ascendente).
- `TestLinkGraphDefaultVemDoServico`: `A.md` linka `[[B]]`; consulta o grafo a partir de `B.md`
  SEM `direction`. B.md so tem aresta de ENTRADA — se o default nao caisse em "both", a
  travessia a partir de B.md nao encontraria A.md. Afirma que A.md aparece nos nodes.

## Evidencia — prova de mutacao

O comando exato do brief FALHOU por ancora ambigua (medido, nao assumido):

```
$ pwsh -File scripts/mutate.ps1 -Path internal/service/search.go -Anchor "opts.Limit = 20" -Replacement "opts.Limit = 21" -Test TestVaultSearchDefaultVemDoServico -Package ./internal/mcpsrv/
Carregado em 514ms
[!] Ancora ocorre 2 vez(es) em internal/service/search.go; precisa ocorrer exatamente 1.
    Amplie a ancora com as linhas vizinhas ate ela ficar unica.
EXIT: 2
```

Motivo: `"opts.Limit = 20"` e substring literal de `"opts.Limit = 200"` (a linha do teto,
duas linhas abaixo). Ancora ampliada para o bloco inteiro, unico no arquivo:

```
$ pwsh -File scripts/mutate.ps1 -Path internal/service/search.go \
    -Anchor $'if opts.Limit <= 0 {\n\t\topts.Limit = 20\n\t}' \
    -Replacement $'if opts.Limit <= 0 {\n\t\topts.Limit = 21\n\t}' \
    -Test TestVaultSearchDefaultVemDoServico -Package ./internal/mcpsrv/

[...] Mutando internal/service/search.go
      - if opts.Limit <= 0 {\n		opts.Limit = 20\n	}
      + if opts.Limit <= 0 {\n		opts.Limit = 21\n	}

[...] go test -race -run TestVaultSearchDefaultVemDoServico ./internal/mcpsrv/
----------------------------------------------------------------------
--- FAIL: TestVaultSearchDefaultVemDoServico (0.11s)
    tools_read_test.go:737: effective_limit = 21, queria 20 (padrao do service.Search, sem limit no pedido)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.943s
FAIL
----------------------------------------------------------------------
[OK] internal/service/search.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT CODE: 0
```

`git diff --stat -- internal/service/search.go` depois do restore: vazio (confirmado).

## `git diff --stat`

```
 internal/mcpsrv/tools_read.go      |  91 ++++++++++++-------------
 internal/mcpsrv/tools_read_test.go | 134 +++++++++++++++++++++++++++++++++++++
 2 files changed, 178 insertions(+), 47 deletions(-)
```

## `verify.ps1` — ultima linha

```
[OK] Bateria completa. Pode commitar.
```
14 etapas, todas `[OK]` na segunda rodada (mesmos 6 skips legitimos e ja conhecidos).

## Desvios do brief

1. **Comando de mutacao do brief nao funcionou como escrito.** `-Anchor "opts.Limit = 20"` e
   ambiguo em `search.go` (substring de `"opts.Limit = 200"`). Ampliei a ancora para o bloco `if`
   inteiro, preservando a mesma mutacao semantica ("o zero vira 20" -> "o zero vira 21"). Saida
   real de AMBAS as tentativas colada acima, sem editar a que falhou.
2. **`gofmt` reprovou apos a primeira edicao de `tools_read_test.go`.** Um script Python usado
   para renomear a chamada de `connectTestSession` gravou o arquivo inteiro com CRLF (o Python no
   Windows traduz `\n` para `\r\n` em modo texto por padrao), enquanto o arquivo original usava
   LF. Isso fez o `gofmt -d` mostrar TODO o arquivo como alterado — nao uma mudanca real de
   conteudo, so de terminador de linha. Corrigido convertendo de volta para LF em modo binario e
   reconferido com `gofmt -l` (vazio) antes do `verify.ps1` verde.
