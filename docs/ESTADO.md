# ESTADO.md — onde o projeto está

Fonte da verdade sobre marcos, medições e dívidas abertas. **Nenhum número aqui
é estimativa** — quando não foi medido, diz "não medido".

Para o que está sendo executado agora, o ledger é a autoridade:

```bash
pwsh -File scripts/sdd.ps1 status
```

---

## Marcos

**M0 — completa**, etiquetada `m0-lifecycle`: ciclo de vida, `internal/vault`,
servidor MCP mínimo com `vault_stats`, `doctor`, e 100 ciclos de encerramento
abrupto com zero órfãos.

A dívida de revisão do M0 foi **paga**. As Tasks 9, 10 e 11 haviam fechado sem
revisão fresca; a revisão que faltava rodou e virou trabalho. Três defeitos
reais que os gates existentes não pegavam: `doctor` saindo 0 com cofre
inacessível; o gate de órfãos não gateando em `reason=`, de modo que servidor
morrendo sozinho dava rodada verde sem mecanismo nenhum disparar; e
`cmd/gobsidian` sem teste algum.

Lint limpo nos três alvos (`GOOS=linux/darwin/windows`), depois de 39 achados
que estavam vermelhos desde o commit de bootstrap. O CI ganhou `fmt` (gofmt +
vet cruzado) e `lint-windows`, sem o qual todo arquivo `//go:build windows`
ficava sem análise.

**M1 — completa** (Tasks 12 a 26): parser e as quatro extensões goldmark
congelados por um corpus de 48 golden files, o índice com offsets de byte,
resolução, backlinks, consultas, a fachada de serviço, as cinco tools de leitura
e os resources, e paridade verificada contra um dump real do `metadataCache` do
Obsidian.

**M2 — completa** (Tasks 27 a 32): fachada sobre fsnotify com filtro de
relevância unificado em `vault.Classify`, debounce de tique único com conjunto
sujo, verificação de mudança real ligada a `index.Replace`, reconciliação por
overflow, correlação de rename por `xxhash`, e os contadores em `vault_stats`.

**M2.1 — completa** (Tasks 33 a 42). A revisão do M2 (2026-07-28) encontrou três
Critical, cinco Important e um lote de higiene, cada um reproduzido por mutação
ou sonda, nenhum inferido:

- `CorrelateRenames` abria anexo e placeholder somente-nuvem, furando duas
  regras fechadas que `index.Replace` respeita.
- O reconciliador de overflow (P0, RF-05) tinha cobertura zero: removido
  inteiro, o teste continuava verde.
- Link resolvia para nota deletada, por divergência de caixa na chave de
  `byAlias`.

Três decisões foram fechadas antes de escrever as tarefas e **não devem ser
re-litigadas**: `--debounce-ms=0` passa a ser recusado na config;
`index.MoveNote` fica, pagando as dívidas que contraiu ao entrar fora do
contrato; e os contadores de descarte são publicados desdobrados por motivo em
`vault_stats`.

**M7 — completa** (Tasks 78 a 93), em duas partes.

A **Parte I** (78–87) atacou a busca: `sync.Pool` de transformers, `TitleNorm`
pré-computado, chave única `nomeChave` para resolução, e o corpus de contraste
que tornou as perguntas de verificação respondíveis. Medido:

| | antes | depois |
|---|---|---|
| busca | 218,5 ms | **115,0 ms** |
| alocação | 188,49 MiB | **50,42 MiB** (−73%) |
| allocs | 128,89k | **14,12k** (−89%) |

A **Task 82 foi revertida**: `benchstat` deu `~`, e mudança sem ganho
significativo é dívida pura.

A **Parte II** (88–93) atacou memória entre instâncias: carga preguiçosa do
índice de busca (`--eager-search` liga a antiga), arena mapeada em memória, o
RNF-30 reformulado, o transporte IPC, e o daemon. Medido no cofre real de 4.513
notas, memória física agregada:

| sessões | pré-M7 | sem daemon | com daemon |
|---|---|---|---|
| 1 | 579,1 MB | 244,6 MB | 223,6 MB |
| 3 | 1.681,3 MB | 508,5 MB | 262,2 MB |
| 5 | 2.916,4 MB | 773,4 MB | 229,4 MB |

A coluna do daemon **não escala com N** — é a assinatura de um índice pago uma
vez só.

---

## Decisões fechadas que não se re-litigam sem dado novo

**O RNF-30 mudou de redação, não de intenção** (Task 90, autorizado pelo dono em
2026-08-05). Era "nenhum socket"; é "nenhum socket que saia da máquina".
`tools/netcheck` aceita `net.Dial`/`net.Listen` **apenas com a rede na constante
literal `"unix"`** — rede vinda de variável é recusada. Escreva a string no
lugar; guardá-la numa variável deixa o `check_net` vermelho, e isso é a regra
funcionando. A redação normativa está em `PRD.md` §6.4 e `ARCHITECTURE.md`.

**A escolha do transporte foi medida (D-M7-6).** AF_UNIX contra named pipe, ida
e volta, 20.000 repetições: 25,7 contra 82,9 µs em 256 B; 23,0 contra 93,5 em
4 KB; 42,9 contra 110,0 em 64 KB. Está na biblioteca padrão e é o mesmo código
nos três sistemas; build tag só para o caminho do socket e a limpeza.

**`GOGC` foi rejeitado duas vezes** — ver `ARMADILHAS.md`, seção de medição.

---

## RNF-07 foi redefinido (2026-08-28)

Era `RSS em repouso ≤ 60 MB`. Agora é **`heap vivo ≤ 8 MB + 32 KB × notas`**, no
estado nomeado **`servindo`** — depois de ao menos uma busca, que é onde toda
sessão real está. Redação normativa em [`PRD.md`](PRD.md) §6.1; medições e o
método em [`OPERACAO.md`](OPERACAO.md).

Três defeitos motivaram, e cada um está medido:

1. **RSS não media o que o requisito queria.** Ele acompanha a meta de heap do GC,
   e por isso inverteu de sinal: um binário **sem** um campo consumia 3,6 MB **a
   mais** que o binário com ele, reprodutivelmente.
2. **"Em repouso" não dizia se a busca já tinha acontecido**, e desde a carga
   preguiçosa isso muda o número por até 3,9×.
3. **O alvo era absoluto** e vinha de um cofre sintético. Num cofre real de 5.686
   notas o mesmo protocolo dava 129,6 MB de RSS contra os 37,95 MB publicados.

Sob a regra nova, os cinco cofres do dono **passam**, com folga de 15% a 64%.
`scripts/measure.ps1` foi reescrito para medir isto — ele media a **ponte** em vez
do servidor quando havia daemon, media um estado só, e não conferia se o índice
tinha vindo do cache.

**Consequência de escala, dita e não escondida:** a 20.000 notas, que é o que o
RNF-09 promete, o teto vira **633 MB**. Se isso for inaceitável, o que muda é a
estrutura do índice invertido — não o requisito.

---

## Estado dos achados da auditoria de 2026-08-25

`docs/SUGESTOES.md` levantou 61 achados. Em 2026-08-27 o quadro é:

| Severidade | Fechados | Rejeitados após verificação | Abertos |
|---|---|---|---|
| Críticos | 3 de 3 | — | — |
| Altos | 8 de 8 | — | — |
| Médios | 15 | M1 | **—** |
| Desempenho | 14 | P11 | **—** |
| Baixos | 17 | B15 | **—** |

**Todos os 61 achados estão fechados ou rejeitados com fundamento** (2026-08-31).
Fecharam também o item 4 do brief da Task 126 (o lock do `EnsureStarted`) e o
teste da tempestade.

Três foram **rejeitados depois de verificados**, e a verificação é o registro
que importa: **M1** estava prescrito ao contrário; **B15** pede guarda para um
caso que o próprio sistema de arquivos torna indistinguível; e **P11** foi
rejeitado por uma sondagem que, em 2026-08-31, se descobriu **mascarada** — a
máquina tem `LongPathsEnabled = 1`, e um caminho **relativo** de 327 caracteres
também passa, o que o `fixLongPath` do Go não explica. A metade do P11 que era
defeito de verdade — o descarte silencioso de erro de subárvore — está corrigida
por `SweepResult`; se o prefixo explícito é necessário com a chave desligada
**segue sem verificação**. Os três estão em [`OPERACAO.md`](OPERACAO.md), o P11
com a correção da rejeição.

**P1, P2 e P3 saíram do congelamento em 2026-08-28**, junto com a
**Oportunidade 1** (BM25 em IDs densos) que os subsumia. O perfil que destravava
a decisão foi feito e mudou a resposta: o BM25 vale 16% da CPU da busca mas
**79% da alocação** dela. Medido intercalado: busca **−31% a −45% de tempo** e
**−42% a −47% de alocação**, com ordem de ranking idêntica em seis consultas
contra cofre real. É o maior ganho de desempenho da série.

A seção de achados do próprio `SUGESTOES.md` **não foi reescrita** e tem aviso no
topo: riscar dezenas de itens a mão é edição em que um fica para trás, e achado
marcado como fechado sem ter sido é pior que o contrário. O ledger é a fonte.

---

## Formato de cache

**O formato do cache de busca é o 6, e não é `gob`** (formato 5 em 2026-08-03/04;
formato 6 na Task 89). A extensão continua `.gob` por compatibilidade de
caminho; o conteúdo é um codec binário próprio em
`internal/search/persist_codec.go`.

O índice virou duas camadas: base imutável em arrays achatados vinda do cache
(`soa.go`) mais um delta em mapas com o que mudou desde a partida. O formato 6
acrescentou uma arena mapeada em memória (`mmap.go`), que é o que permite várias
instâncias compartilharem as páginas do índice.

Medido no cofre real de 3.152 notas na virada do 4 para o 5:

| | formato 4 | formato 5 |
|---|---|---|
| carregamento | 5,59 s | **659 ms** |
| arquivo | 482 MB | **67 MB** |
| boot quente | ~7 s | **842 ms** |

**O formato do cache de metadados é o 3** (2 até 2026-08-26). O 3 acrescentou o
`Context` de cada link — o texto ao redor da referência, que `docs/TOOLS.md` já
prometia em `backlinks` e que o código entregava vazio desde sempre (achado A8).

Ele é persistido, e não recalculado como `Resolved`/`Via`/`State`, porque **não é
derivável do que o índice guarda**: recortá-lo de novo exigiria reler o corpo de
cada nota no boot, que é exatamente o custo que este cache existe para não pagar.

Medido em 2026-08-26 no cofre real de 5.686 notas e 42.329 links (109 MB):

| | formato 2 | formato 3 |
|---|---|---|
| arquivo | 19,53 MB | **32,62 MB** (+67%, +13,09 MB) |
| `LoadIndexCache`, mediana de 5 | 275,5 ms | 282,2 ms |

O custo de disco é real e está medido. **O de tempo não é distinguível de ruído
nesta amostra**: as duas distribuições se sobrepõem (com contexto: 236–450 ms;
sem: 258–292 ms), e a rodada *com* contexto produziu as duas amostras mais
rápidas.

O boot completo foi medido em seguida, no mesmo cofre, contra o teto de 300 ms do
RNF-02 — `index_ms` do log `servidor pronto`, em processo (`GOBSIDIAN_NO_DAEMON`),
somente-leitura e com `--cache-dir` próprio, para não gravar formato 3 no cache
que as sessões vivas do dono leem:

| | formato 2 | formato 3 |
|---|---|---|
| boot quente, mediana de 5 | **891 ms** | **921 ms** |
| amostras | 810–1079 ms | 872–1034 ms |
| boot frio, n=1 | 1741 ms | 2326 ms |

**O RNF-02 está estourado nas duas — 3× o teto — e já estava.** Ele é publicado
como NÃO ATINGIDO em [`OPERACAO.md`](OPERACAO.md) desde 2026-08-06. Neste cofre o
delta mediano de +30 ms não é distinguível de ruído: as faixas se sobrepõem, e a
do formato 2 é a mais *larga* das duas.

**O formato do cache de metadados é o 5** desde 2026-08-26. O backlink ganhou
`Heading` — o título da seção da nota de origem, **derivado** dos headings já
indexados, a custo zero de disco (ver `headingDoLink`). `contextoBytes` foi para
40 no formato 4 e **voltou para 80 no 5**, quando a medição que motivara o corte
foi retratada. Os dois bumps mudam o CONTEÚDO sem mudar o layout, e são
necessários pelo mesmo motivo: um cache gravado com outro `contextoBytes` carrega
trechos de outro tamanho, e aceitá-lo faria a mesma pergunta ser respondida com
recortes diferentes conforme o cache fosse velho ou novo.

Uma versão anterior desta seção afirmava que o formato 3 empurrava o RNF-02 de
atingido para não atingido num cofre local. **Retratado.** Aquelas bateladas eram
sequenciais e a máquina derivou entre elas; alternar a ORDEM das bateladas não
basta, porque elas continuam separadas no tempo. Refeito com três binários lado a
lado e **uma rodada de cada por vez**, n=10, no cofre local
`Obsidian\Jurisprudência`:

| variante | cache | mediana de 10 | acima do teto de 300 ms |
|---|---|---|---|
| sem contexto | 9,76 MB | 179 ms | 0 de 10 |
| **contexto de 80** (o formato 5) | **19,05 MB** | 193 ms | 0 de 10 |
| contexto de 40 | 16,95 MB | 191 ms | 0 de 10 |

**Os três passam no RNF-02 nesse cofre.** As medianas diferem em 14 ms, menos que
a variação dentro de uma única variante. **O tamanho do cache é o único custo que
sobrevive à medição** — e ele é determinístico. Foi com esse número na mão que o
dono escolheu ficar com 80: os 2,1 MB de diferença compram o dobro de contexto, e
não custam tempo mensurável. Detalhe e a lição de método em
[`OPERACAO.md`](OPERACAO.md).

O boot frio tem **uma amostra só de cada** e não sustenta conclusão. A diferença
está na direção que se espera de gravar 13 MB a mais — `index_ms` inclui
`SaveIndexCache` —, mas com n=1 isso é hipótese, não medida. As duas passam no
alvo de 3 s do RNF-01.

O tamanho é governado por `contextoBytes` (80 de cada lado) em
`internal/index/contexto_link.go`, num lugar só, para poder ser discutido.

**Toda troca de formato reconstrói o cache de todo cofre no boot seguinte**, em
segundo plano, com as outras doze tools respondendo desde o primeiro segundo — e
isso vale para quem atualizar de uma v1.0.x, que grava `gob` e invalida o cache
desta versão a cada alternância.

`docs/PRD.md` Q3 decidiu persistir **dois** caches, e desde a Task 85 os dois
existem: o cache do índice de metadados entrou, e o boot com cache válido caiu de
1.192–1.396 ms para 371–472 ms num cofre real.

---

## Gates

**O gate de órfãos cobre os quatro cenários, e o padrão roda os quatro.**
`scripts/test_orphans.ps1 -Cycles 100` executa `stdin-eof`, `parent-death`,
`signal` e `daemon-idle` em sequência, e cada um **reprova se o `reason=` não
for o do mecanismo que ele nomeia** — encerrar pelo motivo certo por acidente
não conta.

- `parent-death` desconecta o EOF (cadeia keeper → host → servidor, com o keeper
  segurando a ponta de escrita do pipe).
- `signal` deixa tudo vivo e só manda CTRL_BREAK.
- `daemon-idle` é estruturalmente diferente: o daemon não tem pai nem stdin de
  host, então a vigília do pai **não se aplica** — não a ligue por consistência.
  Quem substitui é a ociosidade, com padrão de 15 minutos
  (`daemon.DefaultIdleSeconds`); o cenário usa `--idle-seconds` curto, e esse
  valor não pode vazar para o padrão.

Isto esteve escrito como lacuna aberta por mais tempo do que foi verdade. Os
cenários existem desde 2026-08-02 e o CI os chama explicitamente, mas o padrão
do script era `stdin-eof` — então quem rodava o comando documentado localmente
via `[OK]` depois de exercitar **um** dos três. O padrão passou a ser `all` por
causa disso. **Gate cujo padrão cobre parte do que ele aparenta cobrir é pior
que gate ausente.**

**`test_orphans.ps1` não compila — ele roda o que estiver em `bin/`.** Hoje
recusa binário mais velho que o código. Antes disso, um binário de quatro dias
antes deu três `[OK]` nos cenários que não dependiam do código novo e 100 falhas
de "daemon nao anunciou prontidao" no que dependia — mensagem que aponta para o
daemon quando a causa era o subcomando não existir naquele build. A guarda fica
**antes de qualquer despacho**: a primeira versão dela cobria três cenários e não
o quarto, que é exatamente o defeito que ela existe para impedir.

---

## Dívidas abertas

- **`scripts/measure.ps1` continua fora de gate nenhum.** É o único instrumento
  que responde por RNF-01 e RNF-07, e roda quando alguém lembra. A forma de
  fechar existe — `gen_vault.ps1` produz cofre determinístico, e `bench.yml` já
  tem a disciplina de um runner só com referência commitada —, mas o que isso
  compraria é **detecção de regressão em cofre sintético**, não validação dos
  números publicados de RNF-07, que vêm dos cinco cofres reais do dono.
- ~~**Os tetos de latência não eram cobrados por gate nenhum.**~~ **Fechado em
  2026-09-01.** Eles só valem sem `-race`, e o `ci.yml` roda só com `-race`.
  Agora há o job `tetos-de-latencia` no `bench.yml`. O runner foi **medido antes
  de ligar**: pior p95 de 1,64 ms / 1,79 ms / 143 µs contra tetos de 100 ms /
  22 ms / 80 ms — folga de 12× a 550×. Os 107,1 ms que assustavam eram uma
  medição COM `-race`.
- **A folga do RNF-07 em Jurisprudência é de 15%, a mais apertada das cinco.**
  O requisito foi redefinido em 2026-08-30 — heap vivo ≤ 8 MB + 32 KB × notas,
  nos estados `pronto` e `servindo`, decisão do dono — e **os cinco cofres reais
  passam**. O que segue aberto é o caminho para folga maior: o índice de
  metadados é **67% do heap vivo**, e dentro dele `Link.Raw` repete `Link.Target`
  em **100,0% dos 28.045 links** medidos, o que vale ~3,3 MB sem perder
  informação. Medido em 2026-08-30; tabela em `OPERACAO.md`.

  > Uma correção do próprio texto: até 2026-08-31 este item começava por "o
  > RNF-07 **não é atingido** em Jurisprudência", e o corpo dele dizia, duas
  > linhas abaixo, que os cinco passam. O título era resíduo da redação anterior
  > à redefinição do requisito, quando o alvo era RSS ≤ 60 MB. Re-medido em
  > 2026-08-31: `pronto` 30 MB, `servindo` 40 MB, contra teto de 47,2 MB.
- ~~**A corrida residual do daemon**~~ **fechada em 2026-08-31.** A posse virou
  trava do kernel (`flock` / `LockFileEx`), e com ela não existe lock obsoleto
  nem recuperação — a classe inteira de corrida sumiu junto com
  `lockObsoleto`, `pidVivo` e ~120 linhas. Medido: **0 de 40** em Linux, contra
  11 de 20 antes. Tabela completa em `OPERACAO.md`.
- **As duas listas de achados estão fechadas.** A auditoria de 2026-08-25 fechou
  em 2026-08-27; as quatro tarefas restantes da revisão de 2026-08-15 — 107, 109,
  112 e 114 — foram entregues em 2026-08-31, depois de uma auditoria do ledger
  achá-las **em estado nenhum**: nem feito, nem aberto. Elas não estavam no
  ledger porque as irmãs foram entregues sob a numeração da auditoria. Conferir
  no código foi o que as achou.
- **Uma questão que a Task 114 abriu e não fechou:** as chaves do índice
  normalizam para NFC, mas o disco pode ter as duas formas ao mesmo tempo. Duas
  notas cujos nomes só diferem na normalização são dois arquivos para o sistema
  de arquivos e **uma chave só** para o índice; hoje a segunda ganha o lugar da
  primeira em `lowerPath`. **Não medido** em cofre real.

---

## Onde o trabalho é rastreado

O ledger fica em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`. O
caminho plano antigo virou ponteiro: os dois derivaram e um tinha 16 tarefas
enquanto o outro tinha 6.

**`.superpowers/` é versionado.** Era ignorado por inteiro — então o ledger, a
única coisa que atravessa sessões, existia só na cópia de trabalho. Remover a
linha do `.gitignore` não bastou: havia um segundo `.gitignore` com `*` dentro
de `.superpowers/sdd/`, que o `sdd-workspace` do plugin **recria**, e que negação
no diretório pai não cancela. `sdd.ps1` o apaga a cada chamada. O arquivo
`task-N-base.txt` fica sujo de propósito — commitá-lo move o HEAD e a base
recursa.
