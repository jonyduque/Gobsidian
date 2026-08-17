---
title: Escrita
type: feature
status: active
description: As cinco tools de escrita, gravação atômica, travas por caminho e preservação de EOL/BOM.
source_paths:
  - internal/service/write.go
  - internal/writer/atomic.go
  - internal/writer/lock.go
  - internal/writer/diff.go
  - internal/writer/section.go
  - internal/writer/linkrewrite.go
source_commit: b2be492
tags: [escrita, atomicidade]
language: pt-BR
updated_at: '2026-08-16'
---

# Escrita

Cinco tools: `note_create`, `note_append`, `note_patch`, `note_move`,
`note_delete`. Todas passam por `internal/writer` para tocar o disco.

## Gravação atômica

`writer.WriteAtomic` faz temporário no mesmo diretório → `Write` → `Sync` →
`Close` → `os.Rename`, com retentativa no rename (10 × 10 ms) porque no Windows
um antivírus ou o próprio Obsidian pode segurar o arquivo por um instante.

O temporário desta escrita é removido no `defer` em qualquer falha; o de um
processo morto é removido **no boot** por `SweepStaleTempFiles`.

> `CleanStaleTempFiles` **não pode** ser chamada no início de uma escrita. O glob
> apaga todos os temporários do diretório, inclusive o de outra escrita em voo. A
> trava é por *caminho*; o recurso compartilhado ali é o *diretório*.

## Travas por caminho

`writer.PathLocker` dá um mutex por caminho, com contagem de referência e chave
em minúsculas (casing uniforme no Windows). Duas notas na mesma pasta escrevem em
paralelo; a mesma nota, não.

## Preservação de forma

O cofre é do usuário e costuma estar sob Git. Duas coisas são preservadas:

- **EOL** — `writer.DetectEOL` lê do arquivo. Escrever LF numa nota CRLF produz
  diff de arquivo inteiro (RF-38).
- **BOM** — detectado na leitura e considerado no cálculo de offsets.

## Recortes

`note_append` e `note_patch` operam por **heading** ou por **bloco** (`^id`), com
os offsets que o índice já guarda. `FindHeading` e `FindBlock` distinguem "não
achei" de "achei mais de um" — ambiguidade devolve erro, nunca escolhe por conta
própria.

`note_patch` tem três modos: `replace_section` (preserva o título, troca o
conteúdo abaixo), `replace_heading_and_section` (troca a partir do título) e
`replace_block`.

## `note_move` e a reescrita de links

Com `update_links`, `note_move` percorre os backlinks da nota, monta as
substituições por nota citante e reescreve cada arquivo — em ordem alfabética,
para o resultado não depender da ordem de iteração de mapa.

O alvo novo respeita a forma do link original: wikilink que citava só o nome
continua citando só o nome; wikilink com caminho recebe o caminho novo.

## `dry_run`

Toda tool de escrita aceita `dry_run`, que devolve o **diff unificado** e não
toca no disco. É a via segura para o modelo propor uma edição e o usuário
conferir antes.

## Concorrência com o disco

`expected_hash` existe para o controle otimista: o cliente manda o hash que leu, e
a escrita recusa se não bater. Ver o item aberto sobre a **fonte** desse hash em
[Achados em aberto](../notes/achados-abertos.md).

## Superfície não confinada

A checagem de caminho das tools de escrita **não passa por `vault.Resolve`** —
`checkWriteAllowed` é uma verificação própria, e `CanonicalPath` é construída por
conversão. É o achado crítico nº 1 da revisão. Ver
[Regras não negociáveis](../risks/regras-nao-negociaveis.md) e
[Achados em aberto](../notes/achados-abertos.md).

## Ver também

- [Onde ficam os dados](../overview/onde-ficam-os-dados.md)
- [Nota e caminho](../entities/note-e-caminho.md)
