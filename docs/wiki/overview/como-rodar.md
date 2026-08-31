---
title: Como rodar
type: overview
status: active
description: Subcomandos, flags, configuração e a bateria de verificação.
source_paths:
  - cmd/gobsidian/main.go
  - cmd/gobsidian/serve.go
  - internal/config/config.go
  - internal/config/defaults.go
  - scripts/verify.ps1
source_commit: c6804e1e
tags: [cli, config]
language: pt-BR
updated_at: '2026-08-31'
---

# Como rodar

## Subcomandos

Registrados em `cmd/gobsidian/main.go:59`.

| Comando | Para quê |
|---|---|
| `serve` | O servidor MCP sobre stdio. É o que o host chama. |
| `daemon` | Processo de vida longa que serve N sessões por socket. |
| `doctor` | Diagnóstico de ambiente. Sai ≠ 0 quando o cofre está inacessível. |
| `version` | Versão, injetada pelo linker. |
| `index` | Constrói o índice e reporta, sem servir. |
| `search` | Busca pela linha de comando. |
| `inspect <nota>` | Despeja o que o índice sabe de uma nota. |

**`doctor`, `version`, `index`, `search` e `inspect` imprimem em stdout de
propósito** — são comandos de CLI, não servidores. Só `serve` e `daemon` têm o
stdout reservado ao JSON-RPC.

`--max-results` na ponte deixou de ser no-op silencioso no modo daemon em
2026-08-28: a saudação do IPC passou a carregá-lo, e o protocolo virou **2** por
causa disso. Antes, a flag existia, o usuário a passava, e o daemon servia o
valor dele.

## Configuração

`internal/config` é o **único** lugar que decide precedência:
**flag > variável de ambiente > default**. Abaixo dessa camada ninguém volta a
olhar `os.Getenv`.

| Flag | Env | Default |
|---|---|---|
| `--vault` | — | **obrigatório** |
| `--log-level` | `GOBSIDIAN_LOG_LEVEL` | `info` |
| `--read-only` | `GOBSIDIAN_READ_ONLY` | `false` |
| `--debounce-ms` | `GOBSIDIAN_DEBOUNCE_MS` | `250` (zero é **recusado**) |
| `--cache-dir` | — | derivado do hash do cofre |
| `--eager-search` | — | `false` (carga preguiçosa) |

`GOBSIDIAN_NO_DAEMON` com qualquer valor não vazio pula a ponte e serve em
processo.

### A armadilha do valor zero

Flag booleana ou inteira **não distingue "omitida" de "definida com zero"**. Por
isso `config.Flags` tem os companheiros `ReadOnlySet` e `DebounceMSSet`,
preenchidos com `cmd.Flags().Changed(nome)`. Esquecer isso num subcomando novo
faz a flag virar no-op silencioso.

O mesmo motivo faz `service.Options.SnippetCacheEntries` ser `*int`: `nil` usa o
padrão, zero explícito **desliga** o cache.

## Bateria de verificação

```bash
pwsh -File scripts/verify.ps1        # build, go test -race, vet nos 3 alvos, gofmt, check_net
pwsh -File scripts/build.ps1
pwsh -File scripts/test_orphans.ps1 -Cycles 100   # os quatro cenários
golangci-lint run ./...              # exige v2.12.2
```

`verify.ps1` existe porque a lista solta convida a rodar três dos cinco. Aceita
`-SkipCross` e `-SkipNet` para iteração rápida; o gate roda tudo.

**`golangci-lint` local verde não significa CI verde.** Um binário compilado com
Go mais antigo que o `go` directive recusa o config antes de analisar linha
nenhuma. Confira `golangci-lint version` antes de confiar num zero.

Detalhe de cada gate em [Gates e scripts](../reference/gates-e-scripts.md).

## Dependências

**Nunca rode `go mod tidy`.** Várias dependências estão fixadas sem importador
ainda; `tidy` removeria elas e o pin do SDK MCP, que é decisão fechada.

Quando um pacote importa uma dependência pela primeira vez e faltam entradas
transitivas em `go.sum`, o comando certo é
`go get <caminho-do-pacote>@<versão-fixada>` — caminho do **pacote**, não do
módulo.

## Ver também

- [Onde ficam os dados](onde-ficam-os-dados.md)
- [Fluxo de boot](../flows/boot.md)
