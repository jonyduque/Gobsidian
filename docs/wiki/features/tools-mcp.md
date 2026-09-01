---
title: As 13 tools MCP
type: feature
status: active
description: Superfície pública do servidor — 8 tools de leitura, 5 de escrita, e os resources.
source_paths:
  - internal/mcpsrv/server.go
  - internal/mcpsrv/tools_read.go
  - internal/mcpsrv/tools_write.go
  - internal/mcpsrv/resources.go
  - internal/mcpsrv/recover.go
source_commit: f7de8e81
tags: [mcp, tools, api]
language: pt-BR
updated_at: '2026-08-31'
---

# As 13 tools MCP

O contrato de cada uma está em `docs/TOOLS.md`. Esta página explica como elas se
ligam ao código.

## Leitura (7)

| Tool | Fachada | Toca o disco? |
|---|---|---|
| `vault_stats` | `service.VaultStats` | não |
| `vault_search` | `service.Search` | sim, para recortar trechos |
| `note_read` | `service.ReadNote` / `ReadNotes` | sim, só a faixa pedida |
| `note_outline` | `service.Outline` | sim, a nota inteira |
| `note_list` | `service.ListNotes` | não |
| `note_metadata` | `service.NoteMetadata` | não |
| `link_graph` | `service.LinkGraph` | não |
| `tag_list` | `service.TagList` | não |

Cinco das oito respondem **só do índice em memória**. É o que torna `note_list` a
tool barata, e é por isso que ela devolve a projeção `ListItem` em vez da `Note`
inteira — despejar headings, blocos e links por nota transformaria "que notas
existem na pasta X" numa resposta de dezenas de milhares de tokens.

`note_read` aceita `path` **ou** `paths` (lote de até 50), mutuamente
exclusivos: os dois preenchidos é erro de validação, não precedência silenciosa.
No lote, a falha de um item não derruba os demais — cada um vira um
`ReadNoteItem` na mesma posição de `Alvos`.

**Cada item de `paths` é uma string ou um objeto**, misturados na mesma lista, e
o objeto sobrepõe os campos de topo **campo a campo** só para aquele item. Seis
capítulos com seis headings diferentes numa chamada, e não seis. Os campos do
objeto são ponteiros porque `max_bytes: 0` ("sem teto") é um pedido diferente de
omitir `max_bytes` (herdar o do topo).

**`note_read(heading=)` resolve CANDIDATO quando nenhum heading Markdown casa**,
desde 2026-09-01. Heading de verdade sempre vence; o retorno traz
`section_synthetic` para palpite não passar por estrutura; e a ESCRITA fica de
fora, porque os limites de um candidato são heurística — ler no lugar errado
devolve o parágrafo errado, escrever no lugar errado apaga trabalho. Calculado na
chamada e só no caminho de fallback: leitura que casa no índice não paga por
isso, e nada é persistido.

`note_outline` é a única que lê a nota **inteira**, e por isso recusa
somente-nuvem. Ela existe porque `parseATXHeading` só aceita `#`, e nota
convertida de PDF/DOCX/EPUB marca título com parágrafo em negrito — nessas notas,
`note_read` por heading, `note_patch` por seção, âncora de wikilink e o peso de
heading do BM25 não funcionam. Ela não conserta as quatro: separa o que é
estrutura do que é palpite e **diz qual é qual**.

## Escrita (5)

`note_create`, `note_append`, `note_patch`, `note_move`, `note_delete`.

Com `--read-only` ligado, elas **não são registradas** — e não apenas recusadas
na chamada. Um host que vê a tool anunciada vai tentar usá-la, e a recusa vira
uma rodada desperdiçada.

Todas aceitam `dry_run`, que devolve o diff unificado sem tocar no disco.
Ver [Escrita](escrita.md).

## Resources

`internal/mcpsrv/resources.go` registra um *template* de resource
(`obsidian://<caminho>`) mais uma entrada por nota. É a via de leitura que o
protocolo oferece ao lado das tools.

## Duas regras do adaptador

**`guard` embrulha todo handler.** Um panic vira resultado de erro com stack
trace no stderr, nunca derruba o servidor (RNF-13). É o que distingue um servidor
robusto de um que exige reiniciar o host toda vez que um caminho inválido passa.

**Erro de tool é devolvido como `error` Go, não como resultado.** `toolErr`
prefixa o código legível por máquina e devolve. O SDK então monta o
`CallToolResult` de erro **sem** serializar o valor zero de `Out` em
`StructuredContent` — sintetizar o resultado à mão deixava `structuredContent`
com um objeto zerado indistinguível de um cofre legitimamente vazio.

A exceção é `noteReadValidationError`, que monta o resultado à mão de propósito:
num erro de lote, o cliente se beneficia de `Out` estruturado para inspecionar o
código por campo.

## Onde o schema já mentiu, e o gate que pega

`tag_list.sort` e `tag_list.hierarchical` foram declarados no schema e **nunca
lidos** até 2026-08-28; hoje o handler os repassa ao domínio. `max_results` era
<!-- wiki-refs: ignore max_results -- nome de flag de CLI, nao de campo de schema -->
cadeia de configuração morta no modo daemon, porque não atravessava a saudação —
fechado pelo protocolo 2. Mesma classe de `note_list.fields` e
`ensure_blank_line`: **ou implemente, ou tire do schema e da documentação.**

Quem pega isso é `scripts/check_tool_params.ps1`, e ele também já passou verde
por cima do defeito: casava nome de campo no **pacote inteiro**, então
`tag_list.sort` passava porque outra tool tem um `Sort` que é lido. Desde a
Task 104 ele decide por `$pVar.$Campo` com `-cmatch`, e struct sem handler
resolvido vira achado (`HANDLER-NAO-RESOLVIDO`) em vez de cair para o pacote.
Limite conhecido, escrito no cabeçalho do script: o nível 2 casa `.Campo` em
qualquer lugar de `internal/`, sem escopo da struct de destino.

## Ver também

- [Busca](busca.md) · [Escrita](escrita.md)
- `docs/TOOLS.md` — o contrato normativo.
