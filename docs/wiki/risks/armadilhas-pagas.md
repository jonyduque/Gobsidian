---
title: Armadilhas já pagas
type: risk
status: active
description: Defeitos que passaram por revisão e só apareceram depois. Estão aqui para não voltarem.
source_paths:
  - internal/index/update.go
  - internal/index/alias.go
  - internal/search/inverted.go
  - internal/watcher/rename.go
  - cmd/gobsidian/serve.go
source_commit: b2be492
tags: [defeitos, licoes]
language: pt-BR
updated_at: '2026-08-16'
---

# Armadilhas já pagas

Cada uma custou caro. O padrão comum: **a forma menos usada é a que diverge**.

## Chave derivada em dois lugares

`byAlias` era escrito minúsculo por `alias.go` no boot e **cru** por `Replace`;
`resolve.go` lia minúsculo. Enquanto o índice só era construído no boot, os dois
concordavam. Quando o watcher tornou `Replace` e `Remove` alcançáveis, `Remove`
passou a procurar uma chave inexistente, e a entrada velha sobrevivia: `[[STJ]]`
continuava resolvendo, **com `state=ok`**, para uma nota já removida do índice.

Correção: toda chave derivada passa por **uma** função — `aliasKey`, `nomeChave`,
`classificar`. Não é para consertar os errados; é para tornar a próxima
divergência impossível sem tocar na função.

## Quem roda antes do guarda precisa do mesmo guarda

`CorrelateRenames` roda antes de `index.Replace` e chamava `vault.ReadAll` em todo
caminho do lote — anexo inclusive, placeholder de nuvem inclusive. `Replace`
respeita as duas regras; a camada anterior a ele não respeitava, então as regras
valiam para um caminho que nunca era alcançado.

## Reparar metade do estado é pior que não reparar

A reconciliação por overflow reparava o índice de metadados e deixava o de busca
obsoleto. Como `service.Search` descarta a posting cujo caminho não está nos
metadados, uma nota movida durante o overflow devolvia **zero resultados para
sempre**.

## Teste de fallback que deixa o caminho normal ligado mede o caminho normal

`TestOverflowReconciliationFull` injetava overflow com o watcher ativo; os eventos
comuns aplicavam as mudanças e a reconciliação nunca era exercida. Removido o
reconciliador inteiro, o teste passava em 2,8 s — **cobertura zero num requisito
P0**, através de uma revisão que o aprovou.

## Watch em diretório novo não vê o que já está dentro

Uma pasta que chega ao cofre com arquivos entrega **um** evento e nenhum arquivo.
Medido: 3 notas, 1 evento, 0 indexadas, invisíveis até o próximo reinício. Era
também o que fazia `note_move` perder a nota. Todo `Add` de watch precisa ser
seguido de varredura.

## `io.TeeReader` não propaga EOF

Copia bytes, e EOF não é byte. Sem `mirrorReader`, o monitor de stdin fica inerte
e `lc.Wait()` só retorna por acidente.

## Índice reconstruído e recarregado têm de responder igual

`DocLength` era derivado na leitura somando as posições de cada termo — e um token
cuja forma reduzida difere da raiz entra em **duas** postings. Um documento de 5
tokens que todos reduzem media 5 recém-construído e 10 recarregado. `DocLength` é
o divisor da normalização do BM25: o mesmo cofre ranqueava diferente conforme o
servidor tivesse acabado de indexar ou de ler o cache.

O que prova isso não é conferir um valor escrito à mão — é **comparar as duas
construções campo a campo**.

## Trocar um crash por violação de regra não é conserto

O fix do pânico de `index.Build` com placeholder de nuvem tornou o boot possível —
e, com ele, tornou alcançável um caminho que baixa todo placeholder do cofre.
Antes, o processo morria antes de chegar lá.

**Depois de remover um `panic` ou um `return` que abortava cedo, a pergunta é: o
que passa a rodar agora que antes não rodava?** A resposta se acha percorrendo os
chamadores a jusante, não relendo a função consertada.

## Feature P1 não apaga dado P0

O campo inline do Dataview consumia o span do valor, e `fonte:: [[STJ]]` deixava
de produzir link nenhum — links que o commit anterior já coletava.

## Cache ligado no harness que mede latência muda a pergunta

Os laços repetem a MESMA consulta, então um cache de trecho ligado acerta quase
todas e o p95 passa a descrever a consulta repetida, que nenhum usuário vê na
primeira busca. Todo harness de latência desliga o cache; quem mede repetição é um
benchmark com "Repetido" no nome.

## Armadilhas de ferramenta

- **Pipe engole código de saída.** `cmd | tail` devolve o status do `tail`.
- **`str.replace` que não casa não falha** — segue em silêncio. Toda edição por
  script leva `assert` do texto-âncora antes e conferência depois.
- **`-update` de golden grava o que o código produz, não o que está certo.**
- **Dois agentes na mesma worktree colidem.** Matar sempre por PID que você mesmo
  lançou, nunca por nome — um `Stop-Process -Name gobsidian` já matou a sessão
  real do usuário.

## Ver também

- [Regras não negociáveis](regras-nao-negociaveis.md)
- `CLAUDE.md` — a lista completa, com o detalhe de cada caso.
