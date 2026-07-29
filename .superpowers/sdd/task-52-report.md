# Relatório Task 52: Medição Real da Q3, RNF-02 e RNF-04

- **Status**: DONE
- **Commit**: `test(search): measure Q3 on a corpus of distinct paths and a real p95`

## Resumo das Medições Realizadas

### 1. Corpus de Medição
Criado gerador de corpus `geraCorpus(t, 500)` em `internal/search/persist_test.go`, garantindo **500 notas com caminhos DISTINTOS** (`pasta00..09/nota0000..0499.md`) e conteúdos individuais com frontmatter.
Asserção estrita no gerador: `if got := idx.NoteCount(); got != n { t.Fatalf(...) }`.

### 2. Medição da Q3 (PRD §11)
Medição comparativa dos dois cenários sobre o mesmo corpus de 500 notas distintas:
- **(a) LoadInvertedCache (disco):** `26,96 ms`
- **(b) Reconstrução do Invertido (a partir do índice de metadados em memória):** `106,58 ms`

**Decisão da Q3:** Fechada e confirmada a persistência de ambos os caches (`index_cache.gob` e `inverted_cache.gob`). A leitura do disco (a) é 3.95x mais rápida que a reconstrução na memória (b), economizando 79.62 ms no boot frio e mantendo o boot com cache válido dentro do limite do RNF-02 (26,96 ms <= 300 ms).

### 3. Medição do RNF-04 (Latência de Busca p95 Real)
Executada distribuição de 200 consultas BM25 com termos variados sobre o corpus de 500 notas distintas:
- **Mínimo:** `0s` (`0 µs`)
- **Mediana:** `0s` (`0 µs`)
- **p95:** `579,8 µs (0,58 ms)`
(Cumprido com folga em relação ao limite de 100 ms do RNF-04).

## Evidência de TDD

### RED
Comando:
`go test -v ./internal/search/... -run TestQ3PerformanceMeasurement` (com laço antigo de 100 inserções do mesmo caminho)
Saída:
    persist_test.go:174: Q3 Medição: 100 notas | Save: 4.15ms | Load: 15.82ms | Notes: 1
    (O rótulo declarava 100 notas, mas o cache continha apenas 1 nota por causa do caminho duplicado, e a comparação de reconstrução não existia)

### GREEN
Comando:
`go test -v ./internal/search/... -run "TestQ3PerformanceMeasurement|TestRNF04SearchLatencyPercentile"`
Saída:
=== RUN   TestQ3PerformanceMeasurement
    persist_test.go:234: Q3 Medição em 500 notas distintas:
    persist_test.go:235:   (a) LoadInvertedCache (disco): 26.9629ms
    persist_test.go:236:   (b) Reconstruir Invertido (metadados): 106.5805ms
--- PASS: TestQ3PerformanceMeasurement (1.75s)
=== RUN   TestRNF04SearchLatencyPercentile
    persist_test.go:261: RNF-04 Medição de Latência de Busca (200 consultas em 500 notas):
    persist_test.go:262:   Mínimo:  0s
    persist_test.go:263:   Mediana: 0s
    persist_test.go:264:   p95:     579.8µs
--- PASS: TestRNF04SearchLatencyPercentile (0.94s)
PASS
ok  	github.com/jonyd/gobsidian/internal/search	3.465s

## Prova de Mutação

### Amarra entre Rótulo e Tamanho do Corpus (`n -> 1`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/search/persist_test.go -Anchor 'v, idx, inv, _ := geraCorpus(t, n)' -Replacement 'v, idx, inv, _ := geraCorpus(t, 1)' -Test TestQ3PerformanceMeasurement -Package ./internal/search/`
Saída:
--- FAIL: TestQ3PerformanceMeasurement (0.03s)
    persist_test.go:212: header.NoteCount = 1, quer 500
FAIL
[OK] internal/search/persist_test.go restaurado byte a byte (SHA-256 confere).

## Verificações Exigidas pelo Brief
| Verificação | Resultado | Evidência |
|---|---|---|
| Tamanho do corpus afirmado no teste? | SIM | `idx.NoteCount() == n` verificado em `geraCorpus` |
| Dois números da Q3 medidos e comparados? | SIM | (a) 26,96 ms vs (b) 106,58 ms em 500 notas |
| RNF-04 p95 medido com amostra real? | SIM | 200 consultas, p95 = 579,8 µs |
| `TestQ3PerformanceMeasurement` corrigido? | SIM | Substituído laço de caminho único por `geraCorpus` |
| `docs/PRD.md` e `docs/OPERACAO.md` atualizados? | SIM | §11 Q3 e §5 RNF-02/RNF-04 preenchidos com números reais |

## Arquivos Alterados
- `internal/search/persist_test.go`
- `docs/PRD.md`
- `docs/OPERACAO.md`
- `.superpowers/sdd/task-52-report.md`

## O Que Ficou de Fora
Nada.
