---
title: Nota, anexo e caminho canônico
type: entity
status: active
description: Os tipos centrais do domínio e as regras de confinamento de caminho.
source_paths:
  - internal/index/note.go
  - internal/vault/path.go
  - internal/vault/walk.go
  - internal/parser/types.go
source_commit: c6804e1e
tags: [modelo, caminho, seguranca]
language: pt-BR
updated_at: '2026-08-31'
---

# Nota, anexo e caminho canônico

## `vault.CanonicalPath`

A única forma interna de identificar uma nota: **relativa à raiz do cofre, com
separador `/`, sem `./` inicial, e com a grafia exata do disco** — maiúsculas e
minúsculas inclusive.

A grafia exata importa: cofres reais acumulam inconsistência de casing entre
pastas, e o índice precisa refletir o disco, não uma normalização inventada.

> **Esta camada não consulta o disco e portanto NÃO garante a grafia dele.** Ela
> preserva o que o chamador passou. Quem produz a grafia real é `vault.Walk`, que
> lê as entradas de diretório.

### A chave do índice é normalizada; o caminho guardado, não

A distinção é o ponto inteiro: `CanonicalPath` continua sendo a grafia do disco,
porque o servidor abre o arquivo por ele — normalizá-lo faria o produto tentar
abrir um arquivo que não existe. As **chaves derivadas** dos mapas do índice
passam por `chaveDeCaminho`, que aplica **NFC** além da caixa.

O defeito que isso fecha: `í` precomposto (U+00ED, NFC) e `i` + acento
combinante (U+0069 U+0301, NFD) são a mesma letra para quem lê e **strings
diferentes** para um mapa de Go. Um cofre sincronizado com macOS grava NFD e um
cliente Windows pede NFC, e até 2026-08-31 `ResolvePath` devolvia
`ErrPathNotFound` para uma nota que existe. Este é um cofre em português, onde
acento é a regra.

NFC, e não NFD, porque é o que a maioria dos clientes envia e o que o Go emite
por padrão. E a normalização **não remove acento** — `text.Normalize` faz isso,
mas ela existe para BUSCA, onde "Capitulo" tem de casar com "Capítulo". Aplicada
a chave de índice, faria duas notas distintas colidirem numa entrada só.

As quatro derivações — `lowerPath`, `byName`, `byAlias` e a resolução por nome —
moram em `internal/index/chave.go`, e **todas** passam por lá, inclusive as que
já estavam certas. Não é para consertar as erradas: é para tornar a próxima
divergência impossível sem tocar na função.

## Confinamento em duas camadas

`vault.Resolve(root, input)` é o portão. Duas verificações, e as duas são
necessárias:

**`validateLocal`** (léxica) rejeita byte nulo, `..` inicial, caminho enraizado em
qualquer forma, e o que o sistema operacional resolve para fora sem nunca produzir
um `..`: os **nomes de dispositivo** do Windows. `filepath.IsLocal` é quem os
barra — antes disso, `Resolve(root, "COM1")` escrevia em porta serial.

A regra de ponto ou espaço no fim de componente vale **só no Windows**: em Linux
`Notas ` é nome legal, e rejeitá-lo lá torna notas reais inalcançáveis.

**`Canonicalize`** (por componente) verifica via `filepath.Rel` e reaplica
`validateLocal` na saída — a forma canônica precisa ser estável sob ida e volta.

**Limite conhecido:** a verificação é puramente léxica. Não resolve links
simbólicos nem junctions.

## `index.Note`

Metadados e **offsets**, sem o corpo. Ler o corpo custa uma ida ao disco, de
propósito.

| Campo | Nota |
|---|---|
| `Path`, `Title`, `TitleNorm` | `TitleNorm` é pré-computado, para o BM25 não normalizar por consulta |
| `Size`, `ModTime` | usados por `VerifyFreshness` e pelo watcher |
| `Hash` | xxhash do conteúdo **bruto**, com frontmatter e BOM. É o exposto e o aceito em `expected_hash` |
| `EOL`, `BOM` | preservação de forma na escrita |
| `CloudOnly` | placeholder do OneDrive: indexado por metadados de diretório, sem leitura |
| `Frontmatter`, `Tags`, `Aliases` | do YAML |
| `Headings`, `Blocks`, `Links`, `Inline` | do corpo, com offsets |

## `index.Asset`

Anexo. **Não tem hash nem conteúdo**: é indexado por nome e nunca aberto — abri-lo
dispararia download de arquivo somente-nuvem, e não abri-lo ainda impede que todo
embed de imagem seja contado como link quebrado.

## Estados de link

`ResolvedLink` é o `parser.Link` mais o resultado da resolução, que depende do
cofre inteiro e por isso não pode ser feita no parser.

| Estado | Significa |
|---|---|
| `LinkOK` | resolveu para nota ou anexo |
| `LinkTargetMissing` | deveria resolver e não resolveu — é o que conta como quebrado |
| `LinkAnchorMissing` | a nota existe; a âncora depois do `#`, não |
| `LinkExternal` | tem esquema de URI, nunca foi para o cofre |

`LinkExternal` existe porque sem ele toda URL numa nota entrava como link
quebrado, e a contagem — que o PRD chama de principal sinal de saúde do cofre —
afogava em falso positivo. Confirmado contra o `metadataCache` real: o Obsidian
não registra URL externa nem em `resolvedLinks` nem em `unresolvedLinks`.

`ResolveVia` registra **qual regra** resolveu (`path`, `name`, `asset`, `alias`).
Duas notas com o mesmo nome em pastas diferentes tornam isso útil para
diagnosticar um alvo inesperado.

## Ver também

- [Escrita](../features/escrita.md) · [Camadas e fronteiras](../concepts/camadas-e-fronteiras.md)
