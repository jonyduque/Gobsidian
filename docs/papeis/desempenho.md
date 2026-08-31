# Papel: melhorador de desempenho

Você vai tentar deixar algo mais rápido ou mais leve.

**A regra que governa tudo aqui: nenhuma mudança de desempenho entra sem
baseline.** Rodar a bateria ANTES, declarar a regra de decisão antes de medir, e
aceitar `~` como resposta final. **Mudança sem ganho significativo é dívida
pura** — precedente Task 82, revertida por isso.

A skill `go-performance` traz a metodologia (go-perfbook); `golang-benchmark`
traz o instrumento. Este documento traz o que já foi medido aqui, para não se
re-medir, e as armadilhas do próprio instrumento.

---

## O laço

```bash
# 1. cofre sintético determinístico (sem ele o bench PULA, e ausência = erro)
pwsh -File scripts/gen_vault.ps1 -Notes 5000 -Seed 42 -Out $env:TEMP\vault_5000

# 2. DOIS binários, construídos ANTES de medir qualquer um dos dois
go test -c -o antes.exe ./internal/service/     # com a árvore no estado anterior
# ... aplique a mudança ...
go test -c -o depois.exe ./internal/service/

# 3. INTERCALE: uma rodada de cada, alternando, 7 ou mais vezes
for i in $(seq 7); do
  ./antes.exe  -test.bench=BenchmarkSearchLimit200Cache -test.run='^$' -test.benchmem >> antes.txt
  ./depois.exe -test.bench=BenchmarkSearchLimit200Cache -test.run='^$' -test.benchmem >> depois.txt
done
benchstat antes.txt depois.txt

# 5. comparacao POR BENCHMARK contra o baseline versionado
pwsh -File scripts/bench_compare.ps1
```

`scripts/bench_compare.ps1` compara contra `docs/bench-baseline.json`, **por
benchmark, nunca agregado**: um agregado esconde a regressão de um caminho atrás
da melhora de outro.

### Por que INTERCALAR, e não `-count=8` de um lado e depois do outro

Rodar todas as amostras de um braço e depois todas do outro mede a **deriva da
máquina** junto com a diferença entre os braços. Em 2026-08-27 isso produziu
**dois números errados publicados e depois retratados** neste projeto. O segundo
dizia que um campo novo empurrava o RNF-02 de atingido para não atingido —
medianas de 243 ms contra 323, faixas quase disjuntas, sinal aparentemente forte.
Refeito com os binários lado a lado e uma rodada de cada por vez, as três
variantes ficaram em 179 / 193 / 191 ms, com a diferença entre elas **menor que a
variação dentro de qualquer uma**.

**Alternar a ORDEM das bateladas não corrige** — foi o cuidado tomado na segunda
tentativa, e não bastou: as bateladas continuam separadas no tempo, e o cache de
arquivo do SO aquecendo, outro processo ou throttling térmico atingem cada uma
de forma diferente.

Sinal de alerta específico: **se a variante medida PRIMEIRO for sistematicamente
a mais lenta, suspeite do cache de arquivo do SO antes de acreditar.** O primeiro
boot contra um cofre de 1,3 GB mediu 4.534 ms; a repetição com o SO quente, 765.

### Meça o que o requisito nomeia, não o que é fácil de medir

Três vezes nesta base a medição estava certa e a **métrica** errada:

- **RSS não mede quanto dado se guarda** — acompanha a meta de heap do GC, e
  inverteu de sinal entre dois binários. Para memória, use o heap vivo do
  `gctrace` (`antes->pico->vivo`), não RSS e nem `MemStats.Alloc`, que é heap
  vivo MAIS lixo não coletado.
- **`index_ms` deixou de cobrir o boot inteiro** quando a varredura de
  temporários passou a rodar em paralelo com a carga do índice. Pelo `index_ms`
  a mudança era regressão de 23%; pelo relógio até servir, ganho de 26%.
- **Perfil velho descreve código velho.** O perfil que punha o BM25 em 0,73% era
  anterior a uma mudança que cortou 87% do tempo total; refeito, ele valia 16% da
  CPU e **79% da alocação**.

---

## Perfilar antes de otimizar

Otimizar sem perfil escolhe o alvo errado com confiança. Exemplo real, medido em
2026-08-25 em `BenchmarkSearchLimit200` (cofre de 5.000 notas):

| Caminho | CPU (cum) | Alocação |
|---|---|---|
| `GenerateSnippet` (I/O: `os.Open` por hit) | **21,26%** | 3,9% |
| `CalculateBM25` | 10,21% | **40,08%** |
| `getFieldWeight` | 1,34% | — |
| `idx.Paths` | 0,85% | 2,70 MB |

Linha a linha dentro do BM25, as quatro maiores alocações somam 153 MB = **34%
de toda a alocação do benchmark**, e são todas `map[string]…`:

```
53.26MB   docTermFreqs[p.Path] = make(map[int]float64)
48.01MB   docTermFreqs[p.Path][m.queryIdx] += fw*mult
28.20MB   docsWithTerm[p.Path] = true
23.59MB   docScores[path] = score
```

Três conclusões que só o perfil dá:

1. **IDs densos pagam em alocação, não em latência.** O BM25 é 10% do CPU: mesmo
   zerado, a latência mal se move.
2. **O gargalo de latência é o trecho, e é I/O.**
3. **Itens que pareciam otimizações valem ruído.** "Re-tokeniza os termos por
   hit" era falso na prática: das alocações de `Analyze`, **96,58% vêm do setup
   de indexação e só 2,50 MB da busca**.

---

## Armadilhas do instrumento

**Cache ligado no harness transforma o requisito em outra pergunta.** Os laços
de RNF-04 repetem a MESMA consulta 30 vezes, e `b.Loop` faz o mesmo — um cache de
trecho ligado acerta 29 das 30. Medido: `limit: 200` deu 21,49 ms quente contra
25,98 ms frio, e o quente teria entrado na documentação como "RNF-04 atingido"
pelo motivo errado. **Todo harness de latência desliga o cache por
`semCacheDeTrecho`**; quem mede repetição é um benchmark com "Repetido" no nome.

**Distribuição com mediana zero está medindo o relógio, não o código.** Uma
medição publicou `Mínimo: 0s, Mediana: 0s` ao lado de um p95, porque consultava
termos que casavam **uma** nota em 500. Se metade das amostras dá zero, a carga
não existe. E p95 sobre formatos de custo diferente não é p95 de nada: vira o
percentil do formato mais caro presente. **Meça por formato.**

**Meça pela camada que o requisito nomeia.** RNF-04 diz "latência de
`vault_search`" — não de `CalculateBM25`. A diferença entre as duas leituras foi
de 0,58 ms para 6–174 ms.

**Não rode gate concorrente com medição.** Um mata os processos do outro e
produz falso verde.

**Pipe engole código de saída.** Redirecione para arquivo e leia o `$?`.

---

## O que já foi medido e não se re-litiga

Ver [`../ESTADO.md`](../ESTADO.md) para os números completos. Resumo do que está
**fechado**:

- **`GOGC`**: rejeitado duas vezes, com estatística. O que pagou foi
  `debug.FreeOSMemory()` depois do índice pronto: −195 MB.
- **Transporte IPC (D-M7-6)**: AF_UNIX contra named pipe, medido em três
  tamanhos; AF_UNIX ganha em todos.
- **Formato de cache 5 e 6**: carregamento 5,59 s → 659 ms, arquivo 482 → 67 MB.
- **Busca (M7 Parte I)**: 218,5 → 115,0 ms, alocação −73%, allocs −89%.
- **Task 82**: revertida. `benchstat` deu `~`.

---

## Alvo não atingido é informação; alvo escondido é dívida

Quando o número estoura, **não use `t.Skip` e não afrouxe o alvo dos outros
casos**. Faça o teste cobrar um **teto medido** naquele caso, escreva ao lado
que o alvo do RNF não está atingido, e registre a lacuna em
[`../ESTADO.md`](../ESTADO.md). O teto guarda contra piorar enquanto a lacuna
espera tarefa.

E se não mediu, escreva **"não medido"**. Ninguém vai brigar com isso.
