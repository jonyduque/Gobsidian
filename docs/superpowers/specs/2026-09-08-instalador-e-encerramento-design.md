# Instalador no próprio binário, e encerramento que não pendura — Design

**Data:** 2026-09-08
**Estado:** aprovado pelo dono em 2026-09-08
**Origem:** investigação do `Server transport closed unexpectedly` relatado em
2026-09-08, cujos achados estão na seção "O que foi medido".

---

## 1. Por que este documento existe

Uma sessão MCP caiu. A investigação achou quatro coisas, e só a primeira era a
que o dono tinha visto:

1. A desconexão veio do **instalador**, que mata processos `gobsidian` para
   poder substituir o binário (`docs/OPERACAO.md:1970`). Comportamento
   desenhado.
2. Um daemon **pendurou no encerramento** e ficou vivo 20 h.
3. O daemon seguinte **não conseguiu abrir o socket**, 20+ vezes desde
   2026-09-01, e toda ponte caiu para o modo em processo.
4. Duas instâncias passaram a **gravar o mesmo cache de busca**, que é
   exatamente o estado que `internal/ipc.Listen` existe para impedir.

Os itens 2 a 4 encadeiam. O item 1 encadeia com o resto porque a forma atual de
instalar — 1.738 linhas da mesma lógica escritas duas vezes, em PowerShell e em
Node, sem um único teste — é o que dispara o ciclo.

Este documento fecha as duas frentes de uma vez: **o instalador passa a ser o
próprio binário**, e **nenhuma espera de encerramento fica sem orçamento**.

---

## 2. O que foi medido

Nenhum número aqui é estimativa. Cada um saiu de um comando rodado em
2026-09-08 na máquina do dono (Windows 11 Home Single Language 10.0.26220,
Go 1.26.5).

### 2.1 O daemon pendurado

`C:\Users\jonyd\AppData\Local\gobsidian\run\db03d9f55cea7459.sock.log`:

```
2026-09-07T17:45:30  daemon iniciado
2026-09-07T17:45:36  servidor pronto (3372 notas)
2026-09-07T19:30:36  encerramento solicitado reason=idle
                     <nenhuma linha "daemon encerrado">
```

PID 42856 seguia vivo em 2026-09-08 às 16h: 274 MB residentes, **0 s de CPU em
3 s de amostragem**, 26 threads todas em `Wait,UserRequest`. Todo encerramento
anterior no mesmo arquivo tem o par `encerramento solicitado` /
`daemon encerrado`. Esse não tem.

`lifecycle.Shutdown` tem guarda de `os.Exit(1)` em 6 s
(`internal/lifecycle/shutdown.go:48`). O processo estar vivo prova que o
travamento **não está dentro dela** — está numa das três esperas sem orçamento
nenhum:

| Espera | Onde |
|---|---|
| `wg.Wait()` | `internal/daemon/daemon.go:105` |
| `lc.Wait()` | `cmd/gobsidian/daemon.go:180` |
| `c.Esperar()` | `cmd/gobsidian/daemon.go:181` |

**Qual das três, não foi determinado.** Delve não fecha a conta: o binário é
compilado com `-s -w` (`scripts/build.ps1`, `release.yml`) e `dlv attach`
responde `could not find goroutine array`. A decisão do dono foi partir para o
anteparo em vez de continuar caçando (decisão D-12).

### 2.2 O socket que não some

`ipc.cleanupSocketFile` (`internal/ipc/ipc.go:380`) é `os.Remove` puro. Falhou
20+ vezes desde 2026-09-01 com `The file cannot be accessed by the system`
(ERROR_CANT_ACCESS_FILE, 1920), e cada falha derrubou o daemon para o modo em
processo.

Dois cenários foram reproduzidos, e **nenhum dos dois** produz o par observado
em produção:

| Cenário | `net.Dial` | `os.Remove` |
|---|---|---|
| listener fechado limpo | 10061 | arquivo já não existe — `Close` desvincula no Windows |
| processo morto à força | 10061 | **sucesso** |
| **produção** | **10022** | **1920** |

Ou seja: o comentário de `cleanupSocketFile` afirma que `os.Remove` "se comporta
igual nas três" plataformas, e produção mostra um terceiro estado no Windows que
o código não trata. O mecanismo desse terceiro estado **continua desconhecido**.

### 2.3 Dois escritores no mesmo cache

Em 2026-09-08 às 16h havia dois processos servindo o cofre `Estudo`:

| PID | modo | início |
|---|---|---|
| 7920 | ponte para o daemon 10160 | 07/09 20:21 |
| 42628 | **em processo** (fallback) | 07/09 18:50 |

Prova de que os dois tinham watcher próprio: cada rename aparece duas vezes no
log, com milissegundos diferentes (`10:48:20.137` e `10:48:20.152`). O sintoma
foi `falha ao salvar cache invertido de busca ... apos 10 tentativas:
Access is denied.`

### 2.4 O diretório de runtime acumula

`%LocalAppData%\gobsidian\run`: **960 arquivos `.lock`**, **3 `.sock` órfãos**
de cofres que não existem mais, **11 `.log`** — um deles com 727 261 bytes, sem
rotação. `trava.go` documenta que o `.lock` **nunca** é removido, de propósito.
O que falta é alguém que limpe o que comprovadamente sobrou.

### 2.5 Troca de binário a quente

Medido nesta máquina, com um processo Go vivo rodando do executável alvo:

| Operação | Resultado |
|---|---|
| sobrescrever o `.exe` em execução | **falha** — `Device or resource busy` |
| **renomear** o `.exe` em execução | **funciona** |
| pôr binário novo no caminho original | **funciona** |
| processo antigo segue vivo depois | **sim** |
| apagar o antigo renomeado, ainda rodando | **funciona** |

Consequência que decide o desenho: `gobsidian update` **é** o `gobsidian.exe`
que precisa substituir. Renomear a si mesmo é o único caminho — não é otimização,
é requisito.

---

## 3. Decisões fechadas

Cada uma foi decidida pelo dono em 2026-09-08. Onde houve alternativa
considerada, ela está registrada, porque decisão sem alternativa é preferência.

| ID | Decisão |
|---|---|
| **D-01** | O instalador é um **subcomando do próprio binário**, para as três plataformas. `install.ps1` e `installer/` são apagados. |
| **D-02** | Sobra um **bootstrap mínimo** por shell — bash, PowerShell e nushell — que só baixa o executável e o roda. Além disso, o executável **baixado e rodado sem argumentos** detecta que não está instalado e se autoinstala. |
| **D-03** | A conferência de SHA-256 é feita **só pelo `update`** — o binário velho conferindo o novo. O bootstrap não confere: um binário não verifica a si mesmo com credibilidade depois de já estar rodando. |
| **D-04** | **Sem elevação.** Destino é `%LOCALAPPDATA%\Programs\gobsidian` (Windows) e o equivalente por plataforma. Nunca UAC. |
| **D-05** | A limpeza remove **só lixo comprovadamente órfão**. Cache de cofre existente nunca é tocado. |
| **D-06** | Instalar e atualizar **encerram todos os processos `gobsidian`**, sob trava global, depois de listar PID e cofre e perguntar. Recusa aborta. |
| **D-07** | Não há aposentadoria gradual nem convivência de versões. Alternativa considerada e descartada: "pato manco" — daemon velho para de aceitar conexão, para de gravar cache e sai quando a última ponte vai embora. Descartada porque exige manter três invariantes simultâneas, e invariante é o que quebra em silêncio; a indisponibilidade o usuário vê. **Medido**: a desconexão de 2026-09-07 durou 15 s (17:45:15 → 17:45:30) e o host se recuperou sozinho. |
| **D-08** | **Nenhum daemon encerra outro daemon.** `internal/daemon/trava.go` registra que o esquema de caçar PID teve corridas que três rodadas de medição não fecharam (55% de reprovação, caindo a 20%, com a causa do resto sem nome). Quem encerra é o instalador, com aval explícito do usuário. |
| **D-09** | Não haverá `gobsidian uninstall`. |
| **D-10** | Subcomandos separados de topo: `install`, `update`, `path`, `vaults`. |
| **D-11** | `gobsidian` **sem argumentos**: se não instalado **e** o terminal for interativo, autoinstala; caso contrário, ajuda. O teste de interatividade evita que um host MCP que invoque o binário sem argumento dispare uma instalação. |
| **D-12** | Sobre o travamento de 2.1: **anteparo, não caça**. Orçamento e watchdog nas três esperas, que resolvem o sintoma sem depender de achar a causa. |
| **D-13** | A RNF-30 ganha uma **segunda exceção nomeada**: `net/http` é permitido **só** em `internal/selfupdate`, com host em constante literal. É o mesmo molde da exceção de `net.Dial`/`net.Listen` com `"unix"` literal, aberta em 2026-08-05 (Task 90). |

### 3.1 Sobre a D-13

A garantia que a RNF-30 protege — *o produto não vira serviço de rede, não
escuta em porta, não fala com terceiros durante a operação normal* — continua de
pé. Um cliente HTTPS de saída, para host literal, num pacote só, alcançável
apenas pelo subcomando `update`, é coisa diferente de abrir socket.

O que a exceção exige, no mesmo rigor da primeira:

- `tools/netcheck` ganha a regra: `net/http` só em `internal/selfupdate`;
  em qualquer outro pacote continua reprovando.
- O host é constante literal. URL vinda de variável é recusada, exatamente
  como rede vinda de variável já é.
- Três casos em `scripts/check_gates.ps1`: o que recusa, o que aceita, o
  inverso.
- Redação normativa nova em `docs/PRD.md` §6.4.

---

## 4. Arquitetura

### 4.1 Pacotes novos

```
hosts      → (folha)                     os 9 hosts MCP: detecção e merge de JSON
selfupdate → config                      ÚNICO pacote com net/http
instalar   → config, console, daemon, doctor, hosts, selfupdate, vault
```

`instalar` é importado **só** por `cmd/gobsidian`. Nenhum pacote de domínio o
importa, pela mesma razão que nenhum importa `boot`. O grafo continua acíclico.

Justificativa de cada aresta nova, no rigor que o CLAUDE.md exige:

- `instalar → daemon`: a trava global reusa `tentarTravar`, a primitiva de trava
  de kernel que já existe em `internal/daemon/trava.go`. Escrever uma segunda
  trava seria duas contas da mesma regra — a classe de defeito que este projeto
  registra em vários lugares.
- `instalar → doctor`: a limpeza de lixo órfão é a mesma conta que
  `doctor --fix` faz. Uma função, dois chamadores.
- `instalar → hosts`, `instalar → selfupdate`: composição direta.
- `selfupdate → config`: precisa do diretório de instalação e da chave de
  versão. Nada mais.
- `hosts` é folha: recebe a raiz do sistema de arquivos e o comando a registrar
  por parâmetro, e não conhece cofre nem índice.

Código de plataforma fica atrás de build tag, em arquivo separado. Nunca
`if runtime.GOOS ==` dentro de lógica compartilhada.

### 4.2 Superfície CLI

| Comando | O que faz |
|---|---|
| `gobsidian install` | instala o binário; **pergunta** se faz PATH e cofres |
| `gobsidian update` | confere SHA-256, encerra processos, troca o binário |
| `gobsidian path --add \| --remove` | só a entrada de PATH |
| `gobsidian vaults` | só a configuração de cofres e hosts MCP |
| `gobsidian` (sem args) | D-11 |

Flags de `install`, todas com equivalente em variável de ambiente, como o
`install.ps1` já tem hoje: `--vault`, `--hosts`, `--install-dir`, `--yes`,
`--read-only`, `--no-path`.

`--yes` faz **as três coisas** — binário, PATH e cofres. É o que o `install.ps1`
faz hoje, e mudar o padrão quebraria quem já automatiza. `--no-path` e
`--hosts none` recortam.

### 4.3 Manifesto

`install` grava um manifesto JSON com: caminho do binário, versão, hash
instalado, entrada de PATH criada, e quais hosts foram tocados.

Sem ele, `update` adivinha o que reconfigurar, e "estou instalado?" (D-11) não
tem resposta mecânica. **A detecção de D-11 é**: existe manifesto **e** o
executável em curso está no caminho que ele registra.

### 4.4 Sequência de `update`

1. Toma a **trava global de instalação**. A partir daqui, nenhum `serve` e
   nenhum `daemon` sobe.
2. Baixa o release e o `SHA256SUMS.txt`; **confere; aborta em divergência**.
3. Lista os processos `gobsidian` com PID e cofre. Pergunta. Recusa aborta.
4. Encerra: sinal primeiro, força depois de um orçamento.
5. Confirma que ninguém subiu de novo — a trava garante, este passo prova.
6. Renomeia o binário velho, põe o novo no caminho original, apaga o velho
   (§2.5).
7. Solta a trava.

Escrita interrompida no passo 4 é recuperável: `vault.ReplaceFile` é
temp+sync+rename e o boot já varre temporários órfãos — garantia que
`internal/vault/atomic.go` já documenta.

### 4.5 Trava global de instalação

Trava de kernel (`LockFileEx` / `flock`), no molde de `internal/daemon/trava.go`:
arquivo nunca removido, posse decidida pela trava e nunca pela existência.

Todo `serve` e todo `daemon`, ao subir, consultam a trava e **saem na hora**,
com mensagem explícita, se ela estiver tomada. Sem isso, o host respawna o
servidor no meio da troca e duas versões voltam a conviver — que é o defeito que
D-06 e D-07 existem para fechar.

### 4.6 Regra de limpeza (D-05)

Mecânica, nunca heurística:

| Alvo | Condição para remover |
|---|---|
| `.lock` | trava **não** tomada (`TravaEmUso`) **e** cofre não existe mais |
| `.sock` | sem ouvinte (`alguemEscuta`) **e** cofre não existe mais |
| `.log` | acima de 5 MB → **rotacionado**, guardando um arquivo anterior; nunca apagado |
| diretório de cache | cofre não existe mais |

"Cofre não existe mais" é decidível: `search.CacheHeader.VaultPath`
(`internal/search/persist.go:44`) já guarda o caminho e já é gravado
(`persist_codec.go:187`). A limpeza portanto funciona também para instalações
que já existem.

**Cache de cofre existente nunca é tocado.** Reconstruir custou 3021 ms no cofre
de referência do dono (3 257 notas, medido em 2026-09-07T18:50:56).

---

## 5. Logs

Só **campos novos**. Nenhuma linha existente muda de texto — `scripts/measure.ps1`
faz parsing de `servidor pronto` e de `index_ms=`, e `scripts/test_orphans.ps1`
lê `reason=`.

| Mudança | Por quê |
|---|---|
| `pid=` e `versao=` em todo log do daemon | O arquivo é único e recebe append de N instâncias. Determinar quem escreveu cada linha custou três rodadas de análise nesta investigação, por mtime de arquivo e `StartTime` de processo. |
| fallback em processo sobe de `INFO` para `WARN`, com `motivo=` | Os três casos — socket mudo, versão divergente, config divergente — dão hoje a mesma linha. Um deles ficou 20 h ligado sem ninguém ver, e `docs/OPERACAO.md` já registra que isso custou um marco inteiro desligado em produção antes. |
| as três esperas de encerramento ganham nome e log de entrada e saída | É o que teria respondido §2.1 em segundos. |
| `doctor` e `vault_stats` passam a dizer o modo e o dono do `--cache-dir` | Dois escritores no mesmo cache só apareceram por comparação de milissegundos entre linhas duplicadas. |

`ErrVersionMismatch` deixa de ser silencioso mesmo com D-07 no lugar: hoje a
ponte que o recebe cai calada para o modo em processo.

---

## 6. Anteparo (D-12)

O guard de `os.Exit` que hoje cobre só `lifecycle.Shutdown` passa a cobrir as
três esperas de §2.1. Orçamento **único** para o encerramento inteiro, não por
etapa — orçamento por etapa somaria e o total ficaria sem teto.

O número é **6 s**, e o desenho é um guarda-chuva **único**, armado no início da
sequência de encerramento e cobrindo `lifecycle.Shutdown` **e** as três esperas
seguintes. Não é um segundo relógio somado ao que já existe: é o mesmo relógio,
com alcance maior.

O valor não pode ser escolhido livremente. `scripts/test_orphans.ps1` mede com
uma janela de `$SettleMs = 8000`, e o comentário dela (linha 6) fixa a relação:
*"8s > o guarda-chuva de 6s de lifecycle.Shutdown. (...) Se este numero mudar, o
de la mudou primeiro."* Ou seja, a janela do harness tem de ser **maior** que o
guarda-chuva do produto. Adotar 8 s aqui igualaria os dois e criaria corrida na
borda — o processo sairia exatamente quando o gate para de esperar.

Manter 6 s preserva a relação, e **nenhum script de gate precisa mudar**. Os
orçamentos por etapa dentro de `Shutdown` (watcher, 500 ms) continuam como
estão.

O cenário `parent-death` do harness já usa `$SettleMs + 7000`, porque ali a
detecção da morte do pai custa até 5 s antes de o encerramento começar. Essa
soma continua válida sem alteração.

Isso resolve o sintoma de 2026-09-07 independentemente de a causa ser
encontrada. A causa continua registrada como aberta em `docs/ESTADO.md`.

---

## 7. Testes

O que existe hoje e **não pegou** o defeito: `scripts/test_orphans.ps1
-Scenario daemon-idle` roda 100 ciclos no CI e passa. Produção produziu
exatamente um encerramento por ociosidade pendurado. A hipótese é que o cofre
sintético do cenário é pequeno demais para exercitar watcher e índice reais.

| Teste | O que prova |
|---|---|
| cada uma das três esperas travada de propósito | o watchdog dispara. Exige subprocesso, porque é `os.Exit`; o harness de órfãos já sabe fazer isso |
| `cleanupSocketFile` no ramo de falha | 20+ falhas em produção, zero teste hoje |
| dois processos no mesmo `--cache-dir` | a corrida de §2.3 |
| `hosts` contra golden files dos 9 hosts | 1 009 + 729 linhas com zero teste hoje, e é o código que mata processo do usuário |
| `instalar` com as dependências perigosas como interfaces | a sequência de §4.4 sem tocar na máquina |
| `selfupdate` com transporte falso em memória | sem `net` em teste |
| instalador ponta a ponta, em diretório temporário | PATH e manifesto de verdade |

Cada regra nova entra com prova de mutação (`scripts/mutate.ps1`), como o
projeto exige. Um teste que não pode falhar é pior que teste ausente.

---

## 8. CI

| Lacuna atual | Mudança |
|---|---|
| `verify.ps1` é o gate documentado, mas o CI reimplementa um subconjunto — duas contas da mesma regra | `verify.ps1` passa a rodar no CI |
| `install.ps1` (729 linhas) e `install.js` (1 009) têm zero cobertura | job novo do instalador, nas três plataformas |
| `release.yml` não depende de `ci.yml`: uma tag libera com CI vermelho | passa a exigir `ci.yml` verde |
| `mutate.ps1`, `check_gates.ps1`, `audit_reports.ps1` nunca rodam no CI | entram, via `verify.ps1` |
| `go-version: '1.25'` fixo, enquanto a toolchain do dono é 1.26.5 | alinhado |
| os três bootstraps de D-02 não existem ainda | smoke test para bash, PowerShell e nushell |

---

## 9. Fora de escopo

- `gobsidian uninstall` (D-09).
- Migração de versão do formato de cache. Com D-06/D-07 não há convivência de
  versões, então o problema não se coloca.
- Assinatura Authenticode. Foi considerada em D-03 e descartada: custa
  certificado, e a conferência de hash pelo `update` cobre o caminho de
  atualização, que é por onde quase todo mundo passa depois da primeira vez.
- Achar a causa de §2.1 (D-12).

---

## 10. Riscos conhecidos

| Risco | Mitigação |
|---|---|
| A causa de §2.1 continua desconhecida e pode se manifestar em outro lugar | O anteparo é genérico: cobre as três esperas, não uma causa específica. A dívida fica registrada em `docs/ESTADO.md`. |
| O terceiro estado de socket de §2.2 não foi reproduzido | `cleanupSocketFile` ganha plano B e passa a **relatar** o estado que encontrou, em vez de só devolver o erro. O que não se sabe reproduzir, se registra. |
| Trava global esquecida tomada trava toda partida | Trava de kernel: se o processo morre, o kernel solta. É a razão pela qual `trava.go` abandonou o arquivo com PID. |
| Encerrar processos derruba trabalho em curso | Pergunta antes, listando PID e cofre, e aborta na recusa — comportamento que os dois instaladores atuais já têm e que D-06 preserva. |
