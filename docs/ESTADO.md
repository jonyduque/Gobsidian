# ESTADO.md — onde o projeto está

Fonte da verdade sobre marcos, medições e dívidas abertas. **Nenhum número aqui
é estimativa** — quando não foi medido, diz "não medido".

Para o que está sendo executado agora, o ledger é a autoridade:

```bash
pwsh -File scripts/sdd.ps1 status
```

---

## Marcos

**Go 1.27, e o que a troca de toolchain já tinha mudado — completo, 2026-09-09.**
Dez commits, `verify.ps1` verde em cada um, CI verde nos catorze jobs. O plano é
[`superpowers/plans/2026-09-09-go-1-27.md`](superpowers/plans/2026-09-09-go-1-27.md),
com as onze tarefas fechadas; as medições estão em "Go 1.27 — o que foi medido".

Metade do marco não foi adoção de novidade: a v1.6.0 já tinha sido publicada
compilada com a 1.27.1, e ninguém tinha olhado o que isso mexeu. Mexeu em duas
chaves derivadas que passavam por tabela Unicode da **stdlib** — que a toolchain
move sozinha, ao contrário do `x/text`, que é módulo fixado. `config.VaultKey`
nomeia o diretório de cache **e o caminho do socket**, e o analisador de busca
decide o que é termo no índice **persistido**. A primeira deixou de depender de
tabela móvel; a segunda passou a carimbar a geração Unicode no cabeçalho do
cache, então ele se invalida sozinho na próxima. O cache de metadados não
precisou de nada — ele recalcula toda chave derivada ao carregar, e essa é a
distinção que decide se uma tabela móvel importa.

*Diagnóstico:* o guarda-chuva sabia que o encerramento tinha travado, dizia isso
no log e saía sem olhar. Agora ele despeja as pilhas de todas as goroutines e o
perfil `goroutineleak` antes do `os.Exit(1)` — em processo, que é o que `-s -w`
não atrapalha e o `dlv attach` não conseguiu fazer com o PID 42856. Os rótulos
saem no traceback pela diretiva `go 1.27`, então o dump **nomeia** a espera que
travou em vez de mostrar endereço.

*Harness:* sete gates novos no dia, cada um com o defeito concreto que o
originou — grafo, pins, isolamento de teste, invariante de partida, prompt,
tabela Unicode e pin de toolchain contra a diretiva. `check_gates.ps1` foi de 29
para 49 casos e `verify.ps1` de 15 para 22 etapas.

**Instalador no binário e encerramento com teto — completo, 2026-09-08/09.**
Vinte commits, `verify.ps1` verde em cada um, gate de órfãos verde nos quatro
cenários. Duas frentes que a investigação de um `Server transport closed
unexpectedly` abriu ao mesmo tempo.

*Encerramento:* nenhuma espera fica sem orçamento. `lifecycle.ArmarGuardaChuva`
cobre o intervalo inteiro nos três pontos de saída, e as cinco esperas dizem no
log que começaram — a linha que faltou para determinar qual delas pendurou o
PID 42856 por 20 h. O fallback para o modo em processo deixou de ser `INFO` e
passou a nomear o motivo entre quatro casos distintos.

*Instalador:* virou subcomando do próprio binário. `install.ps1` (729 linhas) e
`installer/` (1 090, dos quais 1 009 são o `install.js`) foram apagados;
**126 linhas de bootstrap substituem 1 819**. A lógica duplicada de verdade —
os dois instaladores, um em PowerShell e outro em Node — eram 1 738 dessas
linhas, e é esse o número que a spec e os comentários dos pacotes citam.
Três pacotes novos — `hosts`, `selfupdate`, `instalar` —, quatro subcomandos, e a
RNF-30 com uma segunda exceção nomeada, cujo gate entrou **antes** do código que
ele governa.

O que **não** foi resolvido está nas dívidas abertas abaixo, e é honesto: a causa
do encerramento pendurado e o estado de socket `10022`/`1920` seguem
desconhecidos. O que entrou foi anteparo e saída, não conserto.

Duas coisas que o próprio trabalho pegou, e que valem mais que a lista acima:
o gate de órfãos mostrou que a presença de processos, como escrita primeiro,
levava o diretório de runtime de 6 para 130 arquivos em 100 ciclos — a mesma
forma do lixo de 960 `.lock` que a limpeza existe para varrer. E uma prova de
mutação mostrou que o teste de `vault_stats.modo` passava com o campo fixado em
vazio dentro da função. Os dois foram consertados com o teste que os nomeia.

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

### Baseline dos benchmarks de análise — 2026-09-02, HEAD 6c5d1f1

Máquina: i7-10750H, windows/amd64, Go 1.26.5. Binários e saídas brutas em
`%LOCALAPPDATA%\gobsidian-bench\2026-09-02\` (`antes_<pkg>.test.exe`,
`antes_<pkg>_run<N>.txt`). São os binários "antes" de TODAS as comparações
deste plano — não os recompile; o `benchstat` compara contra eles.

`go test -run '^$' -bench . -benchmem -count=5` por pacote (**5 amostras** —
abaixo das 7 que `desempenho.md` exige, então esta tabela é REFERÊNCIA, não
veredito; toda comparação deste plano roda o binário `antes_*` de novo,
intercalado com o `depois_*`, com `-count=7` ou mais, e é o `benchstat` dessa
rodada que decide). Mediana das 5, via `benchstat antes_all.txt`:

| Benchmark (pacote) | sec/op | B/op | allocs/op |
|---|---|---|---|
| `LinkGraphBothDepth2` (service) | 15,21 µs | 2,06 KiB | 41 |
| `TagListPlano` (service) | 31,78 µs | 13,4 KiB | 9 |
| `TagListHierarquico` (service) | 6,163 ms | 1,03 MiB | 10 880 |
| `NoteListPorTag` (service) | 403,8 µs | 41,5 KiB | 210 |
| `SearchLimit200Cache` (service) | 14,58 ms | 2,38 MiB | 10 170 |
| `SearchTermoAmploCache` (service) | 7,324 ms | 1,94 MiB | 5 594 |
| `SearchFiltroFrontmatter` (service) | 23,20 ms | 4,31 MiB | 30 120 |
| `SearchLimit200CacheTrechoRepetido` (service) | 7,270 ms | 2,11 MiB | 7 654 |
| `IndexBuild` (service) | 352,4 ms | 95,6 MiB | 918 400 |
| `InvertedLoad` (service) | 17,50 ms | 3,51 MiB | 46 380 |
| `SearchTermoAmplo` / `DoisTermos` / `FraseExata` / `Limit200` (service) | 8,705 / 3,658 / 23,27 / 15,41 ms | — | — |
| `SaveIndexCacheReal` (index, com fsync — Task 172) | 22,23 ms ± 15% | 1,17 MiB | 6 103 |
| `TagsSemPrefixo` (index) | 20,02 µs | 7,35 KiB | 8 |
| `ListPorTag` (index) | 856,1 µs | 119 KiB | 12 |
| `BuildComHub` (index) | 115,4 ms | 12,95 MiB | 128 400 |
| `TotalSizeRepetido` (index) | 5,610 µs | 0 | 0 |
| `SaveInvertedCacheReal` (search, com fsync — Task 172) | 163,5 ms ± 30% | 24,95 MiB | 255 000 |
| `EscreveCache` (search) | 24,56 ms | 1,04 MiB | 822 |
| `InvertedUpdateLote` (search) | 6,549 s | 105 MiB | 565 200 |
| `RewriteLinksMuitos` (writer) | 1,496 ms | 3,98 MiB | 795 |
| `ParseNotaLonga` (parser) | 2,611 ms | 1,19 MiB | 9 247 |
| `DetectCandidatesNotaLonga` (parser) | 309,9 µs | 127 KiB | 1 301 |

Saída completa do `benchstat`: `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\antes_all.txt`
(concatenação dos cinco `antes_<pkg>_run*.txt` válidos; `antes_service_run1.txt`
NÃO entra — rodou contra o cofre velho e tem dois FAIL).

**Task 172 (2026-09-06):** `SaveIndexCache` e `SaveInvertedCache` passaram a
gravar via `vault.ReplaceFile`, que inclui `fsync` do arquivo e do diretório
(ver `internal/vault/atomic.go`). As duas linhas acima são a medição NOVA,
depois da mudança — as colunas `ComFsync` da linha anterior mediam um `Sync`
extra aplicado por fora, num caminho de escrita diferente do de produção, e
não são mais comparáveis. `benchstat` intercalado (antes = binário do commit
`011042f`, sem fsync; depois = pós-Task-172), `-test.count=7`:

```
SaveIndexCacheReal-12          17.58m ± 7%   22.23m ± 15%  +26.43% (p=0.001 n=7)
SaveInvertedCacheReal-12       154.1m ± 30%  163.5m ± 30%  ~ (p=0.097 n=7)
```

O índice sobe de forma estatisticamente significativa (+26,43 %), mas fica
abaixo do que a antiga coluna `ComFsync` insinuava (41,87 ms) — a rotina
antiga não é o mesmo caminho de código. O invertido não mostra diferença
estatisticamente significativa nesta rodada (p=0,097): a variância de ±30 %
do benchmark domina o efeito do fsync num arquivo de ~25 MiB. Binários e
saída bruta: `%LOCALAPPDATA%\gobsidian-bench\2026-09-02\{antes,depois}172_{index,search}.test.exe`
e `.txt`.

**Task 180 (2026-09-06):** chave de tag única desde 2026-09 — `index.ChaveDeTag`
(minúscula, NFC, sem `#`) é a conta de `ix.tags` nos três pontos de escrita e em
todos os pontos de leitura, e o casamento hierárquico passou a ser o mesmo em
`note_list`, `vault_search` e `tag_list`. **O formato de cache não muda:** o
cache guarda `Note.Tags` cru e o reload republica por `publishNoteLocked`, que é
onde a dobra mora. `benchstat` intercalado, 7 rodadas alternadas de uma
execução cada (o protocolo de `papeis/desempenho.md`, não `-count=7` por braço),
`-benchmem`; antes = binários do commit `937e54c`:

```
index (antes180_index × depois180_index)
TagsSemPrefixo-12   18.11µ ± 3%   15.85µ ± 12%  -12.47% (p=0.001 n=7)
ListPorTag-12       631.3µ ± 2%   635.2µ ± 10%  ~ (p=1.000 n=7)
  B/op e allocs/op idênticos nos dois (119.1Ki, 12).

service (antes180_service × depois180_service)
TagListPlano-12              19.72µ ± 13%  17.63µ ± 7%   -10.62% (p=0.001 n=7)
TagListHierarquico-12        2.750m ± 21%  3.134m ± 39%  ~ (p=0.097 n=7)
NoteListPorTag-12            235.9µ ± 52%  235.6µ ± 13%  ~ (p=0.535 n=7)
SearchFiltroFrontmatter-12   13.04m ± 50%  13.67m ± 3%   ~ (p=0.128 n=7)
SearchFiltroTags-12          13.36m ± 3%   12.85m ± 8%   ~ (p=0.073 n=7)
  B/op: só SearchFiltroTags move — 1.926Mi -> 1.935Mi, +0,48% (p=0.001 n=7),
  com allocs/op igual (10.15k). É o conjunto de caminhos que o filtro de tag
  agora resolve UMA vez por consulta (index.PathsComTags) no lugar do mapa por
  resultado; o mapa antigo não escapava e saía da pilha, então o que se paga
  são ~9,6 KiB de heap por consulta em troca da semântica correta, sem piora
  de tempo.
```

Uma regressão real apareceu no meio do caminho e foi corrigida antes do commit:
a primeira forma de `candidatosPorTagLocked` ordenava e compactava por tag
também em `tag_mode=any`, que ordena uma vez só no fim — `ListPorTag` deu
**+16,08 % de tempo (p=0,026) e +40,31 % de B/op (p=0,001)**. Com o casamento
anexando à fatia do chamador, voltou a `~` com B/op idêntico (a medição acima).
Binários e saída bruta:
`%LOCALAPPDATA%\gobsidian-bench\2026-09-02\{antes,depois}180_{index,service}.test.exe`
e `t180b_{index,service}_{antes,depois}.txt`.

Um número visível ao usuário muda com a dobra: o subcomando `index` da CLI
reporta `len(idx.Tags("", 1))`, que agora conta **chaves dobradas** — num cofre
onde a mesma tag aparece em duas grafias, a contagem de tags distintas cai. De
quanto, em cofre real: **não medido**.

Fatos medidos sem benchmark:

- `vault_5000`: 112 tags distintas, 0 hierárquicas. Cofres reais do dono: **não medido** neste plano.
- Tag inline em NFD **não chega inteira ao índice** (medido em 2026-09-06, Task 180): `parser.tagNameChar` aceita letra, dígito, `-`, `_` e `/`, e não `unicode.Mn`, então o corpo `#Ação` gravado em NFD (`A c U+0327 a U+0303 o`) indexa como a tag `Ac`. É defeito do parser, anterior à chave única e fora do alcance dela — nenhuma dobra de chave conserta uma tag que já chegou cortada. Pelo frontmatter (`tags: ["Ação"]`) o YAML entrega a string inteira e a dobra funciona: chave `ação`, e um pedido em NFC casa a nota. Aberta como **B20** em `docs/SUGESTOES.md`.
- Cobertura por função (`go test -coverprofile`, 2026-09-02): `construirServico` 6,5 %, `carregarIndiceDoCache` 0 %, `prepararIndiceDeBusca` 0 %, `runServe` 0 %, `serveEmProcesso` 20,6 %, `buildInvertedIndex` 60,7 %; `WriteAtomic` 71,1 %, `SweepStaleTempFiles` 62,5 %, `CleanStaleTempFiles` 0 %; `SaveIndexCache` 64,5 %, `SaveInvertedCache` 56,0 %; `Tags` 91,7 %, `coletarLocked` 87,6 %, `tagListHierarchical` 94,1 %, `LinkGraph` 78,1 %; `Search` 96 %; `mcpsrv.Server.Serve` 0 %, `Close` 0 %; `doctor.checkDaemonLog` 44,4 %, `checkLocksDeDaemon` 58,6 %. Depois da Task 174, `construirServico` é `boot.Montar`, `prepararIndiceDeBusca` é `boot.PrepararBusca` e `buildInvertedIndex` é `boot.construirBusca`; `carregarIndiceDoCache` continua com o mesmo nome, hoje em `internal/boot/indice.go`; `CleanStaleTempFiles` já não existe (ver a nota da linha seguinte).
- Tempo de suíte (`go test ./... -count=1 -cover`): service 51,7 s, writer 32,5 s, search 31,1 s, watcher 26,2 s, index 22,3 s, vault 22,0 s, doctor 19,4 s, mcpsrv 17,0 s.
- Raio de explosão de `service.Index` (gopls references, medido antes da Task 173): `Get` 13, `ResolvePath` 8, `Backlinks` 5, `List` 4, `NotePaths` 2, `TotalSize`/`Tags`/`NoteCount`/`Generation`/`AssetCount`/`AliasCollisions` 1 cada, `Paths` **0**. `service.New` recebe `*index.Index`; a interface foi removida em 2026-09 (Task 173) — tinha uma implementação e nenhum fake.
- Raio de `writer.WriteAtomic` (medido em 2026-09-02, antes das Tasks 171-172): 6 sítios em `internal/service/write.go` (`:149,:248,:378,:619,:751,:832`) + 3 arquivos de teste; `SweepStaleTempFiles`: 1 sítio (`cmd/gobsidian/servico.go:78`). Depois das Tasks 171-172, `writer.WriteAtomic` e `writer.SweepStaleTempFiles` não existem mais — os chamadores usam `vault.WriteAtomic`/`vault.SweepStaleTempFiles` diretamente.
- Prefixo `.gobsidian-tmp-` em 4 literais (medido em 2026-09-02, antes das Tasks 171-172): `writer/atomic.go:14` (constante), `vault/walk.go:74`, `search/persist.go:87`, `index/persist.go:124`. Depois das Tasks 171-172, o literal existe numa só linha: `vault/atomic.go:14`.
- `hits` × `results`: `search --json --limit 200 --vault vault_5000 "execucao"` (binário de 6c5d1f1): `hits` e `results` são 200 itens e **byte a byte iguais** (`hits == results` → `True` em Python). JSON compacto: 195 481 bytes com `hits`, 97 787 sem — **50,0 % do payload é a cópia**. Arquivo indentado: 216 304 bytes. **Depois (Task 179, `hits` removido, commit `b6cf8be`+):** mesmo comando, mais `--max-results 200` (sem a flag, o teto administrativo `MaxResults` — padrão 50 desde a Task 168 — corta a página a 50 itens antes que a comparação faça sentido) e `--cache-dir` novo: 200 itens em `results`, sem `hits`. Arquivo indentado: **108 204 bytes**. JSON compacto: **95 969 bytes** (medido com o mesmo one-liner Python; a pequena diferença ante os 97 787 esperados vem do conteúdo do corpus — snippets e `modified` variam com o estado atual do `vault_5000`, não com o formato).
- CLI a frio × `serve` com cache: cofre `vault_5000`, mesmo binário. `search` a frio (Build + `inv.Update` serial por nota): **10 520 / 10 819 / 11 046 ms** de parede em 3 execuções. `inspect` a frio (só Build): **770 / 767 / 739 ms**. `serve` em processo (`GOBSIDIAN_NO_DAEMON=1 --eager-search`) com cache quente, 5 execuções: `index_ms` **101–123**, índice de busca `duracao_ms` **13–20**, parede boot→saída **475–528 ms**. Ou seja: a CLI de busca paga ~10 s que o `serve` não paga; adotar o cache na CLI (Task 175) tem teto de ganho medido, não estimado.
- Task 175, `search` reaproveitando `boot.AbrirIndice`/`boot.PrepararBusca` como o `serve`: cofre `vault_5000`, `--json --limit 200 "execucao"`, 3 execuções frias + 3 quentes por binário (`Measure-Command`, `TotalMilliseconds`). **Antes** (commit `ebf29ad`, sem `--cache-dir`, constrói tudo em memória sempre): frias **8338,6 / 1687,0 / 1631,9 ms**, "quentes" (mesmo binário, sem cache — repetição idêntica) **1700,4 / 1631,0 / 1597,7 ms**. **Depois**: frias (`--cache-dir` novo, apagado antes de cada uma) **1905,6 / 1921,2 / 1796,7 ms**; quentes (mesmo diretório de cache) **234,7 / 214,0 / 190,2 ms**. A quente do depois fica bem abaixo de 1/5 da fria do depois (1/5 de ~1800–1920 ms é ~360–384 ms; medi 190–235 ms). Os números não batem com a faixa do M4 acima (10 520–11 046 ms de "antes", 475–528 ms de `serve` quente) — a máquina/estado de disco nesta rodada não é o mesmo; a primeira chamada "antes" (8338,6 ms) sugere cache de disco do SO ainda frio nela e já quente nas cinco seguintes. Binário `antes`: worktree Git separado no commit base, sem tocar a árvore principal (`git worktree add --detach`).

**Tasks 182–185 (2026-09-06) — âncoras que resolvem, e a tool que lista o que
sobra.** Cinco commits, `d91b2fb..b34814a`.

- **A separação do `#` virou uma conta só, para os três `LinkKind`** (Task 182).
  `splitAnchor` é a única separação de `#` do parser; o ramo Markdown a chama
  **antes** do percent-decode, então `%23` continua sendo um caractere e não um
  separador. Antes disso só o wikilink separava, e `[x](b.md#Sec)` guardava
  `Target = "b.md#Sec"` — caminho que nenhuma nota tem, logo alvo inexistente.
  Junto veio a segunda metade: **alvo vazio com âncora resolve para a nota de
  origem** (`[[#Seção]]`, `![[#Seção]]`, `[x](#Seção)`), `ok` quando o heading
  existe e `anchor_missing` quando não.

  Medido em 2026-09-06 pelo orquestrador, em **quatro cofres reais**: **zero**
  alvos quebrados contêm `://`, `www.`, `.com`, `.br` ou `.org` — URL com
  esquema já saía por `LinkExternal` e nunca entrou em `broken_links`, então o
  pedido do dono ("retire os links de sites") não tinha o que retirar. Os falsos
  positivos eram os de âncora: no cofre *Estudo*, **267 alvos começando com `#`
  e 10 vazios**; no *Oral*, **372 com `#`**. Que a contagem de `broken_links`
  desses cofres tenha caído depois da correção: **não re-medido**.
- **`IndexCacheParserVersion` 1 → 2** (Task 182, `internal/index/persist.go:49`).
  O portão existe para exatamente este caso: o parser passou a produzir
  estrutura diferente para a mesma entrada, mtime e tamanho dos arquivos não
  mudaram, e sem o bump `VerifyFreshness` não veria motivo para descartar o
  cache — o cofre reabriria com `Target = "b.md#Sec"` e o defeito pareceria não
  corrigido. `IndexCacheFormatVersion` **não** muda: `Anchor` já existia em
  `parser.Link` e já era codificada; o que mudou foi o valor, não o layout.
- **`vault_broken_links`** (Task 183, `internal/service/broken.go`): a lista do
  que `vault_stats` apenas conta. Filtro por `state` e por `prefix` da nota de
  origem, `limit` padrão 100 com teto 500, `total` antes da paginação, ordem
  determinística por `source` e depois pela posição da referência no corpo.
  Externo e resolvido ficam de fora. É a 14ª tool, e **não** toca o índice
  invertido: percorre `index.NotePaths()` e os `Links` de cada nota.
- **Dois consertos em `note_move`** (Tasks 182 e 185), o segundo aberto pela
  revisão do primeiro. Auto-referência **só de âncora** sai da lista de
  referenciadoras — não há alvo escrito para reescrever. Auto-referência **com
  alvo escrito** (`[[a]]` dentro de `a.md`) fica na lista, e passou a ser lida e
  gravada **no caminho novo**: o corpo se move antes do laço, então ler pela
  chave antiga dava ENOENT e `note_move` devolvia `CodeInternal` com o move pela
  metade. Consequência visível no contrato: a nota movida aparece em `rewritten`
  sob o caminho novo e conta em `links_updated` — `docs/TOOLS.md`, `note_move`.

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

## Go 1.27 — o que foi medido

O plano é [`docs/superpowers/plans/2026-09-09-go-1-27.md`](superpowers/plans/2026-09-09-go-1-27.md).
Aqui ficam só os números.

**Malloc especializado por tamanho: abaixo do piso de ruído desta máquina.**
Medido em 2026-09-09 com `GOEXPERIMENT=nosizespecializedmalloc` contra o
padrão — mesmo código, mesma toolchain (go1.27.0), um flag, que é o único jeito
de isolar o efeito sem instalar uma segunda toolchain. Quatro benchmarks de
`service`, `-count=7`, i7-10750H windows/amd64:

| Benchmark | sem | com | delta |
|---|---|---|---|
| `LinkGraphBothDepth2` | 4,679 µs ± 23% | 3,537 µs ± 25% | −24,41% (p=0,002) |
| `TagListHierarquico` | 3,601 ms ± 56% | 5,470 ms ± 49% | ~ (p=0,097) |
| `NoteListPorTag` | 253,2 µs ± 17% | 334,1 µs ± 21% | +31,96% (p=0,004) |
| `InvertedLoad` | 15,34 ms ± 22% | 14,04 ms ± 68% | ~ (p=0,902) |

**Conclusão: inconclusivo, e o número não vai para lugar nenhum.** Os intervalos
de ±23% a ±68% são uma ordem de grandeza maiores que o ~1% que as notas de
lançamento prometem; dois benchmarks apontam para lados opostos com p<0,01, o
que é assinatura de ruído e não de efeito. Publicar "−24%" ou "+32%" daqui seria
escrever número que não se mediu. `B/op` e `allocs/op` saíram **idênticos** nos
quatro, como esperado — a mudança é no custo da alocação, não no tamanho nem na
contagem.

Para virar veredito isto precisa de máquina parada, e a máquina do dono não
estava: a rodada saiu no meio de builds e testes desta mesma sessão.

**`encoding/json` v1 sobre v2: sem diferença observável.** O Go 1.27 passou a
implementar o v1 sobre a máquina do v2. O corpus de round-trip de
`internal/hosts` (`roundtrip_test.go`) roda igual com e sem
`GOEXPERIMENT=nojsonv2`, e as duas medições dizem o mesmo: `Fundir` **atravessa
byte não-UTF-8 sem alterar** — não troca por U+FFFD, que seria corromper o
config do usuário em silêncio — e **aceita chave duplicada preservando
`mcpServers`**. Nenhum pin de `nojsonv2` foi necessário.

**Modernizadores do `go fix`:** `waitgroupgo` toca **11 arquivos**;
`slicesbackward` 1; `atomictypes`, `embedlit` e `unsafefuncs`, 0.

**O 3 que estava publicado aqui era erro de medição, e o mecanismo importa.** A
medição saiu de `go fix -diff` rodado **antes** de a diretiva do `go.mod` subir
de `1.25.0` para `1.27.0`. O modernizador é conservador com a versão de
linguagem declarada; depois do bump ele achou mais oito. Ao aplicar, o `git add`
por caminho explícito staged apenas os três que a medição velha nomeava, e os
outros oito ficaram na árvore por horas sem ninguém notar.

O que escondeu isso não foi o `git add` — foi conferir `git diff --stat` **só
dos arquivos esperados** em vez do diff inteiro. Medição de alcance de
ferramenta automática vale para a árvore no estado em que a ferramenta rodou, e
conferir o resultado exige olhar o que ela mexeu, não o que se esperava que ela
mexesse.
`strings.CutLast` tinha 1 sítio real (`index/resolve.go`); o outro candidato
usava `LastIndexAny` com dois separadores e não converte.

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

**O formato do cache de metadados é o 5**, e este parágrafo dizia "o 3" até
2026-09-06 — conferido contra `internal/index/persist.go:38`, não contra a
memória de quem escreveu. O 3 é de 2026-08-26 (2 até ali) e acrescentou o
`Context` de cada link — o texto ao redor da referência, que `docs/TOOLS.md` já
prometia em `backlinks` e que o código entregava vazio desde sempre (achado A8).
O 4 e o 5 vieram no mesmo dia e **sem mudança de layout**: `contextoBytes` caiu
de 80 para 40 e voltou para 80, quando a medição que motivara o corte foi
retratada. Bump sem mudança de layout é fácil de julgar desnecessário e não é:
um cache gravado com outro `contextoBytes` carrega trechos de outro tamanho, e
aceitá-lo faria a mesma pergunta ser respondida de forma diferente conforme o
cache fosse velho ou novo.

Ao lado dele há um **segundo portão, independente**:
`IndexCacheParserVersion`, que muda quando o parser passa a produzir estrutura
diferente para a mesma entrada. Está em **2** desde 2026-09-06 (Task 182) — ver
o marco acima.

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

Desde a Task 172, `index_cache.gob` e `inverted_cache.gob` são gravados via
`vault.ReplaceFile`, e um arquivo novo nasce `0644` — a mesma postura do
diretório de cache, criado `0755` sob `os.UserCacheDir` (decisão aceita; não é
regressão a corrigir, ver ruling N4 na revisão da Task 172).

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

**Codec de metadados (`internal/index/persist_codec.go`): testes de caixa-branca
em `persist_codec_test.go` (Task 181, 2026-09-06).** Cobertura por função antes →
depois (`go tool cover -func`, `go test ./internal/index/ -coverprofile=...`):

| função | antes | depois |
|---|---|---|
| `escritor.uvarint` | 75,0% | 75,0% |
| `escritor.varint` | 75,0% | 75,0% |
| `escritor.fixed64` | 0,0% | 80,0% |
| `escritor.str` | 75,0% | 75,0% |
| `escritor.boolean` | 100,0% | 100,0% |
| `escritor.timeBlob` | 60,0% | 60,0% |
| `escritor.strSlice` | 100,0% | 100,0% |
| `escritor.headings` | 100,0% | 100,0% |
| `escritor.blocks` | 37,5% | 100,0% |
| `escritor.links` | 100,0% | 100,0% |
| `escritor.inline` | 27,3% | 100,0% |
| `escritor.value` | 65,8% | 97,4% |
| `escritor.note` | 100,0% | 100,0% |
| `escritor.asset` | 100,0% | 100,0% |
| `escreveIndexCache` | 85,7% | 85,7% |
| `leitor.falha` | 0,0% | 100,0% |
| `leitor.uvarint` | 57,1% | 100,0% |
| `leitor.uvarintLivre` | 62,5% | 100,0% |
| `leitor.varint` | 62,5% | 100,0% |
| `leitor.fixed64` | 0,0% | 62,5% |
| `leitor.str` | 77,8% | 100,0% |
| `leitor.boolean` | 100,0% | 100,0% |
| `leitor.timeBlob` | 61,5% | 84,6% |
| `leitor.strSlice` | 75,0% | 83,3% |
| `leitor.headings` | 75,0% | 83,3% |
| `leitor.blocks` | 33,3% | 83,3% |
| `leitor.links` | 75,0% | 83,3% |
| `leitor.inline` | 30,8% | 84,6% |
| `leitor.value` | 61,1% | 94,4% |
| `leitor.note` | 90,9% | 95,5% |
| `leitor.asset` | 83,3% | 83,3% |
| `leIndexCache` | 80,0% | 80,0% |

Pacote inteiro (`internal/index`, todos os testes): 83,5% → 89,8% (rodada de
correção da revisão, 2026-09-06, acrescentou o caso `[]any(nil)` — ver abaixo;
89,7% era o número antes dessa rodada). Não é a média do arquivo —
`go tool cover -func` não dá esse número direto; só a de cada função, coladas
acima.

Achado durante a tarefa, não corrigido (fora do escopo de teste/doc): o branch
`default` de `leitor.value` ("tag de valor desconhecida") é código morto para
qualquer tag além de `valMap` — a leitura da tag já passa por um `uvarint` com
teto `valMap`, então uma tag inválida é recusada ali, com mensagem "acima do
limite", antes de alcançar o `switch`. Ver `docs/SUGESTOES.md` B21.

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

**Quatro gates novos em 2026-09-09, um por afirmação que o projeto fazia sem
provar.** `verify.ps1` passou de 15 para 19 etapas e `check_gates.ps1` de 29
para 41 casos; cada gate entrou com três casos, e os mutantes saem dos arquivos
**vivos**, não de cópias — fixture com cópia de pin ou de grafo envelhece e
reprova pelo motivo errado.

| Gate | O que era afirmação | O defeito que a derrubou |
|---|---|---|
| `check_graph.ps1` | o grafo do `CLAUDE.md` foi "re-extraído dos imports" | terceira redação errada: `selfupdate → config`, e `go list` diz folha |
| `check_pins.ps1` | as versões fixadas concordam | o pin do `golangci-lint` cobrava v2.12.2 dizendo v2.13.2; o gate do release rodou em toolchain diferente do build |
| `check_test_isolation.ps1` | teste não escreve fora do `t.TempDir()` | `t.Setenv("XDG_CACHE_HOME")` desviava `os.UserCacheDir` no Linux e em nenhuma outra plataforma |
| `check_partida.ps1` | "nada roda antes de os mecanismos de encerramento estarem armados" | I/O antes de `boot.VigiarHost`: 2 de 100 ciclos sem `reason=`, nas duas rodadas do CI |

**Um órfão sem diagnóstico custou 200 ciclos para ser classificado.** Em
2026-09-09 o CI deu `1 orfao(s) em 100 ciclos` no `parent-death`, com
`parent-gone: 100x` — a decisão de encerrar estava certa nos 100 —, e o
relatório não dizia se o encerramento **travou** ou apenas **saiu tarde**. As
duas respostas pedem ações opostas, e o log já as distinguia: a linha
`encerramento travou alem do guarda-chuva` só existe no primeiro caso. Foram
precisas mais 200 rodadas para responder o que o log tinha — o re-run do CI
(100/100 nos quatro cenários) e 100 ciclos locais (100/100, `parent-gone:
100x`). **Medido: 1 sobrevivente em 300 ciclos de `parent-death`, com o motivo
certo em 300/300**, no dia em que o job do CI atravessou uma janela ~2× mais
lenta (os ciclos 60→70 levaram 135 s contra ~60 s no resto da rodada). Nenhum
limiar foi mexido: alargar a janela por causa de uma falha é o que o gate
existe para impedir. O que entrou foi o diagnóstico — ao achar um
sobrevivente, o script agora diz qual dos três casos é.

**`test_orphans.ps1` não compila — ele roda o que estiver em `bin/`.** Hoje
recusa binário mais velho que o código. Antes disso, um binário de quatro dias
antes deu três `[OK]` nos cenários que não dependiam do código novo e 100 falhas
de "daemon nao anunciou prontidao" no que dependia — mensagem que aponta para o
daemon quando a causa era o subcomando não existir naquele build. A guarda fica
**antes de qualquer despacho**: a primeira versão dela cobria três cenários e não
o quarto, que é exatamente o defeito que ela existe para impedir.

---

## Dívidas abertas

- **A causa do encerramento pendurado do daemon não foi encontrada.** Medido em
  2026-09-07 na máquina do dono: o daemon PID 42856 registrou `encerramento
  solicitado reason=idle` às 19:30:36, **nunca** registrou `daemon encerrado`, e
  seguia vivo 20 h depois — 274 MB residentes, **0 s de CPU em 3 s** de
  amostragem, 26 threads em `Wait,UserRequest`. Bloqueado, não girando.

  O que se **sabe**: o travamento está **fora** de `lifecycle.Shutdown`, porque
  a guarda dela teria feito `os.Exit(1)` em 6 s e o processo continuou vivo.
  Sobram três esperas — `wg.Wait` em `internal/daemon/daemon.go`, `lc.Wait` e
  `c.Esperar` em `cmd/gobsidian/daemon.go`.

  O que **não** se sabe: qual das três. `dlv attach` responde `could not find
  goroutine array`, porque o binário é compilado com `-s -w`
  (`scripts/build.ps1` e `.github/workflows/release.yml`).

  A Task 189 pôs **anteparo, não conserto**: `lifecycle.ArmarGuardaChuva` cobre
  o intervalo inteiro nos três pontos de saída. O sintoma não pode mais durar
  20 h; a causa continua aberta. Para investigar de novo: compilar sem `-s -w`,
  instalar, e esperar a reprodução.

  **Caminho novo, a partir de 2026-09-09:** o Go 1.27 traz um *goroutine leak
  profile* — um tipo de perfil que reporta goroutine bloqueada num primitivo de
  concorrência que **não pode mais ser desbloqueado**, detectado pelo coletor de
  lixo. É exatamente a forma deste defeito: uma espera que nunca volta. Os
  binários publicados passaram a ser compilados com 1.27.1 (ver
  `.github/workflows/release.yml`), então o perfil está disponível no que o
  usuário roda. Isto **não foi tentado ainda** — é um caminho anotado, não um
  resultado.

  Nota de escopo: o cenário `daemon-idle` de `scripts/test_orphans.ps1` roda
  **100 ciclos no CI e passa**. Ele não pega este defeito — a hipótese, **não
  medida**, é que o cofre sintético do cenário é pequeno demais para exercitar
  watcher e índice reais.
- **O estado de socket que produz `dial 10022` + `remove 1920` não foi
  reproduzido.** `ipc.cleanupSocketFile` falhou 20+ vezes desde 2026-09-01 no
  cofre Estudo com `The file cannot be accessed by the system`, e cada falha
  derrubou toda ponte para o modo em processo. Dois cenários foram medidos em
  2026-09-08 e **nenhum** produz o par observado: listener fechado limpo dá
  `10061` e o arquivo já não existe (o `Close` desvincula no Windows); processo
  morto à força dá `10061` e o `os.Remove` **funciona**.

  A Task 192 pôs **saída, não diagnóstico**: se o caminho não se apaga, o
  arquivo sai do caminho por `rename` — permitido no Windows onde apagar não é,
  medido inclusive sobre executável em uso. E o erro passou a relatar o que
  havia no caminho, que era o que faltava em campo.

- **O pico de memória da reconstrução do índice não tem requisito, por decisão.**
  Com cache frio o `servindo` fica de 5× a 12× acima do alvo de cache quente —
  Estudo 58 → 706 MB, TJSP 127 → 1.180 MB. O RNF-07 passou a nomear "com cache
  válido" (2026-09-01); criar um RNF-07b com teto de 12× foi **descartado pelo
  dono**, porque um alvo igual ao pior caso medido não pode ser violado e
  descreve em vez de exigir. O pico está nos limites conhecidos de `OPERACAO.md`.

  O que atacaria a causa, e não tem prazo nem requisito: construir o índice
  invertido direto na forma achatada, em vez de montar mapas e só depois
  compactar. Mecanismo inferido, **não perfilado**.
- **`measure.ps1` rotula como RNF-01 um número que, com cache quente, é RNF-02.**
  O `index_ms` da partida `pronto` mede carga de cache e é comparado ao teto de
  3.000 ms em vez do de 300 ms. Registrado em `OPERACAO.md`.
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
- ~~**Uma questão que a Task 114 abriu**~~ **decidida em 2026-09-01: fica
  documentada, sem detecção.** Duas notas cujos nomes só diferem na normalização
  Unicode são dois arquivos para o sistema de arquivos e **uma chave só** para o
  índice. O comportamento já está correto desde que `lowerPath` virou lista:
  `ResolvePath` devolve `ErrAmbiguousPath` em vez de escolher uma em silêncio, e
  remover uma não apaga a entrada da outra. **Medido: zero ocorrências** nos cinco
  cofres reais, 5.186 notas, todos NTFS e todos em NFC.

  Decisão do dono: **não vale detectar e avisar.** O custo seria uma varredura a
  mais no boot para um caso que não ocorre, e a resposta já é a certa quando
  ocorre. Registrado em `docs/wiki/entities/note-e-caminho.md`.
- **O hook `scripts/pre_commit_docs.ps1` não gateia nada** (achado do
  implementador da Task 182, 2026-09-06; **corrigido em 2026-09-07** (Task 187):
  o hook lê a mensagem de `-m`/`-F`; `scripts/check_gates.ps1` prova que o
  comentário de shell é recusado). Ele procurava `[sem-doc]` no **texto do
  comando** que o `PreToolUse` recebe, não na mensagem do commit. Um comentário
  de shell na linha do `git commit` já o satisfazia — foi assim que a Task 182
  passou por ele, declaradamente. O hook existe para recusar `.go` de produção
  sem documentação, e qualquer um o desligava sem intenção nenhuma de burlá-lo,
  porque a escotilha casava antes de o gate olhar o que está em stage. A
  escotilha `[sem-doc]` em si é decisão fechada e fica — ver
  `docs/papeis/documentador.md`. A exceção de `--amend`, que era allow
  incondicional e vivia fora do alcance de `-Simular`, foi removida na mesma
  data (rodada 3 da revisão final); amend segue a regra normal.
- **`scripts/audit_reports.ps1` casa a palavra, não a evidência** (achado do
  implementador da Task 184, 2026-09-06; **corrigido em 2026-09-07** (Task 188):
  seção só conta em cabeçalho Markdown; `check_gates.ps1` prova que a prosa que
  nega é sinalizada. Efeito medido em 2026-09-07 sobre 150 relatórios: `79` →
  `199` `SECAO-AUSENTE` — a medição da Task 188 deu `203` com o relatório
  dessa task ainda em curso; o número acompanha o corpus, não o corrija sem
  re-medir. Em 2026-09-07 o glob do auditor passou de `task-*-report.md` a
  `*-report.md`, e os dois `final-fix-report.md` reais entraram: `152`
  relatórios, `203` `SECAO-AUSENTE` — os quatro novos são todos do
  `final-fix-report.md` de broken-links). As quatro seções obrigatórias de um
  relatório eram
  procuradas por regex de palavra solta sobre o corpo inteiro
  (`audit_reports.ps1:113-118`: `red`, `green`, `muta`, `verifica`). A frase
  "**não** há ciclo RED/GREEN nem prova de mutação para colar" satisfazia três
  delas — uma negação explícita passava pelo mesmo portão que uma evidência
  real, e o achado `SECAO-AUSENTE` desaparecia de um relatório que de fato não
  tinha nenhuma das três. Era a **mesma classe** do hook acima: checador que
  casa a palavra em vez do fato não gateia nada. Conserto: exigir um cabeçalho
  Markdown com o nome da seção, e não a palavra em qualquer lugar. Os outros
  achados do script (`HEDGE`, `SHA-FANTASMA`, `MUTACAO-CONDICIONAL`) não têm
  esse defeito e seguem úteis.
- **`[x](b.md#)` — âncora vazia depois do `#` — perde o `#` na reescrita.**
  `splitAnchor` devolve `("b.md", "")` e `anchorMarkdown` devolve `""` para
  âncora vazia, então um `note_move` reescreve `[x](b.md#)` como `[x](c.md)`.
  Fidelidade mínima perdida numa forma que, **medida em 2026-09-07 em cinco
  cofres reais** (Estudo, Jurisprudência, Oral, Revisão, _automacao), tem
  **zero** ocorrências internas: os dois únicos acertos de `\]\([^) ]*#\)` são
  URLs `http://...#mce_temp_url#` em Jurisprudência — externas, que `note_move`
  nunca reescreve. `[[b#]]` deu zero nos cinco.
  **Parqueado por decisão**, não esquecido: não vale um ramo a mais no formatador
  por uma forma cuja frequência medida é zero. Se aparecer, o conserto é
  distinguir "sem âncora" de "âncora vazia" no `parser.Link`, que hoje são a
  mesma coisa.

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
