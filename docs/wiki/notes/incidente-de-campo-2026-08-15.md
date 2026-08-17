---
title: Incidente de campo de 2026-08-15
type: note
status: active
description: Sessao real do Claude Desktop em que o modelo abandonou o gobsidian e usou outro servidor
  MCP, e os cinco defeitos que a causaram.
source_paths:
- docs/REVISAO-2026-08-15.md
source_commit: b2be4925
tags:
- revisao
- incidente
- uso-real
language: pt-BR
updated_at: '2026-08-16'
---

# Incidente de campo de 2026-08-15

Uma sessão real do Claude Desktop tentou responder três perguntas de Direito
Eleitoral consultando `Livros/Direito Eleitoral/13 Registro de candidatura.md`
— 255.568 caracteres, convertida de livro. **O modelo abandonou o gobsidian e
usou outro servidor MCP** (`obsidian2`), que estourou o limite de tokens do
host, gravou o resultado em arquivo temporário e obrigou o modelo a fatiar o
texto por posição de caractere com Python.

Instado a tentar de novo pelo gobsidian, as duas ferramentas falharam:

```
note_read {"heading":"13.1.10 Substituição de candidatos", "path":"..."}
→ HEADING_NOT_FOUND: heading "..." nao encontrado. Disponiveis:

vault_search {"query":"\"13.1.10 Substituição de candidatos\"", "snippet_chars":4000}
→ acha a nota (score 16,4) e devolve um trecho do TOPO do arquivo
```

Conclusão do modelo, na sessão: *"como a nota não tem headings reais, nenhuma
das duas ferramentas do gobsidian consegue recortar a seção específica"*. Ele
estava certo no diagnóstico e o produto ficou sem resposta.

**Não é caso de borda.** É o caso de uso central — cofre de estudo com material
convertido — falhando por inteiro, e sem nenhum sinal no log.

## Os cinco defeitos que o causaram

- **Não há `offset` em `note_read`.** As únicas formas de recortar são heading e
  bloco; falhando as duas, resta ler tudo. `vault.ReadRange` já existe e já é
  usada pelo recorte de trecho — falta o campo no schema. Observado **duas
  vezes** na mesma sessão, a segunda num capítulo de ~56 KB.
- **A âncora do trecho é a primeira ocorrência do primeiro termo da consulta**,
  não o melhor casamento. `13.1.10 Substituição de candidatos` tokeniza como
  `13, 1, 10, substituicao, de, candidatos`; a primeira ocorrência de `13` está
  no offset ~0. O IDF já é calculado em `CalculateBM25` e descartado; a posição
  do casamento já é calculada em `matchPhraseInNote` e descartada.
- **`HEADING_NOT_FOUND` com lista vazia não diz o que houve.**
  `strings.Join(nil, ", ")` é `""`, e a mensagem parece truncamento em vez de
  diagnóstico. Três estados distintos — nota sem heading algum, nenhum casa, o
  filtro de nível excluiu todos — produzem a mesma resposta.
- **Estrutura em negrito não é reconhecida como seção.** `parseATXHeading` só
  aceita `#`. Não aceita `**13.1.10 Substituição de candidatos**`, que é o que
  toda conversão de PDF/DOCX/EPUB produz, nem setext, que é CommonMark válido.
- **`vault_search` não devolve o offset do casamento.** `Snippet` carrega
  `HighlightStart`/`HighlightEnd` e o `SearchHit` descarta os dois. Com
<!-- wiki-refs: ignore match_offset -- campo que a Task 107 vai criar; hoje nao existe, e esse E o achado -->
  `match_offset`, o modelo faria `note_read(offset=match_offset-2000)` e o
  incidente estaria resolvido com código que já existe nos dois lados.

Junto, dois de contorno: **lote sem parâmetro por item** (`heading` vale para
todos os `paths` ou nenhum, então seis capítulos custam seis chamadas) e
**`snippet_chars` clampado em silêncio** — o modelo pediu 4000 e recebeu 1000,
sem aviso e sem o máximo declarado no schema.

## O que ele diz sobre o processo

O modelo tinha as duas ferramentas carregadas e **escolheu a do concorrente para
ler**. Depois de falhar duas vezes com o gobsidian, recomendou ao usuário voltar
para o outro caminho.

Nenhum gate do projeto mede isso. Há gate de órfãos, de lint, de rede, de
formatação, de referência em documentação — e **nenhum que pergunte se uma tool
resolve a tarefa que ela existe para resolver**. O que faltava era um teste
ponta a ponta sobre fixture realista: nota grande, convertida, com estrutura em
negrito, CRLF e sem heading ATX. As fixtures atuais são notas Obsidian
idiomáticas, e nelas todo o produto funciona. **A premissa "cofre Obsidian
idiomático" nunca foi escrita como premissa** — e por isso nunca foi
questionada.

## Correção deste incidente

Fase 1 do plano `docs/superpowers/plans/2026-08-16-revisao-fixes.md`, Tasks 106
<!-- wiki-refs: ignore next_offset match_offset note_outline -- nomes que as Tasks 106, 107 e 112 vao criar; a ausencia deles E o achado -->
a 112: `offset`/`next_offset`/`total_size`, `match_offset`, erro de heading em
três estados, `paths` por item, âncora escolhida por IDF e por casamento de
frase, adjacência de frase verificada, e a tool `note_outline`.
<!-- wiki-refs: ignore note_outline -- tool que a Task 112 vai criar; a ausencia dela E o achado -->

<!-- wiki-refs: ignore note_outline -- tool que a Task 112 vai criar -->
Decisão de produto tomada: **alternativa D** (`note_outline`, que não mente
sobre o documento) em vez da **E** (pseudo-heading no parser, que resolveria
mais e custaria reconstrução do cache de todo cofre). A E fica registrada como
pendente.

## Ver também

- [Achados em aberto](achados-abertos.md) — o resto da revisão.
- [Medições de 2026-08-15](medicoes-2026-08-15.md) — a parte medida.
- `docs/REVISAO-2026-08-15.md` — o relatório completo.
