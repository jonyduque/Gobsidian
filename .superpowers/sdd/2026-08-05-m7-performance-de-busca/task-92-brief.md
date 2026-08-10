# Task 92 — Daemon: uma instância por cofre, com ciclo de vida próprio

**Tier: modelo forte.** É onde o ganho aparece e onde mora o risco: um processo
de longa vida segurando o cofre, sem pai para vigiar.

#### Onde encaixa
Depois da 91. Última das de código desta parte.

#### O que vincula esta tarefa
D-M7-6 (AF_UNIX, já decidido por medição) e a garantia reformulada da Task 90.

Mais uma que é específica desta tarefa: **a vigília do pai não se aplica ao
daemon.** Ele não tem pai que o defina — quem o inicia é uma ponte que sai logo
depois. Os três mecanismos de encerramento do servidor stdio não cobrem este
caso, e o gate de órfãos, como está, não o testa.

Vale também: **não escreva número que você não mediu**, e **escopo não encolhe
em silêncio**.

#### A decisão que esta tarefa tem de acertar
1. **Um daemon por cofre**, chaveado pelo mesmo hash de caminho que já nomeia o
   diretório de cache. Dois cofres, dois daemons.
2. **Corrida de inicialização resolvida por arquivo de bloqueio**, não por
   "tenta conectar, senão inicia": dez pontes subindo juntas iniciariam dez
   daemons. Quem perde a corrida espera e conecta.
3. **Encerra por ociosidade.** Sem cliente conectado por N minutos, sai. Sem
   isso, o daemon é 382 MB permanentes, e a economia vira desperdício numa
   máquina que usou a ferramenta uma vez. **Padrão: 15 minutos**, configurável.
4. **Versão no handshake.** Ponte e daemon de versões diferentes não conversam;
   a ponte cai para o modo em processo da Task 91 e registra no log.
5. **O daemon não fala JSON-RPC pelo stdout**, e sim pelo socket — mas o log
   continua em stderr, e ele precisa de um destino de log próprio, porque não
   tem terminal.

#### Armadilhas já pagas que se aplicam
- **Vigília do pai precisa de `exitTime`, não só creation time.** Se esta tarefa
  fizer o daemon vigiar qualquer processo, a lição vale inteira: no Windows, PID
  e creation time seguem consultáveis depois da morte, e comparar só os dois
  nunca detecta pai morto. Já deixou 5 de 5 órfãos aqui.
- **Reparar metade do estado é pior que não reparar.** O daemon serve vários
  clientes; um cofre reconciliado pela metade agora afeta todos eles.
- **Teste de mecanismo que cruza estruturas afirma sobre o que o usuário
  veria**, não sobre cada estrutura em separado.

#### Verificações além dos passos
- **RSS agregado de três sessões** contra três processos independentes. É o
  número que a Parte II inteira existe para mover.
- Daemon morto no meio de uma chamada: a ponte devolve erro acionável, não trava.
- Dez pontes subindo simultaneamente: **um** daemon, não dez.
- Ociosidade: sem cliente, sai dentro do prazo.
- **Cenário novo no gate de órfãos**: matar todas as pontes deve deixar o daemon
  saindo por ociosidade, e o harness tem de conferir isso — hoje ele não cobre
  processo sem pai.

#### Regras de execução
`verify.ps1`, o gate de órfãos com o cenário novo, e o teste das dez pontes
antes de dizer que acabou. Ledger antes de reportar.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/daemon/daemon.go `
  -Anchor 'if time.Since(ultimoCliente) > cfg.OciosidadeMax {' -Replacement 'if false {' `
  -Test TestDaemonSaiPorOciosidade -Package ./internal/daemon/

pwsh -File scripts/mutate.ps1 -Path internal/daemon/lock.go `
  -Anchor 'if !adquiriu {' -Replacement 'if false {' `
  -Test TestDezPontesIniciamUmDaemonSo -Package ./internal/daemon/
```

**As âncoras nomeiam código que ainda não existe, e isso é deliberado: elas são
o contrato de nomes desta tarefa.**

#### Contrato de relatório
RSS de três sessões, antes e depois, medido. Resultado dos quatro cenários de
verificação. Saída das duas provas de mutação. Se o ganho agregado for menor que
o custo de complexidade, **dizer isso** — a tarefa pode terminar em "medido, não
compensa", e isso é resultado, não falha.

**Files:** `internal/daemon/daemon.go`, `internal/daemon/lock.go`,
`cmd/gobsidian/`, `scripts/test_orphans.ps1`, testes
**Commit:** `feat(daemon): one shared process per vault, with idle exit`

---

