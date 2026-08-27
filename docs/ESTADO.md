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

**Isso vale para ESTE cofre e não generaliza.** Medido em seguida num cofre fora
do OneDrive (`Obsidian\Jurisprudência`, 1.254 notas, disco local), com n=13 por
formato e as bateladas alternadas: mediana **243 ms no formato 2** contra
**323 ms no formato 3**, cache 9,76 → 19,05 MB (+95%). **Ali o formato 3 empurra
o RNF-02 de atingido para não atingido** — 3 de 13 amostras acima do teto no
formato 2, contra 10 de 13 no formato 3, e 9 das 13 do formato 2 ficam abaixo da
MENOR amostra do formato 3.

O cofre em OneDrive escondeu o efeito porque lá o ruído tem ~270 ms de largura e o
efeito tem ~90 ms — e o RNF-02 já estava 3× estourado pelos dois formatos, o que
tornava a comparação acadêmica. No cofre local, quieto, a métrica vive **em cima
da linha**, e é onde 80 ms decidem. Tabela completa em
[`OPERACAO.md`](OPERACAO.md).

O boot frio tem **uma amostra só de cada** e não sustenta conclusão. A diferença
está na direção que se espera de gravar 13 MB a mais — `index_ms` inclui
`SaveIndexCache` —, mas com n=1 isso é hipótese, não medida. As duas passam no
alvo de 3 s do RNF-01.

O tamanho é governado por `contextoBytes` (80 de cada lado) em
`internal/index/contexto_link.go`, num lugar só, para poder ser discutido.

**Toda troca de formato reconstrói o cache de todo cofre no boot seguinte**, em
segundo plano, com as outras onze tools respondendo desde o primeiro segundo — e
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

- **`scripts/measure.ps1` continua fora de gate nenhum.** As decisões de RSS
  (`GOGC`, `FreeOSMemory`, `maxSnippetWorkers`) dependem dele.
- **O RNF-07 é medido por um caminho que exclui o índice de busca.**
  `measure.ps1` emite `initialize` + `vault_stats`, e com a carga preguiçosa
  (Task 88) uma sessão que nunca buscou nunca carregou o invertido. Medido em
  2026-08-26, cache quente, uma variável (uma `vault_search`):

  | Cofre | só `vault_stats` | + uma busca | delta |
  |---|---|---|---|
  | Oral (78 notas) | 22,3 MB | 29,5 MB | 1,32× |
  | Estudo (2.557 notas) | **53,1 MB** | **149,0 MB** | **2,81×** |

  `OPERACAO.md` registra RNF-07 (≤ 60 MB) como **Atingido** com 37,95–38,10 MB.
  Pelo protocolo B, Estudo dá 2,5× o alvo. Decisão do dono pendente: ou o RNF-07
  passa a nomear o estado "já atendeu uma busca" com alvo re-negociado, ou a
  tabela declara que mede o servidor antes da primeira busca.
- **A corrida residual do daemon** (dois daemons vivos sob carga) segue
  registrada nos limites conhecidos de `OPERACAO.md`.
- **Achados abertos da revisão de 2026-08-15** estão indexados em
  `docs/wiki/notes/achados-abertos.md`; a auditoria de 2026-08-25 está em
  `docs/SUGESTOES.md`, com as decisões do dono já registradas lá.

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
