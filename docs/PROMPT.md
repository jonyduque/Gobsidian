# Prompt padrão para quem usa o gobsidian

O host MCP recebe, de cada tool, **só `type` e `description`** — limitação de
`jsonschema-go v0.4.2`, registrada em [`TOOLS.md`](TOOLS.md) na seção "Schemas
servidos". Nenhum `enum`, `minimum`, `maximum` ou `default` chega ao modelo.
Ou seja: os conjuntos fechados existem, o servidor os cobra com
`INVALID_ARGUMENT` — e o modelo não tem como saber quais são.

É esse buraco que o prompt abaixo tapa. Sem ele o modelo acerta os nomes das
tools e erra os valores, e descobre cada conjunto fechado por tentativa.

**Este arquivo não é normativo.** O contrato de cada tool é o de
[`TOOLS.md`](TOOLS.md); aqui só está o que o host não consegue transmitir.
`scripts/check_prompt.ps1` prova, a cada `verify.ps1`, que os conjuntos citados
abaixo são os mesmos que o código cobra.

---

## Onde colar

| Host | Onde |
|---|---|
| Claude Code | `CLAUDE.md` do projeto, ou `~/.claude/CLAUDE.md` |
| Claude Desktop | Instruções do Project |
| Codex / Copilot | `AGENTS.md` |
| Outros | O campo de instruções de sistema que o host oferecer |

Cole o bloco inteiro. Ele é escrito para ser lido por um modelo, não por uma
pessoa: é denso de propósito.

---

## O prompt

```text
Você acessa um cofre Obsidian pelo servidor MCP gobsidian. As regras abaixo
são do servidor, não do host: valor fora de um conjunto fechado volta como
INVALID_ARGUMENT.

CAMINHOS
- Todo `path` é relativo à raiz do cofre, com `/` e com a extensão `.md`.
  Exemplo: `Civil/PONTO 03.md`.
- Caminho absoluto e travessia para fora do cofre são recusados.
- A resolução tenta caixa exata e depois insensível a maiúsculas. Se a busca
  insensível achar mais de um candidato, a chamada FALHA e lista os
  candidatos — o servidor nunca escolhe por você. Escolha e repita.

ANTES DE ESCREVER
- Toda tool de escrita aceita `dry_run`. Use-o quando a nota importa: devolve
  o diff unificado e não toca o disco.
- `note_read`, `note_list` e `note_metadata` devolvem `hash`. Passe-o em
  `expected_hash` na escrita. É a ÚNICA forma de detectar que o Obsidian
  mexeu na nota entre a sua leitura e a sua escrita — sem ele, você
  sobrescreve o que o usuário acabou de digitar.
- O cofre pode estar em modo somente-leitura. Nesse caso as tools de escrita
  não existem na sessão; não prometa a edição antes de ver a tool.

CONJUNTOS FECHADOS (o schema não os carrega; estes são os valores válidos)
- link_graph.direction: both, outgoing, incoming
- note_list.sort: path, modified, size, title
- note_list.order: asc, desc
- note_list.tag_mode: all, any
- tag_list.sort: name, count
- vault_broken_links.state: target_missing, anchor_missing
- note_metadata.include: frontmatter, tags, headings, blocks, links, backlinks, inline_fields
- note_patch.mode: replace_section, replace_heading_and_section, replace_block
  O padrão é o PRIMEIRO valor de cada linha, com duas exceções: `state` e
  `include` omitidos valem "todos", e `note_patch.mode` passa a
  `replace_block` quando você manda `block_id`.
  `note_metadata` sem `include` NÃO traz `blocks` nem `inline_fields`, que
  crescem com o tamanho da nota — peça-os só quando forem o assunto.
  Em `note_patch`, `heading` e `block_id` juntos são INVALID_ARGUMENT.
  `expected_hash` que não confere volta como HASH_MISMATCH: releia a nota,
  não repita a escrita.

LIMITES
- Toda lista aceita `limit` e `offset`. vault_search: padrão 20, teto 200.
  note_list, link_graph e vault_broken_links: padrão 100, teto 500.
- Valor acima do teto volta CLAMPADO, não recusado; o efetivo aparece em
  `effective_*`. Resposta cortada traz `truncated: true` e `total` — se vier
  `truncated`, pagine com `offset` antes de concluir qualquer contagem.

COMO GASTAR MENOS E ACERTAR MAIS
- Prefira `vault_search` a listar o cofre e filtrar por conta própria: o
  ranking é BM25 sobre um índice, e a filtragem local lê o que não precisa.
- `note_metadata` responde estrutura (headings, links, backlinks, tags) SEM
  abrir o arquivo. Use-o para planejar antes de ler.
- `note_read` aceita `heading` ou `block_id` para ler só um trecho, e aceita
  lote. Ler a nota inteira para usar dois parágrafos é o desperdício comum.
- `note_outline` dá o esqueleto de headings; é o passo natural antes de um
  `note_patch` com `replace_section`.
- O filtro `frontmatter` (em vault_search e note_list) casa: escalar contra
  escalar por igualdade sem acento e sem caixa; escalar contra lista por
  pertinência; lista contra lista exigindo TODOS; `null` significa "o campo
  existe". Chave com ponto navega em objeto aninhado (`meta.autor`).
  Para intervalo de data use `modified_after`/`modified_before`, que olham o
  mtime do arquivo, não o frontmatter.

O QUE O SERVIDOR NÃO FAZ
- Anexo é indexado pelo nome e nunca lido. Não peça o conteúdo de um `.pdf`
  ou `.png` — não há.
- Arquivo somente-nuvem (OneDrive) nunca é aberto, porque abrir dispara
  download. Ele aparece nas listas e não no conteúdo.
- Erro de tool volta como resultado de erro com `code` e `message`; o
  servidor não cai. Leia o `code` e corrija a chamada em vez de repetir.

TÍTULO
- O campo `title` sai, nesta ordem: `title` do frontmatter, primeiro heading
  de nível 1, nome do arquivo sem extensão. Sempre tem valor — não o invente
  a partir do caminho.
```

---

## Por que cada bloco está aí

**Caminhos e casing.** O erro de ambiguidade é deliberado: `docs/TOOLS.md`
registra que o servidor **nunca** escolhe entre dois candidatos. Um modelo que
não sabe disso trata a falha como bug e repete a mesma chamada.

**`expected_hash`.** É a única defesa contra escrita sobre edição concorrente do
Obsidian, e nada no schema a anuncia. Sem esta linha o modelo escreve sem hash
porque o parâmetro parece opcional — e ele é opcional, e é assim que se perde
texto do usuário.

**Conjuntos fechados.** A razão de o arquivo existir. São cobrados em
`internal/service` por `ValidarEnum`, e `check_prompt.ps1` compara esta lista
com aquelas chamadas.

**Clamp contra recusa.** `limit: 5000` não falha: volta 500. Um modelo que
assume falha-ou-obedece conclui que o cofre tem 500 notas.

**Anexos e somente-nuvem.** Duas regras de produto que só aparecem como ausência
de resultado. Ver `docs/WINDOWS.md` para o mecanismo do somente-nuvem.
