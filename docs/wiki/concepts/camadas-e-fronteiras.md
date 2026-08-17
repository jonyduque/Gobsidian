---
title: Camadas e fronteiras
type: concept
status: active
description: Quem pode importar o quê, e por que cada fronteira existe.
source_paths:
  - internal/vault
  - internal/index
  - internal/search
  - internal/service
  - internal/mcpsrv
  - cmd/gobsidian
source_commit: b2be492
tags: [arquitetura, fronteiras]
language: pt-BR
updated_at: '2026-08-16'
---

# Camadas e fronteiras

O produto é uma pilha estreita. Cada camada só conhece a de baixo, e três
fronteiras são impostas por ferramenta, não por disciplina.

```
cmd/gobsidian      CLI, ciclo de vida, ponte/daemon
    ↓
internal/mcpsrv    ÚNICO lugar que fala tipos do SDK MCP
    ↓
internal/service   fachada: uma função por tool, tipos de domínio
    ↓
internal/index  ·  internal/search  ·  internal/writer  ·  internal/watcher
    ↓
internal/vault     ÚNICA camada que toca o sistema de arquivos do cofre
```

## As três fronteiras impostas

**`internal/vault` é a única que toca o disco do cofre.** Todo caminho que entra
passa por confinamento antes de virar chamada de sistema. O tipo
`vault.CanonicalPath` existe para ser a *prova* de que o confinamento rodou —
construí-lo por conversão fora do pacote forja essa prova.
Ver [Nota e caminho](../entities/note-e-caminho.md).

**Nenhum tipo do SDK MCP cruza para fora de `internal/mcpsrv`.** O protocolo já
quebrou compatibilidade várias vezes; concentrar o contato num pacote faz de uma
quebra de API um problema de um arquivo. `internal/service` fala tipos de
domínio e não importa o SDK.

**Nenhum pacote sob `internal/` ou `cmd/` importa `net/*`.** A garantia é
verificada por um analisador próprio, `tools/netcheck`, rodado no CI. `net/http`
e `x/oauth2` chegam transitivamente pelo SDK — isso é esperado, porque o check
inspeciona *os nossos* pacotes, não o fecho transitivo.

Desde 2026-08-05 a regra admite uma exceção estreita: `net.Dial`/`net.Listen`
com a rede como **constante literal `"unix"`**, para o IPC local do daemon.
Rede vinda de variável é recusada. Ver [Decisões fechadas](../decisions/decisoes-fechadas.md).

## Regras de forma

**Código de plataforma atrás de build tag, em arquivo separado.** Nunca
`if runtime.GOOS ==` dentro de lógica compartilhada. O padrão aparece em
`vault/cloud_windows.go` × `cloud_other.go`, `lifecycle/parent_windows.go` ×
`parent_unix.go`, `search/mmap_windows.go` × `mmap_unix.go`.

**`ctx` onde há espera real.** Função que pode *bloquear* recebe `ctx` e o
respeita: I/O de arquivo, varredura, worker pool, watcher, chamadas MCP. Leitura
de variável de ambiente, resolução de caminho e cálculo em memória não recebem.
Quando o parâmetro existe só por consistência de assinatura, nomeie-o `_`.

**Sem `helpers.go`, `utils.go`, `common.go`.** Arquivo assim é preocupação que
ninguém nomeou. O pacote `internal/console` nasceu como `internal/utils` e foi
renomeado por isso.

## Onde a fronteira vaza hoje

Duas, registradas em [Achados em aberto](../notes/achados-abertos.md):

- `service.Index` é uma interface de 12 métodos que devolve `*index.Note`,
  `index.Query` e `index.TagCount`. Ela não isola nada — `service` não compila
  sem `index`.
- A superfície de escrita em `internal/service/write.go` constrói
  `vault.CanonicalPath` por conversão, sem passar por `vault.Resolve`.

## Ver também

- [Os dois índices](os-dois-indices.md)
- [As 12 tools MCP](../features/tools-mcp.md) — a superfície que `mcpsrv` expõe.
- [Watcher](../features/watcher.md) — a fronteira do fsnotify.
- [Regras não negociáveis](../risks/regras-nao-negociaveis.md)
- `docs/ARCHITECTURE.md` — a especificação, com os AD-01 a AD-09.
