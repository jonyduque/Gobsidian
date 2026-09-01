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
source_commit: f7de8e81
tags: [daemon, ipc, memoria]
language: pt-BR
updated_at: '2026-08-31'
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
GOBSIDIAN-IPC 2 ro=0 vault=<VaultKey> max_results=100
```

Confere **versão do protocolo** e **configuração**. Versões diferentes não se
falam: a ponte cai para o modo em processo em vez de arriscar um protocolo que os
dois lados entendem diferente.

O protocolo virou **2** em 2026-08-28 (achado M9): a saudação passou a carregar
`max_results`. Até ali, `--max-results` na ponte era **no-op silencioso** no modo
daemon — a flag existia, o usuário a passava, e o daemon servia o valor dele. Um
campo de configuração que não atravessa o handshake é pior que campo ausente,
porque ninguém percebe.

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

## Encerramento da conexão: meio-fechamento antes do adeus

Quando a sessão do host acaba, a ponte não fecha o socket seco. Ela faz
`CloseWrite()` — meio-fechamento — e **drena** o que o daemon ainda estiver
mandando, com orçamento de 2 s, antes de encerrar.

Sem isso, a última resposta em voo morre no fechamento e o host vê uma chamada
sem retorno. `CloseWrite` estava declarado na interface e conferido no handshake
desde sempre, e **nunca era chamado** (achado M8): interface cumprida, contrato
não.

## Encerramento por ociosidade

O daemon **não tem pai vigiável nem stdin de host** — quem o lançou foi uma ponte
que saiu logo depois. Os dois mecanismos correspondentes de `internal/lifecycle`
não se aplicam, e ligar a vigília do pai "por consistência" o derrubaria na hora.

Quem os substitui é a ociosidade: sem nenhum cliente por `OciosidadeMax` (padrão
15 min), o daemon chama `lifecycle.Trigger("idle")` — a **mesma** infraestrutura
de cancelamento e log que sinal usa, para o gate de órfãos conferir `reason=` do
mesmo jeito nos quatro mecanismos.

## A posse é uma trava do kernel

`flock` no Unix, `LockFileEx` no Windows, desde 2026-08-31. Até ali era um
arquivo criado com `O_CREATE|O_EXCL` guardando o PID do dono, e quem o achasse
ocupado lia o PID, perguntava se aquele processo vivia e, se não, **removia o
lock e tomava o lugar**.

Esse esquema reprovava **55% das vezes** num teste de dez daemons concorrentes em
Linux, com até quatro dentro da região crítica. Duas corridas nele foram achadas
e corrigidas; sobrou uma terceira que três rodadas de instrumentação não
nomearam — e o ponto é que cada remendo comprava uma corrida a menos e nenhuma
garantia. O Windows mascarava tudo por acidente: lá `os.Remove` de arquivo com
handle aberto falha.

Com a trava do kernel **não existe lock obsoleto**: o sistema operacional a solta
quando o dono morre. Sem lock obsoleto não há recuperação, sem recuperação não há
PID a parsear, e a classe inteira de corrida some — junto com `lockObsoleto`,
`pidVivo` e ~120 linhas. Medido depois: **0 de 40**.

**O arquivo nunca é removido**, e é isso que fecha a corrida. Remover era a
origem dela — e sob `flock` é pior: quem detém continua com a trava sobre o inode
desvinculado enquanto o próximo trava um inode novo no mesmo caminho, dando dois
donos. O que sobra é um arquivo de poucos bytes com o PID dentro, **só para
diagnóstico**.

## Dois locks, e por que não podem ser um

`EnsureStarted` tem um lock; **escutar tem outro**, em `<sock>.listen.lock`.

O de escuta fechou uma janela que o primeiro não alcança: `ipc.Listen` prova que
o socket está órfão antes de desvinculá-lo, mas **a sonda e o bind não são
atômicos entre si**. Dois daemons lançados no mesmo instante podem ambos sondar
"ninguém escuta" antes de qualquer um bindar — e aí os dois desvinculam e bindam,
duas instâncias gravando no mesmo cache de busca.

Eles não podem ser o mesmo arquivo: o daemon adquire o de escuta **enquanto** a
ponte que o lançou ainda segura o de `EnsureStarted`. Compartilhar o arquivo
trava os dois.

## A corrida residual conhecida

O lock de inicialização serializa quem disputa no mesmo instante, não quem chega
atrasado. Medido: dez pontes sob carga produziram **dois daemons vivos** para o
mesmo cofre. O segundo dial depois de adquirir o lock reduziu a janela a
milissegundos, e o lock de escuta fechou a metade sonda-e-bind — mas o caso sob
carga não é exclusão mútua por construção.

Registrado nos limites conhecidos de `docs/OPERACAO.md`.

## Ver também

- [Encerramento](../flows/encerramento.md)
