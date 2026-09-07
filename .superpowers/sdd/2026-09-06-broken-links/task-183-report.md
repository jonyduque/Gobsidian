# Task 183 — `vault_broken_links`

## Progresso

21:15 - inicio - brief lido, HEAD c987ff1
21:17 - step1-2 - broken_test.go escrito, RED de compilacao confirmado
21:19 - step3-5 - broken.go + registro em mcpsrv; service e mcpsrv verdes
21:22 - step6 - docs (TOOLS/README/wiki/ESTRUTURA) + check_tool_params, check_readme_anchors, check_doc_refs verdes
21:22 - step7 - iniciando as quatro provas de mutacao
21:23 - step7 - cinco provas de mutacao, todas REPROVAM sob mutacao
21:23 - step8 - rodando scripts/verify.ps1 completo (pode demorar)
21:29 - step8 - verify.ps1 verde nas 14 etapas
21:30 - fim - commit 5f9ed57
21:34 - fim - audit_reports.ps1 183: zero achados contra este relatorio

## Status

**DONE_WITH_CONCERNS** — o codigo, os testes e os gates estao todos verdes; as
ressalvas sao decisoes de escopo tomadas por mim que o revisor precisa aprovar
ou reverter, listadas ao fim.

## Commits

`5f9ed57` — `feat(tools): vault_broken_links lists every missing target and anchor in the vault`

13 arquivos, +424 -15. Nada de `test-vault/`, `.claude/`, `Resume-Claude.ps1`
nem `.superpowers/` entrou: o `git add` foi por caminho explicito e o
`git diff --cached --name-only` foi conferido antes do commit.

## Files changed

- `internal/service/broken.go` (novo) — `BrokenLinksRequest`, `BrokenLink`,
  `BrokenLinksResult`, `achadoQuebrado` (privado) e `(*Service).BrokenLinks`
- `internal/service/broken_test.go` (novo)
- `internal/mcpsrv/tools_read.go` — registro da tool `vault_broken_links` e
  `vaultBrokenLinksInput`
- `internal/mcpsrv/tools_read_test.go` — `vault_broken_links` na lista de tools
  esperadas, `[[B#Nada]]` na fixture e o caso de chamada com
  `state: "anchor_missing"`
- `docs/TOOLS.md` — secao `vault_broken_links`; e a frase estacionada da revisao
  da 182 na secao `link_graph` (aresta com `source == target`)
- `README.md` — uma linha na tabela de MCP tools
- `docs/ESTRUTURA.md` — `broken.go` na arvore de `internal/service`
- `docs/wiki/features/tools-mcp.md` — linha na tabela, `## Leitura (9)`,
  "Seis das nove", frontmatter (`description`, `updated_at`, `source_commit`)
- `docs/wiki/Home.md`, `docs/wiki/_Sidebar.md`, `docs/wiki/features/busca.md`,
  `docs/wiki/concepts/camadas-e-fronteiras.md`,
  `docs/wiki/concepts/os-dois-indices.md` — "As 13 tools" -> "As 14 tools"

## Test output

RED antes (Step 2):

```
# github.com/jonyd/gobsidian/internal/service [github.com/jonyd/gobsidian/internal/service.test]
internal\service\broken_test.go:42:19: svc.BrokenLinks undefined (type *Service has no field or method BrokenLinks)
internal\service\broken_test.go:42:36: undefined: BrokenLinksRequest
...
FAIL	github.com/jonyd/gobsidian/internal/service [build failed]
FAIL
```

GREEN depois (Step 4), `go test ./internal/service -run TestBrokenLinks -v -race`:

```
=== RUN   TestBrokenLinksListaOQueVaultStatsSoConta
=== RUN   TestBrokenLinksListaOQueVaultStatsSoConta/sem_filtro
=== RUN   TestBrokenLinksListaOQueVaultStatsSoConta/filtro_por_estado
=== RUN   TestBrokenLinksListaOQueVaultStatsSoConta/filtro_por_prefixo
=== RUN   TestBrokenLinksListaOQueVaultStatsSoConta/prefixo_que_nao_casa_nada_nao_e_erro
=== RUN   TestBrokenLinksListaOQueVaultStatsSoConta/paginacao
=== RUN   TestBrokenLinksListaOQueVaultStatsSoConta/offset_alem_do_fim_devolve_pagina_vazia
=== RUN   TestBrokenLinksListaOQueVaultStatsSoConta/estado_invalido_nao_vira_silencio
--- PASS: TestBrokenLinksListaOQueVaultStatsSoConta (0.02s)
    --- PASS: TestBrokenLinksListaOQueVaultStatsSoConta/sem_filtro (0.00s)
    --- PASS: TestBrokenLinksListaOQueVaultStatsSoConta/filtro_por_estado (0.00s)
    --- PASS: TestBrokenLinksListaOQueVaultStatsSoConta/filtro_por_prefixo (0.00s)
    --- PASS: TestBrokenLinksListaOQueVaultStatsSoConta/prefixo_que_nao_casa_nada_nao_e_erro (0.00s)
    --- PASS: TestBrokenLinksListaOQueVaultStatsSoConta/paginacao (0.00s)
    --- PASS: TestBrokenLinksListaOQueVaultStatsSoConta/offset_alem_do_fim_devolve_pagina_vazia (0.00s)
    --- PASS: TestBrokenLinksListaOQueVaultStatsSoConta/estado_invalido_nao_vira_silencio (0.00s)
=== RUN   TestBrokenLinksAplicaOTetoDeLimit
--- PASS: TestBrokenLinksAplicaOTetoDeLimit (0.03s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	1.987s
```

`mcpsrv` (Step 5), `go test ./internal/mcpsrv/ -run TestReadTools -v`:

```
=== RUN   TestReadTools/vault_broken_links_valid
...
    --- PASS: TestReadTools/Tools_registered (0.00s)
    --- PASS: TestReadTools/link_graph_valid (0.00s)
    --- PASS: TestReadTools/vault_broken_links_valid (0.00s)
    --- PASS: TestReadTools/tag_list_valid (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	0.945s
```

Depois do commit, os dois pacotes com `-race`:

```
ok  	github.com/jonyd/gobsidian/internal/service	2.106s
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	2.628s
```

## Mutation proofs

Todas rodadas com `scripts/mutate.ps1`, que restaura o arquivo byte a byte e
confere o SHA-256. As cinco ja aconteceram; a saida abaixo e a real.

### 1. Deixar `LinkExternal` passar

```
pwsh -File scripts/mutate.ps1 -Path internal/service/broken.go `
  -Anchor 'if l.State != index.LinkTargetMissing && l.State != index.LinkAnchorMissing {' `
  -Replacement 'if l.State == index.LinkOK {' `
  -Test TestBrokenLinksListaOQueVaultStatsSoConta -Package ./internal/service/
```

```
[...] Mutando internal/service/broken.go
      - if l.State != index.LinkTargetMissing && l.State != index.LinkAnchorMissing {
      + if l.State == index.LinkOK {

--- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta (0.02s)
    --- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta/sem_filtro (0.00s)
        broken_test.go:47: Total = 4, queria 3: [{Source:a.md Target:nada ... State:target_missing ...} {Source:a.md Target:b Anchor:Nada ... State:anchor_missing ...} {Source:a.md Target:https://ex.com Anchor: Alias:s Kind:markdown State:external ...} {Source:sub/c.md Target:outra ... State:target_missing ...}]
    --- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta/paginacao (0.00s)
        broken_test.go:125: Total = 4, queria 3: o total nao pode encolher com a pagina
    --- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta/offset_alem_do_fim_devolve_pagina_vazia (0.00s)
        broken_test.go:141: Total = 4, len = 0, queria 3 e 0
FAIL	github.com/jonyd/gobsidian/internal/service	0.920s

[OK] internal/service/broken.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

`Total = 4` e exatamente o `https://ex.com` entrando: e o numero que o brief
previu. A saida tambem mostra o que NAO entrou nem sob mutacao — `[x](#Topo)`,
a auto-ancora que resolve.

### 2. Remover o `ComTeto`

```
pwsh -File scripts/mutate.ps1 -Path internal/service/broken.go `
  -Anchor 'limit := ComTeto(req.Limit)' -Replacement 'limit := req.Limit' `
  -Test TestBrokenLinksAplicaOTetoDeLimit -Package ./internal/service/
```

```
[...] Mutando internal/service/broken.go
      - limit := ComTeto(req.Limit)
      + limit := req.Limit

--- FAIL: TestBrokenLinksAplicaOTetoDeLimit (0.03s)
    broken_test.go:180: len(Links) = 501, queria 500: o limite absurdo chegou ao corte como veio
FAIL	github.com/jonyd/gobsidian/internal/service	0.821s

[OK] internal/service/broken.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### 3. Trocar a ordenacao

```
pwsh -File scripts/mutate.ps1 -Path internal/service/broken.go `
  -Anchor 'return cmp.Compare(a.start, b.start)' `
  -Replacement 'return cmp.Compare(b.start, a.start)' `
  -Test TestBrokenLinksListaOQueVaultStatsSoConta -Package ./internal/service/
```

```
[...] Mutando internal/service/broken.go
      - return cmp.Compare(a.start, b.start)
      + return cmp.Compare(b.start, a.start)

--- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta (0.02s)
    --- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta/sem_filtro (0.00s)
        broken_test.go:61: Links[0] = {a.md b anchor_missing}, queria {a.md nada target_missing}
        broken_test.go:61: Links[1] = {a.md nada target_missing}, queria {a.md b anchor_missing}
    --- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta/paginacao (0.00s)
        broken_test.go:131: Links[0] = {Source:a.md Target:nada ... State:target_missing ...}, queria o SEGUNDO item da ordem (b, anchor_missing)
FAIL	github.com/jonyd/gobsidian/internal/service	0.826s

[OK] internal/service/broken.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### 4. Remover o `ValidarEnum`

```
pwsh -File scripts/mutate.ps1 -Path internal/service/broken.go `
  -Anchor 'estado, err := ValidarEnum("state", req.State, "", "target_missing", "anchor_missing")' `
  -Replacement 'estado, err := req.State, error(nil)' `
  -Test TestBrokenLinksListaOQueVaultStatsSoConta -Package ./internal/service/
```

```
[...] Mutando internal/service/broken.go
      - estado, err := ValidarEnum("state", req.State, "", "target_missing", "anchor_missing")
      + estado, err := req.State, error(nil)

--- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta (0.01s)
    --- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta/estado_invalido_nao_vira_silencio (0.00s)
        broken_test.go:148: state invalido foi ACEITO e caiu no padrao em silencio
FAIL	github.com/jonyd/gobsidian/internal/service	0.809s

[OK] internal/service/broken.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### 5. Desligar o filtro de prefixo (extra, alem das quatro do brief)

```
pwsh -File scripts/mutate.ps1 -Path internal/service/broken.go `
  -Anchor 'if prefixo != "" && !strings.HasPrefix(text.ChaveDeCaminho(string(p)), prefixo) {' `
  -Replacement 'if false && !strings.HasPrefix(text.ChaveDeCaminho(string(p)), prefixo) {' `
  -Test TestBrokenLinksListaOQueVaultStatsSoConta -Package ./internal/service/
```

```
      + if false && !strings.HasPrefix(text.ChaveDeCaminho(string(p)), prefixo) {

--- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta (0.01s)
    --- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta/filtro_por_prefixo (0.00s)
        broken_test.go:101: Total = 3, len = 3, queria 1 e 1: [...]
    --- FAIL: TestBrokenLinksListaOQueVaultStatsSoConta/prefixo_que_nao_casa_nada_nao_e_erro (0.00s)
        broken_test.go:114: Total = 3, len = 3, queria 0 e 0
FAIL	github.com/jonyd/gobsidian/internal/service	0.841s

[OK] internal/service/broken.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

## Gate outputs

`pwsh -File scripts/verify.ps1`, completo, em foreground:

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
Carregamento: 1818,9 ms

[OK] check_tool_params
[...] 13. check_doc_refs
[OK] check_doc_refs
[...] 14. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

Os 6 pulados sao os mesmos de antes desta tarefa (`vaulttest` fora do Windows e
afins); nenhum e novo, e nenhum e desta tool.

`pwsh -File scripts/check_tool_params.ps1`, os quatro campos novos:

```
    vaultBrokenLinksInput.State -> tools_read.go (mcpsrv), internal/ (dominio)
    vaultBrokenLinksInput.Prefix -> tools_read.go (mcpsrv), internal/ (dominio)
    vaultBrokenLinksInput.Limit -> tools_read.go (mcpsrv), internal/ (dominio)
    vaultBrokenLinksInput.Offset -> tools_read.go (mcpsrv), internal/ (dominio)

[OK] todo parametro declarado e lido em algum lugar.
```

`pwsh -File scripts/check_readme_anchors.ps1`:

```
[i] 11 heading(s), 11 link(s) interno(s).
[OK] toda ancora resolve e toda secao H2 e alcancavel pela navegacao.
```

`pwsh -File scripts/check_doc_refs.ps1`:

```
[OK] nenhum token entre crases parece citar artefato ausente do codigo.
```

UTF-8 dos nove `.md` editados, um `python -c "open(...,encoding='utf-8').read()"`
por arquivo: todos `[OK]`.

## Concerns

1. **Sem LSP.** Nenhuma ferramenta `mcp__gopls__*` esta exposta a este agente
   (conferido com `ToolSearch`, que devolve so ferramentas de browser e de
   memoria). Naveguei com `grep` e com o compilador. Nao e escolha minha
   ignorar a regra do dono; a ferramenta nao existe nesta sessao.

2. **`prefix` e prefixo de STRING, nao de segmento.** `"sub"` casa `sub/c.md` e
   tambem casaria `subtotal.md`. Segui o brief ("nao invente comparacao nova") e
   o precedente de `tag_list.prefix`, que e explicitamente prefixo de string, e
   escrevi isso no schema e em `docs/TOOLS.md` em vez de deixar implicito. Se o
   revisor quiser casamento por segmento (a conta de `folder` de `note_list` e
   outra), e uma decisao dele, nao um defeito escondido: esta documentado do
   jeito que se comporta.

3. **Escopo que cresceu, e por que.** O brief pedia "uma linha" no wiki. A linha
   nova tornou falso o titulo da pagina (`As 13 tools MCP`), o cabecalho
   `## Leitura (7)` (que ja estava errado antes de mim — a tabela tinha 8), a
   frase "Cinco das oito" e a `description` do frontmatter, alem do texto de
   link em quatro outras paginas do wiki. Corrigi os seis arquivos, porque
   numero errado em documento normativo e a classe de defeito que este projeto
   ja pagou. Nao encolhi escopo em silencio, mas tambem nao pedi permissao: se o
   revisor preferir o rename do wiki num commit separado, ele esta isolado nas
   cinco linhas de texto de link.

4. **`docs/PRD.md` nao foi tocado, de proposito.** Nao ha tabela de tools la; as
   tabelas sao de RF, e RF-63 ja cobre a capacidade de ancora quebrada. A unica
   mencao a uma lista de tools e a descricao do marco M2 (linha 446), que e
   historia — reescreve-la para incluir uma tool de 2026-09-06 seria falsificar
   o marco.

5. **`achadoQuebrado.start` nao vai para o retorno.** Ele existe so ate a
   ordenacao. `BrokenLink` ficou exatamente com os campos que o brief
   especificou; se o revisor quiser `start` no JSON (para o cliente pular direto
   para o offset, como `vault_search` faz com RF-27), e um acrescimo pequeno,
   mas seria contrato novo alem do brief e nao o inventei.

6. **Nenhuma medicao de desempenho.** Nao rodei benchmark desta tool. Custo por
   chamada num cofre real: **nao medido**. A tool percorre `NotePaths()` e os
   links de cada nota, como `VaultStats` com `include_health`, e mais uma
   ordenacao sobre os achados.

## audit_reports.ps1 183

```
=== Relatorios (1) ===

=== Ledger ===
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:9: [SHA-NAO-CONFERE] Task 4 ...
  ... (13 achados do marco 2026-07-25, todos anteriores a esta tarefa)
  .superpowers\sdd\2026-07-25-gobsidian-v01\progress.md:5158: [SHA-NAO-CONFERE] Task 153 ...

[!] 14 achado(s). Nenhum e automaticamente um defeito — cada um e
    uma frase ou um SHA que precisa de uma pessoa confirmando.
[i] Onde nao houve medicao, 'nao medido' e a resposta certa e nao e sinalizada.
```

O auditor achou **1 relatorio** (este) e **zero achados contra ele**: a secao
`=== Relatorios ===` saiu vazia. Os 14 achados sao todos do ledger do marco
`2026-07-25-gobsidian-v01`, anteriores a esta tarefa e fora do seu escopo.

**Ledger:** nao escrevi em `.superpowers/sdd/.../progress.md`. O brief diz que
`.superpowers/sdd/` e do orquestrador; a linha do ledger para a Task 183 e dele.
