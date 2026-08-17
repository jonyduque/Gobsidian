---
title: Regras não negociáveis
type: risk
status: active
description: O que derruba o produto em silêncio se for violado.
source_paths:
  - internal/mcpsrv
  - internal/vault
  - tools/netcheck
source_commit: b2be492
tags: [invariantes, contratos]
language: pt-BR
updated_at: '2026-08-16'
---

# Regras não negociáveis

Cada uma tem um modo de falha silencioso. Não são preferências de estilo.

## stdout pertence ao JSON-RPC

Todo log vai para **stderr**, via `log/slog`. Um `fmt.Println` em código
alcançável de `serve` corrompe a sessão, e o sintoma é o servidor **sumir do host
sem erro nenhum**.

`doctor`, `version`, `index`, `search` e `inspect` imprimem em stdout de
propósito — são comandos de CLI, não servidores. A distinção merece comentário
onde aparece.

## Nenhum pacote sob `internal/` ou `cmd/` importa `net/*`

Verificado por `tools/netcheck`, no CI. `net/http` e `x/oauth2` chegam
transitivamente pelo SDK — esperado; o check inspeciona *nossos* pacotes, não o
fecho transitivo.

Exceção estreita desde 2026-08-05 (RNF-30 reformulado): `net.Dial`/`net.Listen`
com a rede na **constante literal `"unix"`**. Rede vinda de variável é recusada —
escreva a string no lugar. Guardá-la numa variável deixa o `check_net` vermelho, e
**isso é a regra funcionando**.

## Nenhum tipo do SDK MCP cruza para fora de `internal/mcpsrv`

O protocolo já quebrou compatibilidade várias vezes. `internal/service` fala tipos
de domínio e não importa o SDK — é o que torna migração de protocolo mudança de um
pacote só.

## `ctx` onde há espera real

Função que pode **bloquear** recebe `ctx` e o respeita: I/O de arquivo,
varredura, worker pool, watcher, chamadas MCP. Leitura de variável de ambiente,
resolução de caminho e cálculo em memória **não** recebem.

`ctx` que nenhum corpo verifica ensina o revisor a ignorar `ctx`. Quando o
parâmetro existe só por consistência de assinatura, nomeie-o `_`.

Não há exceção. `lifecycle.Shutdown` recebe `ctx` e descarta o cancelamento com
`context.WithoutCancel`, porque o context raiz já está cancelado quando ela roda.

## Arquivo somente-nuvem nunca é aberto

Abrir um placeholder do OneDrive dispara **download síncrono**, e indexar o cofre
inteiro assim trava a máquina do usuário por minutos, sem dizer por quê.

A guarda mora em `vault.IsCloudOnly`, consultada por `index.classificar` e por
`search.Inverted.Update`. Ela **não** cobre o laço de boot do índice de busca —
ver [Achados em aberto](../notes/achados-abertos.md), achado crítico nº 2.

## Código de plataforma atrás de build tag

Nunca `if runtime.GOOS ==` dentro de lógica compartilhada. Em teste,
`runtime.GOOS` é aceitável para pular casos.

## Saída de console em ASCII puro

`[OK]`, `[*]`, `[!]`, `[i]`, `[...]`. Console PowerShell em CP-850 renderiza o
resto como lixo, e `doctor` é justamente o comando que alguém roda quando já está
confuso. `internal/console` decide sobre cor **pelo destino**, não por
`os.Stdout` global — senão `doctor > relatorio.txt` sai sujo.

## Sem `helpers.go`, `utils.go`, `common.go`

Arquivo assim é preocupação que ninguém nomeou.

## Nunca `go mod tidy`

Ver [Como rodar](../overview/como-rodar.md).

## Um `*Note` publicado é imutável

Quem muda uma nota troca a entrada do mapa por uma cópia (`mutarNotaLocked`). O
mutex protege o mapa; o ponteiro escapa por `Get` e por `List`, e o chamador lê
depois de soltar o `RLock`.

## Ver também

- [Armadilhas já pagas](armadilhas-pagas.md)
