---
title: As 12 tools MCP
type: feature
status: active
description: Superfície pública do servidor — 7 tools de leitura, 5 de escrita, e os resources.
source_paths:
  - internal/mcpsrv/server.go
  - internal/mcpsrv/tools_read.go
  - internal/mcpsrv/tools_write.go
  - internal/mcpsrv/resources.go
  - internal/mcpsrv/recover.go
source_commit: b2be492
tags: [mcp, tools, api]
language: pt-BR
updated_at: '2026-08-16'
---

# As 12 tools MCP

O contrato de cada uma está em `docs/TOOLS.md`. Esta página explica como elas se
ligam ao código.

## Leitura (7)

| Tool | Fachada | Toca o disco? |
|---|---|---|
| `vault_stats` | `service.VaultStats` | não |
| `vault_search` | `service.Search` | sim, para recortar trechos |
| `note_read` | `service.ReadNote` / `ReadNotes` | sim, só a faixa pedida |
| `note_list` | `service.ListNotes` | não |
| `note_metadata` | `service.NoteMetadata` | não |
| `link_graph` | `service.LinkGraph` | não |
| `tag_list` | `service.TagList` | não |

Cinco das sete respondem **só do índice em memória**. É o que torna `note_list` a
tool barata, e é por isso que ela devolve a projeção `ListItem` em vez da `Note`
inteira — despejar headings, blocos e links por nota transformaria "que notas
existem na pasta X" numa resposta de dezenas de milhares de tokens.

`note_read` aceita `path` **ou** `paths` (lote de até 50), mutuamente
exclusivos: os dois preenchidos é erro de validação, não precedência silenciosa.
No lote, a falha de um item não derruba os demais — cada um vira um
`ReadNoteItem` na mesma posição de `Paths`.

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

## Onde o schema mente hoje

Registrado em [Achados em aberto](../notes/achados-abertos.md): `tag_list.sort` e
`tag_list.hierarchical` são declarados no schema e nunca lidos, e
`max_results` é uma cadeia de configuração morta. É a mesma classe de defeito
<!-- wiki-refs: ignore max_results -- declarado no schema e ausente do codigo: e o defeito -->
que `note_list.fields` e `ensure_blank_line` já custaram — **ou implemente, ou
tire do schema e da documentação.**

## Ver também

- [Busca](busca.md) · [Escrita](escrita.md)
- `docs/TOOLS.md` — o contrato normativo.
