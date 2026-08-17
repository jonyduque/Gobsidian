---
title: Daemon e ponte
type: feature
status: active
description: Como N sessões do host compartilham um índice — socket AF_UNIX, handshake e ociosidade.
source_paths:
  - cmd/gobsidian/ponte.go
  - cmd/gobsidian/daemon.go
  - internal/daemon/daemon.go
  - internal/daemon/lock.go
  - internal/ipc/ipc.go
source_commit: b2be492
tags: [daemon, ipc, memoria]
language: pt-BR
updated_at: '2026-08-16'
---

# Daemon e ponte

O problema: cada sessão do host levantava um servidor completo, com seu próprio
índice. Cinco sessões abertas custavam cinco índices.

A solução tem duas peças.

**A ponte** é o processo que o host lança. Ela é **burra**: quando conectada a um
daemon, só copia bytes entre o stdio que recebeu e o socket. Não interpreta
JSON-RPC, não tem índice, não tem estado — é o que a mantém em poucos MB.

**O daemon** é um processo de vida longa, **um por cofre**, que serve N conexões
contra o mesmo `*mcpsrv.Server`. O índice, o watcher e o serviço são
compartilhados.

Medido no cofre real de 4.513 notas, memória física agregada:

| sessões | pré-M7 | sem daemon | com daemon |
|---|---|---|---|
| 1 | 579,1 MB | 244,6 MB | 223,6 MB |
| 3 | 1.681,3 MB | 508,5 MB | 262,2 MB |
| 5 | 2.916,4 MB | 773,4 MB | 229,4 MB |

A coluna do daemon **não escala com N** — é a assinatura de um índice pago uma
vez só.

## A decisão de transporte

AF_UNIX nos três sistemas, sem compilação condicional para o socket em si. Medido
contra named pipe, ida e volta, 20.000 repetições: 25,7 µs contra 82,9 µs em
256 B. Está na biblioteca padrão e é o mesmo código nas três plataformas; build
tag só para o caminho do socket e a limpeza.
Ver [Decisões fechadas](../decisions/decisoes-fechadas.md).

## Handshake

Uma linha, e só ela é interpretada — depois disso a conexão é byte cru:

```
GOBSIDIAN-IPC 1 ro=0 vault=<VaultKey>
```

Confere **versão do protocolo** e **configuração**. Versões diferentes não se
falam: a ponte cai para o modo em processo em vez de arriscar um protocolo que os
dois lados entendem diferente.

A conferência de `ro` não é detalhe: uma ponte iniciada com `--read-only`
recebendo uma sessão que escreve no cofre é bug de segurança. A resposta certa
não é negociar — é recusar o daemon e cair para o modo em processo, que só pode
servir a configuração de quem o está rodando.

## Fallback obrigatório

Em **qualquer** dos três pontos onde pode falhar — socket ausente, `EnsureStarted`
incapaz de iniciar, handshake falhando depois de iniciar — a ponte serve em
processo. Sem isso, um daemon quebrado transformaria a ferramenta em nada. O log
sempre registra o motivo da queda.

`GOBSIDIAN_NO_DAEMON` pula toda essa decisão.

## Encerramento por ociosidade

O daemon **não tem pai vigiável nem stdin de host** — quem o lançou foi uma ponte
que saiu logo depois. Os dois mecanismos correspondentes de `internal/lifecycle`
não se aplicam, e ligar a vigília do pai "por consistência" o derrubaria na hora.

Quem os substitui é a ociosidade: sem nenhum cliente por `OciosidadeMax` (padrão
15 min), o daemon chama `lifecycle.Trigger("idle")` — a **mesma** infraestrutura
de cancelamento e log que sinal usa, para o gate de órfãos conferir `reason=` do
mesmo jeito nos quatro mecanismos.

## A corrida residual conhecida

O lock de inicialização serializa quem disputa no mesmo instante, não quem chega
atrasado. Medido: dez pontes sob carga produziram **dois daemons vivos** para o
mesmo cofre. O segundo dial depois de adquirir o lock reduziu a janela a
milissegundos, mas não é exclusão mútua por construção.

Registrado nos limites conhecidos de `docs/OPERACAO.md`. A revisão aponta a causa
estrutural em [Achados em aberto](../notes/achados-abertos.md).

## Ver também

- [Encerramento](../flows/encerramento.md)
