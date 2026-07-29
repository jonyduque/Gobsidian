# Superfície MCP — gobsidian

Contrato de cada *tool*. Schemas em JSON Schema, como declarados ao host.

---

## Convenções gerais

**Caminhos.** Todo parâmetro `path` é relativo à raiz do cofre, com separador `/`, incluindo a extensão `.md`. Exemplo: `Civil/PONTO 03.md`. Caminhos absolutos são rejeitados. Travessia para fora do cofre é rejeitada.

**Casing.** A resolução tenta correspondência exata primeiro, depois insensível a maiúsculas. Se a busca insensível encontrar mais de um candidato, a chamada falha com erro de ambiguidade listando os candidatos — nunca escolhe por conta própria.

**Limites.** Toda tool que devolve lista aceita `limit` e `offset`. O padrão de `limit` é 50; o teto é 500. Respostas truncadas trazem `truncated: true` e `total`.

**Dry-run.** Toda tool de escrita aceita `dry_run`. Quando verdadeiro, devolve o diff unificado do que seria feito e não toca o disco.

**Erros.** Devolvidos como resultado MCP de erro, com `code` legível por máquina e `message` acionável por humano ou modelo. O servidor jamais cai por erro de tool.

**Título.** O campo `title`, presente em vários retornos, é resolvido nesta ordem: campo `title` do frontmatter, primeiro heading de nível 1 do corpo, nome do arquivo sem a extensão. Sempre tem valor.

**Hash.** `note_read`, `note_list` e `note_metadata` devolvem `hash`, um xxhash do conteúdo bruto do arquivo — frontmatter e BOM incluídos. É o valor aceito em `expected_hash` nas tools de escrita, e a única forma de detectar que o Obsidian alterou a nota entre a leitura e a escrita.

**Filtro `frontmatter`.** Presente em `vault_search` e `note_list`. Um objeto de pares chave/valor, todos os quais precisam casar. As regras:

| Situação | Casa quando |
|---|---|
| Valor escalar, campo escalar | São iguais, com comparação de string insensível a maiúsculas e acentos |
| Valor escalar, campo lista | A lista contém o valor |
| Valor lista, campo lista | O campo contém **todos** os valores |
| Valor `null` | O campo existe, com qualquer valor |
| Chave com ponto (`meta.autor`) | Navega em objetos aninhados |
| Campo ausente | Nunca casa, exceto se o valor pedido for `null` e a chave existir |

Datas do frontmatter são comparadas como datas quando ambos os lados são parseáveis como tal, e como string caso contrário. Não há operadores de intervalo aqui — para isso existem `modified_after` e `modified_before`, que operam sobre o mtime do arquivo, não sobre campos do frontmatter.

---

# Leitura

## `vault_search`

Busca full-text com ranking, combinável com filtros de metadados.

```json
{
  "type": "object",
  "properties": {
    "query":        { "type": "string", "description": "Termos de busca. Aspas duplas delimitam frase exata." },
    "folder":       { "type": "string", "description": "Restringe a uma pasta e suas subpastas." },
    "tags":         { "type": "array", "items": { "type": "string" }, "description": "Notas que contenham TODAS as tags." },
    "frontmatter":  { "type": "object", "description": "Pares chave/valor que devem casar no frontmatter." },
    "modified_after":  { "type": "string", "format": "date-time" },
    "modified_before": { "type": "string", "format": "date-time" },
    "snippet_chars": { "type": "integer", "default": 240, "maximum": 1000 },
    "limit":  { "type": "integer", "default": 20, "maximum": 200 },
    "offset": { "type": "integer", "default": 0 }
  },
  "required": ["query"]
}
```

**Retorno.** Lista de resultados ordenada por score BM25 decrescente, cada um com `path`, `title`, `score`, `snippet` (com os termos casados marcados), `matched_headings` e `modified`.

**Notas.** A consulta é normalizada: sem acentos, sem distinção de maiúsculas, com stemming leve para português. Buscar `usucapiao` encontra `usucapião`. Não há remoção de stopwords — em corpus técnico, palavras frequentes costumam ser termos de arte.

Se `query` for vazio mas houver filtros, a chamada é redirecionada ao caminho de consulta de metadados, muito mais barato. Ainda assim, prefira `note_list` para consultas puramente estruturais.

---

## `note_read`

Lê uma nota inteira, uma seção, ou um bloco.

```json
{
  "type": "object",
  "properties": {
    "path":     { "type": "string" },
    "heading":  { "type": "string", "description": "Texto do heading. Lê a seção até o próximo heading de nível igual ou superior." },
    "heading_level": { "type": "integer", "minimum": 1, "maximum": 6, "description": "Desambigua quando o mesmo texto aparece em níveis diferentes." },
    "block_id": { "type": "string", "description": "Identificador de bloco, sem o circunflexo." },
    "include_frontmatter": { "type": "boolean", "default": true },
    "max_bytes": { "type": "integer", "default": 100000 }
  },
  "required": ["path"]
}
```

**Retorno.** `content`, `path`, `hash`, `truncated`, `total_bytes`, e — quando `heading` foi usado — `section` com nível, texto e faixa de offsets.

**Notas.** `block_id` é mutuamente exclusivo com `heading` e `heading_level`; os dois últimos combinam entre si, onde `heading_level` desambigua. A correspondência de heading é feita sobre o slug normalizado, então `## Capítulo 118` casa com `"Capítulo 118"`, `"capitulo 118"` ou `"CAPÍTULO 118"`.

`hash` é do arquivo inteiro, não da seção lida, e é o valor a devolver em `expected_hash` nas tools de escrita.

Ler uma seção lê apenas os bytes da seção. Uma seção de 2 KB em uma nota de 500 KB custa 2 KB.

**Erros.** `HEADING_NOT_FOUND` inclui na mensagem os headings disponíveis no mesmo nível — permite que o cliente se corrija sem uma chamada adicional.

---

## `note_list`

Lista notas por critérios estruturais. Não toca o índice de texto.

```json
{
  "type": "object",
  "properties": {
    "folder":    { "type": "string" },
    "glob":      { "type": "string", "description": "Padrão de caminho, ex.: 'Civil/PONTO *.md'" },
    "tags":      { "type": "array", "items": { "type": "string" } },
    "tag_mode":  { "type": "string", "enum": ["all", "any"], "default": "all" },
    "frontmatter": { "type": "object" },
    "recursive": { "type": "boolean", "default": true },
    "sort":      { "type": "string", "enum": ["path", "modified", "size", "title"], "default": "path" },
    "order":     { "type": "string", "enum": ["asc", "desc"], "default": "asc" },
    "fields":    { "type": "array", "items": { "type": "string" }, "description": "Campos de frontmatter a incluir no retorno." },
    "limit":     { "type": "integer", "default": 100, "maximum": 500 },
    "offset":    { "type": "integer", "default": 0 }
  }
}
```

**Retorno.** Lista com `path`, `title`, `hash`, `modified`, `size`, `tags`, e os campos pedidos em `fields`.

**Notas.** Servida direto do índice em memória. Latência de microssegundos. É a tool correta para "que notas existem na pasta X", "que notas têm a tag Y" — usar `vault_search` para isso é ordens de grandeza mais caro.

Lista apenas notas `.md`. Anexos são indexados para que embeds resolvam (PRD RF-60), mas não aparecem aqui; para inventário de anexos, use `vault_stats`.

---

## `note_metadata`

Metadados estruturais completos de uma nota, sem o corpo.

```json
{
  "type": "object",
  "properties": {
    "path": { "type": "string" },
    "include": {
      "type": "array",
      "items": { "type": "string", "enum": ["frontmatter", "tags", "headings", "blocks", "links", "backlinks", "inline_fields"] },
      "default": ["frontmatter", "tags", "headings", "links", "backlinks"]
    }
  },
  "required": ["path"]
}
```

**Retorno.** Sempre `path`, `title` e `hash`. Os demais campos conforme pedido em `include`. `headings` traz nível, texto, slug e offsets — o que permite planejar uma leitura ou uma escrita seletiva antes de fazê-la. `links` distingue wikilink, embed e link Markdown, e marca os não resolvidos. `backlinks` traz origem e o contexto textual ao redor de cada referência.

Cada entrada de `links` traz `state` com um de três valores, e `via` com a forma pela qual resolveu:

| `state` | Significado |
|---|---|
| `ok` | Alvo existe; se há âncora, a âncora existe |
| `target_missing` | Nota ou anexo alvo não existe |
| `anchor_missing` | Alvo existe, mas o heading ou bloco da âncora não |

| `via` | Resolveu por |
|---|---|
| `path` | Caminho relativo explícito no link |
| `name` | Nome de arquivo de nota |
| `asset` | Nome de arquivo de anexo |
| `alias` | Campo `aliases` do frontmatter do alvo |

`anchor_missing` é o estado que o Obsidian não sinaliza e que aparece sozinho depois de renomear um heading.

**Notas.** Esta é a tool de reconhecimento. Chamar `note_metadata` antes de `note_patch` é o fluxo correto: descobre-se que headings existem e como estão grafados antes de tentar escrever sob um deles.

---

## `link_graph`

Vizinhança de links de uma nota.

```json
{
  "type": "object",
  "properties": {
    "path":      { "type": "string" },
    "direction": { "type": "string", "enum": ["outgoing", "incoming", "both"], "default": "both" },
    "depth":     { "type": "integer", "default": 1, "minimum": 1, "maximum": 3 },
    "include_broken":  { "type": "boolean", "default": true },
    "include_embeds":  { "type": "boolean", "default": true },
    "limit":     { "type": "integer", "default": 100, "maximum": 500 }
  },
  "required": ["path"]
}
```

**Retorno.** `nodes` (caminho, título, distância da origem) e `edges` (origem, destino, tipo, alias, âncora, resolvido).

**Notas.** `depth` acima de 2 pode devolver uma fração grande do cofre em bases densamente ligadas. O teto de 3 é intencional.

---

## `tag_list`

Todas as tags do cofre.

```json
{
  "type": "object",
  "properties": {
    "prefix":    { "type": "string", "description": "Restringe a uma subárvore, ex.: 'civil/'" },
    "min_count": { "type": "integer", "default": 1 },
    "sort":      { "type": "string", "enum": ["name", "count"], "default": "count" },
    "hierarchical": { "type": "boolean", "default": false, "description": "Retorna árvore em vez de lista plana." }
  }
}
```

---

## `vault_stats`

Estado do cofre e saúde do servidor.

```json
{
  "type": "object",
  "properties": {
    "include_health":  { "type": "boolean", "default": true },
    "include_runtime": { "type": "boolean", "default": false }
  }
}
```

**Retorno.**

- Contagem de notas, tamanho total, contagem de links, contagem de tags
- Contagem de anexos e tamanho total deles
- Notas órfãs (sem backlinks), links quebrados, âncoras quebradas, notas vazias
- Colisões de alias: aliases declarados por mais de uma nota
- Notas somente-nuvem não hidratadas
- Timestamp da última indexação e duração
- Com `include_runtime`: `runtime` (RSS, goroutines, gc) e objeto `watcher` (ausente se desligado) com os campos: `active`, `events_received`, `events_dropped`, `events_dropped_by_reason`, `events_coalesced`, `events_processed`, `events_skipped`, `reconciliations`, `reconciled_updated`, `reconciled_removed`.

**Notas.** Os contadores do watcher são a instrumentação principal para diagnosticar cofres em pastas sincronizadas. `events_received` conta os eventos antes do filtro de relevância (brutos), e `events_dropped` conta os irrelevantes ou ocultos. `events_dropped_by_reason` desdobra o total porque as causas pedem ações diferentes: `chmod` alto é OneDrive em operação normal e pode ser ignorado; `outside_vault` alto indica que a raiz do cofre é um link e o confinamento está recusando eventos; `excluded` alto indica atividade em `.obsidian/` ou `.git/`; `unknown_op` alto indica evento que o filtro não soube classificar e merece `--log-level debug`. `events_coalesced` conta eventos adicionais na mesma nota dentro da janela de debounce. `events_processed` conta o número de absorções de mudanças efetivas, e `events_skipped` as absorções rejeitadas (mesmo mtime e tamanho). `reconciled_updated` e `reconciled_removed` registram os arquivos corrigidos por reconciliação de overflow. Uma razão alta entre `events_received` e `events_processed` é o comportamento esperado e saudável; overflows recorrentes (contabilizados em `reconciliations`) indicam que a janela de debounce precisa ser ampliada.

---

# Escrita

Todas aceitam `dry_run`. Todas preservam o estilo de fim de linha do arquivo. Nenhuma reformata conteúdo que não seja o alvo explícito da operação.

## `note_create`

```json
{
  "type": "object",
  "properties": {
    "path":        { "type": "string" },
    "content":     { "type": "string" },
    "frontmatter": { "type": "object" },
    "create_folders": { "type": "boolean", "default": true },
    "dry_run":     { "type": "boolean", "default": false }
  },
  "required": ["path", "content"]
}
```

**Notas.** Falha se o caminho já existir. Não há sobrescrita silenciosa; para substituir, use `note_patch` ou exclua explicitamente.

---

## `note_append`

```json
{
  "type": "object",
  "properties": {
    "path":    { "type": "string" },
    "content": { "type": "string" },
    "heading": { "type": "string", "description": "Anexa ao fim desta seção. Ausente, anexa ao fim do arquivo." },
    "heading_level":    { "type": "integer", "minimum": 1, "maximum": 6 },
    "create_if_missing": { "type": "boolean", "default": false, "description": "Cria o heading ao fim do arquivo se não existir." },
    "ensure_blank_line": { "type": "boolean", "default": true },
    "expected_hash": { "type": "string", "description": "Hash da nota obtido em leitura anterior. Se divergir, a chamada falha com HASH_MISMATCH." },
    "dry_run": { "type": "boolean", "default": false }
  },
  "required": ["path", "content"]
}
```

**Notas.** "Fim da seção" é imediatamente antes do próximo heading de nível igual ou superior — não o fim do arquivo. Anexar sob `## Capítulo 117` insere antes de `## Capítulo 118`, não depois de tudo.

`create_if_missing` é falso por padrão, deliberadamente. Um heading errado deve produzir erro, não uma seção nova em lugar inesperado.

---

## `note_patch`

```json
{
  "type": "object",
  "properties": {
    "path":     { "type": "string" },
    "content":  { "type": "string" },
    "heading":  { "type": "string" },
    "heading_level": { "type": "integer", "minimum": 1, "maximum": 6 },
    "block_id": { "type": "string" },
    "mode":     { "type": "string", "enum": ["replace_section", "replace_heading_and_section", "replace_block"], "default": "replace_section" },
    "expected_hash": { "type": "string", "description": "Hash da nota obtido em leitura anterior. Se divergir, a chamada falha." },
    "dry_run":  { "type": "boolean", "default": false }
  },
  "required": ["path", "content"]
}
```

**Notas.** `replace_section` preserva a linha do heading e substitui apenas o conteúdo abaixo dela, incluindo subseções. `replace_heading_and_section` substitui também a linha do heading.

`expected_hash` implementa concorrência otimista. Obtido de `note_metadata` ou `note_read`, garante que a nota não mudou entre a leitura e a escrita. Recomendado sempre que houver possibilidade de o Obsidian estar aberto na nota.

Sem `expected_hash`, o servidor ainda verifica internamente se o conteúdo mudou desde o último parse e recalcula os offsets antes de aplicar — o que impede corrupção, mas não impede sobrescrever uma edição concorrente que o cliente não viu.

---

## `note_move`

```json
{
  "type": "object",
  "properties": {
    "from":   { "type": "string" },
    "to":     { "type": "string" },
    "update_links":   { "type": "boolean", "default": true },
    "create_folders": { "type": "boolean", "default": true },
    "dry_run": { "type": "boolean", "default": false }
  },
  "required": ["from", "to"]
}
```

**Retorno.** Caminho novo, lista de notas cujos links foram reescritos, contagem de links atualizados e, em `dry_run`, o diff de cada nota afetada.

**Notas.** A reescrita preserva a forma original de cada link: alias, âncora de heading ou de bloco, e a escolha entre wikilink e link Markdown. Um `[[Civil/PONTO 03|Ponto 3 — Obrigações]]` continua com o mesmo alias após a movimentação.

Não há transação entre arquivos. Cada arquivo é escrito atomicamente e o resultado reporta com precisão o que foi aplicado se algo falhar no meio. **Recomendação forte: rodar com `dry_run` antes de qualquer movimentação em lote.**

---

## `note_delete`

```json
{
  "type": "object",
  "properties": {
    "path":    { "type": "string" },
    "to_trash": { "type": "boolean", "default": true, "description": "Move para .trash/ do cofre em vez de excluir." },
    "report_broken_links": { "type": "boolean", "default": true },
    "dry_run": { "type": "boolean", "default": false }
  },
  "required": ["path"]
}
```

**Notas.** `to_trash` verdadeiro por padrão. Exclusão definitiva exige passá-lo explicitamente como falso.

Com `report_broken_links`, o retorno lista as notas que passarão a ter links quebrados — informação que frequentemente muda a decisão.

---

# Resources

Notas são expostas como *resources* MCP, permitindo que o host as anexe ao contexto sem chamada de tool.

| Campo | Valor |
|---|---|
| URI | `gobsidian:///<caminho-canônico>`, com escape percent |
| MIME | `text/markdown` |
| Nome | Título da nota (H1, ou nome do arquivo) |
| Descrição | Primeiros 200 caracteres do corpo |

**Três barras, e o caminho vem escapado.** `gobsidian://` seguido do caminho parece natural e está errado: em RFC 3986, o que vem logo depois de `//` é a **autoridade**, não o caminho. Com duas barras, `Civil/PONTO 03.md` faz `Civil` virar nome de host — e uma nota na raiz do cofre com espaço no nome faz o host inteiro ser `Minha nota.md`, que é inválido. O servidor morria no boot, dentro do registro do resource, antes de anunciar qualquer tool.

A terceira barra declara autoridade vazia e faz o caminho começar onde deve. Os bytes fora de `A-Za-z0-9-._~` viram `%XX`; a `/` permanece crua, porque separa segmentos. Assim `Civil/PONTO 03.md` é publicado como `gobsidian:///Civil/PONTO%2003.md`.

Na leitura o servidor também aceita a forma antiga de duas barras, para não transformar documentação desatualizada em nota inalcançável.

A listagem de resources é paginada e serve o índice em memória. Em cofres grandes, listar todas as notas como resources é caro para o host; a listagem respeita um limite configurável (padrão: 200, ordenadas por data de modificação decrescente).

---

# Códigos de erro

| Código | Significado | Ação sugerida ao cliente |
|---|---|---|
| `PATH_OUTSIDE_VAULT` | Caminho resolve para fora do cofre | Corrigir o caminho |
| `NOTE_NOT_FOUND` | Nota inexistente | Verificar com `note_list` |
| `NOTE_ALREADY_EXISTS` | Destino de criação ou movimentação já existe | Escolher outro caminho ou usar `note_patch` |
| `AMBIGUOUS_PATH` | Mais de uma nota casa insensível a maiúsculas | Usar o caminho exato listado na mensagem |
| `HEADING_NOT_FOUND` | Heading inexistente | A mensagem lista os headings disponíveis |
| `AMBIGUOUS_HEADING` | Mesmo texto em mais de um lugar | Passar `heading_level` |
| `BLOCK_NOT_FOUND` | Identificador de bloco inexistente | Verificar com `note_metadata` |
| `HASH_MISMATCH` | Nota mudou desde a leitura | Reler e repetir |
| `FILE_LOCKED` | Arquivo bloqueado após esgotar retries | Fechar a nota no Obsidian e repetir |
| `CLOUD_ONLY_FILE` | Arquivo não hidratado pelo OneDrive | Abrir uma vez no Explorer, ou desmarcar "somente online" |
| `PATH_TOO_LONG` | Caminho excede o limite do sistema | Encurtar o caminho ou habilitar caminhos longos |
| `READ_ONLY_MODE` | Servidor iniciado com `--read-only` | Reiniciar sem a flag |
| `VAULT_UNAVAILABLE` | Raiz do cofre inacessível | Verificar com `gobsidian doctor` |
| `INTERNAL` | Falha inesperada, já registrada em stderr | Reportar |
