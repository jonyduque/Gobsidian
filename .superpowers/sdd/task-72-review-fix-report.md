# Relatório: passe de correção sobre a Task 72 (e dois achados vizinhos)

**Data:** 2026-08-01
**Origem:** revisão pelo modelo principal do lote M6 entregue pelo modelo barato
(Tasks 75, 69, 74, 70, 71, 72). Cinco das seis passaram; a 72 não.
**Máquina de todas as medições:** maquina de referencia, 12 núcleos, Windows 11.

---

## O que a revisão encontrou

O relatório da Task 72 afirmava "10/10 etapas verdes, provas de mutação
validadas, nada ficou de fora, nenhum defeito fora do escopo". A bateria estava
mesmo verde — isso confere. O resto não.

### 1. Cancelamento de `ctx` devolvia página parcial como sucesso

A goroutine que perdia o `select` para `ctx.Done()` retornava sem escrever seu
slot, e o filtro final (`res.Path != ""`) descartava o slot vazio. Sonda, mesmo
corpus, mesma consulta, `Limit: 200`:

```
REFERENCIA:       err=<nil> len(Results)=200 Total=500
JA CANCELADO  #0: err=<nil> len(Results)=82  Total=500 Truncated=true
JA CANCELADO  #1: err=<nil> len(Results)=99  Total=500 Truncated=true
JA CANCELADO  #2: err=<nil> len(Results)=84  Total=500 Truncated=true
MEIO DO VOO   #0: err=<nil> len(Results)=88  Total=500 Truncated=true
MEIO DO VOO   #1: err=<nil> len(Results)=24  Total=500 Truncated=true
MEIO DO VOO   #2: err=<nil> len(Results)=47  Total=500 Truncated=true
```

Contagem diferente a cada chamada, `err == nil`, `Total` intacto. O cliente não
tinha como distinguir isso de uma busca que achou pouco.

**É regressão, não limitação herdada.** No laço sequencial anterior à Task 72 não
havia verificação de `ctx` nenhuma e o hit era acrescentado incondicionalmente:
um `ctx` cancelado devolvia os 200. A tarefa de desempenho apagou dado que o
commit anterior entregava.

### 2. A prova de mutação do relatório não reproduz

Comando exato do relatório da Task 72, rodado de novo:

```
[...] Mutando internal/service/search.go
      - if workers <= 1 {
      + if true {

[...] go test -race -run TestRNF04SnippetConcurrencyLimit200 ./internal/service/
----------------------------------------------------------------------
ok  	github.com/jonyd/gobsidian/internal/service	8.341s
----------------------------------------------------------------------
[OK] internal/service/search.go restaurado byte a byte (SHA-256 confere).

[!] O teste PASSOU com a regra mutada.
MUTATE_EXIT=1
```

O relatório colou um `FAIL`. Esse `FAIL` é impossível: `mutate.ps1` roda com
`-race`, sob `-race` a constante `raceEnabled` é `true`, e a asserção do teste é
`if !raceEnabled && p95 > 60*time.Millisecond`. Compilada fora, ela não pode
reprovar. `mutate.ps1` **já tinha `-NoRace`** desde antes, documentado apenas
para deadlock; a ferramenta certa existia e não foi usada.

### 3. O teto de 60 ms não era cobrado em lugar nenhum

`verify.ps1` etapa 2 roda `-race` (asserção compilada fora). Etapa 3 roda sem
`-race`, mas com `-run "TestRNF04VaultSearchLatencyP95"` — o teste novo não
estava na lista. É o mesmo defeito que o commit `67016de` fechou dois dias antes,
reintroduzido na mesma semana.

### 4. `TestRNF04SnippetParity` não comparava nada

Ele afirmava "200 resultados, `Path` e `Snippet` não vazios". Prova por mutação,
invertendo a ordem dos 200 resultados:

```
[...] Mutando internal/service/search.go
      - results[idx] = SearchHit{
      + results[len(results)-1-idx] = SearchHit{

[...] go test -race -run TestRNF04SnippetParity ./internal/service/
ok  	github.com/jonyd/gobsidian/internal/service	4.497s

[!] O teste PASSOU com a regra mutada.
```

O plano exigia comparar a saída completa antes e depois. Um teste chamado
*Parity* que sobrevive à inversão da ordem dos resultados reporta cobertura que
não existe.

### 5. `docs/OPERACAO.md` não foi atualizado

O plano manda, em negrito: *"remeça TAMBÉM no cofre de 5.000 e registre os dois
números em `docs/OPERACAO.md`"*. O commit `6c301f2` tocou três arquivos, nenhum
deles doc. O documento seguia dizendo `414,09 ms` e *"otimização de I/O
concorrente de trechos permanece como oportunidade de melhoria futura"* — falso
depois da própria tarefa que o relatório descrevia.

---

## Evidência de TDD

### RED

Antes de tocar em `search.go`, a sonda de cancelamento (seção 1) e as três
mutações abaixo estabeleceram o vermelho: `Search` devolvia página parcial com
`err == nil`; `TestRNF04SnippetParity` sobrevivia à inversão dos 200 resultados;
e o comando de mutação colado no relatório da Task 72, rodado de novo, saía
`MUTATE_EXIT=1` — o oposto do que o relatório afirmava.

### GREEN

Depois da correção, os três testes passam, e as três provas de mutação saem
`MUTATE_EXIT=0` — cada uma reprovando quando a regra que ela nomeia é removida.
A bateria fecha em `VERIFY_EXIT=0`, 10 de 10 etapas.

## O que a correção fez

### `internal/service/search.go`

**Um único construtor de `SearchHit`.** Havia duas cópias do corpo, uma no ramo
sequencial e outra no concorrente. Agora existe `montaSlot`, chamada pelos dois.
Corpo duplicado diverge, e a divergência aparece no caminho menos usado — foi
exatamente assim que `byAlias` quebrou na Task 33.

**Slot explícito em vez de sentinela.** `slots[i].ok` diz se o slot foi
preenchido. `Path == ""` confundia "nota sumiu do índice entre o filtro e o
recorte" com "goroutine não rodou".

**Cancelamento vira erro.** Depois do `wg.Wait()`, `if err := ctx.Err(); err != nil { return SearchResult{}, err }`.
Página parcial não sai como sucesso. Os dois chamadores já tratavam erro
(`internal/mcpsrv/tools_read.go:57` devolve `toolErr(err)`;
`cmd/gobsidian/search.go:50` propaga).

**`maxSnippetWorkers` passou de 16 para 8, por medição.** Ver a próxima seção.

### O número que justifica o pool

O plano pedia a escolha "com o número que justifique". A varredura, p95 de
`limit: 200`:

| workers | 500 notas, ociosa | 500 notas, 4 cópias | 5.000 notas, ociosa |
|---|---|---|---|
| 1 (sequencial) | 69,2 ms | 79,8 – 90,8 ms | 561,8 ms |
| 4 | 35,9 ms | 82,7 – 105,4 ms | 226,8 ms |
| **8** | **30,7 ms** | 85,0 – 136,3 ms | **177,2 ms** |
| 16 (entregue) | 35,2 ms | 100,1 – 138,1 ms | 217,7 ms |

Três coisas que a varredura mostra:

1. **Mais trabalhadores não é monotonicamente melhor.** 16 é pior que 8 nas duas
   escalas; 12 núcleos não absorvem 16 leitores.
2. **A 500 notas o ganho satura em 4.** 16 não é melhor que 4 com a máquina
   livre — ou seja, o valor entregue não comprava nada no único regime em que
   foi medido.
3. **Sob quatro cópias, qualquer pool fica pior que o sequencial.** Mas quatro
   cópias com pool de 8 põem 32 leitores em 12 núcleos, e um servidor só nunca
   faz isso. O harness de quatro cópias era proxy honesto de "máquina ocupada"
   quando o código era sequencial; depois desta otimização virou proxy de
   "quatro vezes a nossa própria concorrência". **Registrado como lacuna:** não
   há hoje harness que carregue a máquina sem multiplicar o pool do servidor.

### `internal/service/search_test.go`

`TestRNF04SnippetParity` foi reescrito com oráculo real: para cada posição `i`
do resultado concorrente (`Limit: 200`), compara os seis campos com
`Search(Limit: 1, Offset: i)` — uma página de um resultado tem
`min(1, maxSnippetWorkers) == 1` e cai no ramo sequencial. Mesmo ranking, mesmo
corpus, mesma função de montagem, produzidos sem goroutine nenhuma.

`TestSearchCtxCanceladoNaoDevolvePaginaParcial` é novo: exige `err` não-nulo,
`errors.Is(err, context.Canceled)` e zero resultados junto do erro.

### `scripts/verify.ps1`

Etapa 3 passou a rodar `TestRNF04VaultSearchLatencyP95|TestRNF04SnippetConcurrencyLimit200`.
Todo teto novo entra nessa lista no mesmo commit em que nasce.

`$Alvos` ganhou `./tools/...` e `$DirsFmt` ganhou `tools`. Até a Task 74,
`tools/netcheck` era um analisador que ninguém chamava; a partir dela o gate do
RNF-30 depende dele. Fora de `$Alvos`, era o único código Go do repositório que a
bateria não compilava, não vetava, não lintava e cujo teste nunca rodava. A
inclusão achou um achado real de imediato:

```
tools\netcheck\cmd\netcheck\main.go:1:1: package-comments: should have a package comment (revive)
```

### `scripts/mutate.ps1`

A documentação de `-NoRace` cobria só deadlock. Agora nomeia o segundo caso —
teste de teto de tempo, cuja asserção o `-race` compila fora — e cita a Task 72
como o precedente que produziu saída de falha impossível.

---

## Provas de mutação da correção

### 1. A paridade detecta reordenação

```
[...] Mutando internal/service/search.go
      - montaSlot(i, h)
      + montaSlot(len(slots)-1-i, h)

[...] go test -race -run TestRNF04SnippetParity ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestRNF04SnippetParity (1.67s)
    search_test.go:549: posição 0: Path concorrente = "pasta04/nota0494.md", sequencial = "pasta01/nota0001.md"
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	2.951s
FAIL
----------------------------------------------------------------------
[OK] internal/service/search.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
MUTATE_EXIT=0
```

### 2. O retorno de erro no cancelamento é cobrado

```
[...] Mutando internal/service/search.go
      - if err := ctx.Err(); err != nil {
      + if err := ctx.Err(); err != nil && false {

[...] go test -race -run TestSearchCtxCanceladoNaoDevolvePaginaParcial ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestSearchCtxCanceladoNaoDevolvePaginaParcial (1.36s)
    search_test.go:596: rodada 0: ctx cancelado devolveu err == nil com 0 de 200 resultados; página parcial não pode sair como sucesso
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	2.415s
FAIL
----------------------------------------------------------------------
[OK] internal/service/search.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
MUTATE_EXIT=0
```

### 3. O teto de 60 ms é cobrado — desta vez com `-NoRace`

```
[...] Mutando internal/service/search.go
      - const maxSnippetWorkers = 8
      + const maxSnippetWorkers = 1

[...] go test  -run TestRNF04SnippetConcurrencyLimit200 ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestRNF04SnippetConcurrencyLimit200 (8.44s)
    search_test.go:520:   limit: 200 concorrente   mediana 75.7824ms    p95 86.4632ms    teto 60ms
    search_test.go:526:   limit: 200 concorrente   rodada 1/3 estourou (86.4632ms > 60ms); repetindo
    search_test.go:520:   limit: 200 concorrente   mediana 82.6401ms    p95 102.8026ms   teto 60ms
    search_test.go:526:   limit: 200 concorrente   rodada 2/3 estourou (102.8026ms > 60ms); repetindo
    search_test.go:520:   limit: 200 concorrente   mediana 84.5574ms    p95 91.6396ms    teto 60ms
    search_test.go:531: p95 de limit: 200 = 91.6396ms excede o teto de 60ms em 3 rodadas seguidas — carga transitoria nao sobrevive a 3 rodadas, entao o recorte concorrente nao esta ativo
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	9.856s
FAIL
----------------------------------------------------------------------
[OK] internal/service/search.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
MUTATE_EXIT=0
```

Vale reparar que a mutação **reprovou nas três rodadas**. É a forma mais forte
desta prova: mostra que a repetição não cria folga para uma regressão de código,
só absorve pico de carga.

### 3b. O teto precisou do laço de repetição, e isso apareceu no gate

Ao entrar em `verify.ps1`, o teste reprovou na primeira bateria completa:

```
[...] 3. go test (RNF-04, sem -race)
WARNING: [!] go test (RNF-04, sem -race)
     --- FAIL: TestRNF04SnippetConcurrencyLimit200 (2.15s)
         search_test.go:509: p95 com Limit: 200: 63.1721ms
         search_test.go:511: p95 com Limit: 200 foi 63.1721ms, excede a meta de 60ms
```

Antes de mexer no teto, medi se era transiente ou sistemático:

```
rodada 1: p95 com Limit: 200: 33.5985ms      solo 1: p95 com Limit: 200: 30.1844ms
rodada 2: p95 com Limit: 200: 31.9229ms      solo 2: p95 com Limit: 200: 38.2705ms
rodada 3: p95 com Limit: 200: 32.6966ms      solo 3: p95 com Limit: 200: 55.5962ms
```

Transiente: a medição varia de 30 a 56 ms, e a bateria acabara de rodar
`go test -race` sobre tudo. Uma asserção de tiro único a 60 ms ia piscar. A
correção foi adotar o mesmo laço de repetição de `TestRNF04VaultSearchLatencyP95`
— **não** afrouxar o teto, que o plano proíbe explicitamente e que apagaria o
sinal.

---

## Medições finais

**500 notas, ociosa, sem `-race`** — os oito formatos:

```
  termo amplo, limit default     mediana 6.2588ms     p95 10.94ms      teto 100ms
  dois termos                    mediana 7.695ms      p95 13.9683ms    teto 100ms
  termo seletivo                 mediana 1.0219ms     p95 7.2881ms     teto 100ms
  filtro de pasta                mediana 6.0143ms     p95 10.8859ms    teto 100ms
  filtro de tag                  mediana 4.6818ms     p95 9.0426ms     teto 100ms
  frase exata                    mediana 14.8696ms    p95 25.2235ms    teto 100ms
  trecho maximo                  mediana 8.4298ms     p95 14.477ms     teto 100ms
  limit maximo do schema         mediana 22.8632ms    p95 25.477ms     teto 100ms
  p95 com Limit: 200: 28.1567ms
```

**Meta da Task 72 (p95 de `limit: 200` abaixo de 50 ms a 500 notas): atingida,
com 25,48 ms.** Nenhum dos outros sete formatos piorou.

**500 notas, `limit: 200`, quatro cópias**, cinco rodadas de quatro: **73,0 a
136,3 ms**. Última rodada: 73,0 / 84,4 / 91,5 / 94,8 ms. Sequencial no mesmo
harness: 79,8 – 90,8 ms. Ver a ressalva sobre o que esse harness passou a medir.

**5.000 notas** (cofre de `gen_vault.ps1 -Notes 5000 -Seed 42`):

```
  RNF-01  Min: 486.5284ms, Mediana: 500.1096ms, Max: 529.196ms      teto 3s     ATINGIDO
  RNF-02  Min: 77.2884ms,  Mediana: 96.9407ms,  Max: 106.2672ms     teto 300ms  ATINGIDO
  RNF-07  RSS 66,33 / 67,08 / 67,44 MB quente; 107,58 / 108,90 / 112,96 MB frio
                                                                    teto 60MB   NAO ATINGIDO
  termo amplo, limit default     mediana 71.6576ms  p95 94.5359ms   teto 100ms  ATINGIDO
  dois termos                    mediana 17.0534ms  p95 20.5952ms               ATINGIDO
  termo seletivo                 mediana 10.8662ms  p95 16.4048ms               ATINGIDO
  filtro de pasta                mediana 75.405ms   p95 92.3992ms               ATINGIDO
  filtro de tag                  mediana 76.6553ms  p95 92.5528ms               ATINGIDO
  frase exata                    mediana 53.1476ms  p95 64.8953ms               ATINGIDO
  trecho maximo                  mediana 21.6583ms  p95 30.9845ms               ATINGIDO
  limit maximo do schema         mediana 164.231ms  p95 181.2545ms  teto 100ms  NAO ATINGIDO
```

Antes da Task 72, no mesmo cofre e na mesma máquina: `limit: 200` p95 **561,81
ms**, `termo amplo` p95 **140,48 ms**. Sete formatos de oito passaram a caber no
teto. `limit: 200` caiu 68% e continua **81% acima** dos 100 ms.

---

## Achado vizinho: RNF-07 estava medido na grandeza errada (Task 71)

`docs/OPERACAO.md` registrava `Alloc: 29,12 MB` de `runtime.MemStats` e marcava
**"OK no Heap Alloc"**. RNF-07 é **RSS** — o working set do processo —, que
inclui o runtime do Go, os stacks das goroutines, o binário mapeado e os spans
já devolvidos pelo alocador mas ainda residentes. `Alloc` é uma fração disso; a
mesma linha trazia `Sys: 120,55 MB` entre parênteses, o dobro do teto, sem que
isso mudasse o veredito.

Medido no processo real (`gobsidian serve`, `Process.WorkingSet64`, cinco
amostras a 500 ms depois de 8 s de repouso): **67,08 MB com cache quente,
112,96 MB a frio**, contra teto de 60 MB.

O instrumento foi conferido contra um cofre de 100 notas: **20,97 MB**, coerente
com os 18,9–19,3 MB históricos do cofre de 7 notas. Isso descarta erro de escala
no instrumento e sustenta que o número de 5.000 é real.

**RNF-07 a 5.000 notas não está atingido.** Fica em aberto.

---

## Bateria

`pwsh -File scripts/verify.ps1`, rodada isolada, com `tools/` já incluído:

```
Carregado em 350ms
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. go test (RNF-04, sem -race)
[OK] go test (RNF-04, sem -race)
[...] 4. go vet (windows)
[OK] go vet (windows)
[...] 5. go vet (linux)
[OK] go vet (linux)
[...] 6. go vet (darwin)
[OK] go vet (darwin)
[...] 7. gofmt
[OK] gofmt
[...] 8. golangci-lint
[OK] golangci-lint
[...] 9. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 10. check_tool_params
Carregamento: 943,6 ms 

[OK] check_tool_params

[OK] Bateria completa. Pode commitar.
VERIFY_EXIT=0
```

## O que ficou de fora

- **Fechar RNF-07 a 5.000 notas.** Medido e registrado como não atingido; reduzir
  a pegada é trabalho de otimização que ninguém pediu nesta tarefa e que mudaria
  estruturas de dados do índice.
- **Fechar RNF-04 para `limit: 200` a 5.000 notas.** Caiu de 561,81 para 181,25
  ms; os 100 ms exigiriam atacar o custo por resultado, não a concorrência.
- **Harness de carga que não multiplique o pool do servidor.** A lacuna está
  nomeada em `docs/OPERACAO.md` e aqui.
