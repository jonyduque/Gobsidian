---
title: Parser
type: feature
status: active
description: Markdown do Obsidian — frontmatter, headings com offsets e quatro extensões goldmark.
source_paths:
  - internal/parser/parser.go
  - internal/parser/frontmatter.go
  - internal/parser/headings.go
  - internal/parser/ext_wikilink.go
  - internal/parser/ext_blockid.go
  - internal/parser/ext_tag.go
  - internal/parser/ext_inline_field.go
  - internal/parser/slug.go
source_commit: f7de8e81
tags: [parser, goldmark, markdown]
language: pt-BR
updated_at: '2026-08-31'
---

# Parser

Transforma bytes de uma nota em `ParsedNote`. **É puro**: sem I/O, sem estado, sem
conhecimento do índice ou do cofre. Isso o torna trivialmente testável por golden
file e trivialmente paralelizável — o worker pool de `index.Build` depende disso.

O `goldmark.Markdown` é construído **uma vez** como variável de pacote; ele é
seguro para uso concorrente após a construção.

## O que sai

```go
type ParsedNote struct {
    Title, FrontmatterErr string
    Frontmatter           map[string]any
    Tags, Aliases         []string
    Headings              []Heading
    Blocks                []Block
    Links                 []Link
    Inline                map[string][]string
}
```

## Frontmatter

`SplitFrontmatter` separa o bloco YAML e devolve **o offset em que o corpo
começa** — é ele que mantém os offsets de heading e de bloco corretos em relação
ao início do buffer.

> Exige entrada **já sem BOM**. Quem produz essa garantia é `vault.StripBOM`. Sem
> ela, `\xEF\xBB\xBF---` não bate com o prefixo `---`: o frontmatter não fica
> malformado, fica **invisível** — sem `FrontmatterErr`, com tags, aliases e
> título sumindo em silêncio.

Frontmatter quebrado **não invalida a nota**: o corpo continua tendo headings e
links úteis. O erro vai para `FrontmatterErr` — que atravessa o índice e **sai no
retorno de `note_metadata`**. Até 2026-08-28 o parser o preenchia e ninguém o
lia: a nota com YAML quebrado aparecia sem tags e sem aliases, sem nada dizendo
por quê.

## Headings

`ExtractHeadings` faz varredura própria em vez de usar a AST do goldmark, porque
precisa do **offset de fim de seção** — que é propriedade da hierarquia, não do
nó: o fim de uma seção é o início do próximo heading de nível menor ou igual.

Três offsets por heading, e a distinção importa:

| campo | aponta para |
|---|---|
| `Start` | o início da **linha** do heading (não do `#` — a indentação conta) |
| `BodyStart` | logo após a linha do heading — é o que `replace_section` usa |
| `End` | o início do próximo heading de nível ≤, ou o fim do buffer |

Blocos de código cercados são respeitados: um `#` dentro de uma cerca não é
heading. E a cerca de fechamento precisa ter pelo menos o tamanho da de abertura —
tratar qualquer linha de crases como indiferente faz uma cerca de quatro ser
fechada por uma de três, e dali em diante toda a hierarquia sai errada.

**Só ATX (`#`).** Setext (`Título` + `====`) e título em negrito não são
reconhecidos — ver [Achados em aberto](../notes/achados-abertos.md).

## As quatro extensões goldmark

| Extensão | Sintaxe | Observação |
|---|---|---|
| `WikilinkExtension` | `[[nota]]`, `[[nota#âncora\|alias]]`, `![[embed]]` | três `LinkKind` distintos |
| `BlockIDExtension` | `^id` no fim de linha | `Start` é o do **bloco pai**, não do `^` |
| `TagExtension` | `#tag`, `#tag/aninhada` | precisa não casar `#` de heading nem de URL |
| `InlineFieldExtension` | `campo:: valor` (Dataview) | **não pode consumir o span do valor** |

A última carrega uma lição: o campo inline consumia o span do valor, e
`fonte:: [[STJ]]` deixava de produzir link nenhum — links que o commit anterior já
coletava. **Feature P1 não tem direito de apagar dado P0.**

## `Slug`

Normaliza um heading para comparação com a âncora de um wikilink: minúsculas, sem
acento, sem pontuação, espaços colapsados. Reproduz o casamento permissivo do
Obsidian, que faz `[[nota#Capitulo 118]]` encontrar `## Capítulo 118`.

Pontuação vira **nada**, não espaço: `Art. 1.234` precisa virar `art 1234`, não
`art 1 234`.

O valor é pré-computado em `Heading.Slug` e persistido no cache — e desde
2026-08-28 **é lido**: `index.anchors`, `service.read` e `writer.section`
comparam contra ele em vez de recomputar. Eram três lugares recomputando um valor
que já vinha pronto do disco.

## `DetectCandidates` — o que PARECE título e não é

`parseATXHeading` só aceita ATX (`#`). Uma nota convertida de PDF, DOCX ou EPUB
não tem nenhum: ela marca título com parágrafo em negrito
(`**13.1.10 Substituição de candidatos**`) ou com setext.

Desde 2026-09-01 o candidato também resolve em `note_read(heading=)`, quando
nenhum heading ATX casa — mas **na leitura apenas**, e ainda assim sem virar
`Heading`: `service` converte na hora, e o retorno declara
`section_synthetic`.

`outline.go` detecta essas duas formas e devolve `[]Candidate` — **e o tipo é
outro de propósito**. Candidato não é `Heading`, não entra no índice e não entra
no cache. O parser não muda por causa dele, e `IndexCacheParserVersion` não sobe:
os candidatos são calculados na chamada de `note_outline`, sobre os bytes da
nota.

Três reusos, e cada um evita uma segunda conta:

- **`openFence` / `closesFence`** — a mesma máquina de cercas de
  `ExtractHeadings`. Duas máquinas de cerca divergem, e a divergência aparece
  como hierarquia falsa dentro de um bloco de código.
- **`closeSections`** — a mesma regra de `End`: o início do próximo de nível
  menor ou igual, ou o fim do arquivo.
- **`parseATXHeading`** — para um `---` logo abaixo de um heading ATX não
  transformar o heading em título de setext.

`Level` é ponteiro. `13` dá 1, `13.1` dá 2, `13.1.10` dá 3; **sem numeração não
há nível a afirmar**, e um zero literal ali mentiria. No cálculo de `End`,
candidato sem número entra como nível 6 — o mais profundo — para que qualquer
candidato numerado o feche sem que ele engula uma seção numerada seguinte.

Negrito conta **só sozinho na linha**: ênfase no meio de um parágrafo não é
título, e aceitá-la encheria a resposta de ruído.

## Golden files

48 arquivos congelam o parser e as quatro extensões, mais uma verificação de
paridade contra um dump real do `metadataCache` do Obsidian.

> **`-update` grava o que o código produz, não o que está certo.** Depois de
> gerar, leia cada arquivo e confira contra o que você esperava **antes** de rodar.

## Ver também

- [Nota e caminho](../entities/note-e-caminho.md) · [Busca](busca.md)
