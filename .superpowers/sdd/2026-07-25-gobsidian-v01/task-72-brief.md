### Task 72: A folga fina do RNF-04 em `limit: 200`

**Onde encaixa.** Lacuna medida e registrada em 2026-08-01, em `docs/OPERACAO.md`.

**O número, medido em 2026-08-01 no maquina de referencia (12 núcleos, Windows 11), 500 notas:** o formato `limit: 200` — o máximo do schema — tem mediana de 61–76 ms e p95 de 81 ms contra o teto de 100 ms do RNF-04. Rodando **quatro cópias** do binário de teste ao mesmo tempo, o p95 foi a **100,6 / 102,9 / 107,4 ms** em três das quatro. Com **oito cópias**, as oito estouraram, entre 111 e 126 ms. Os outros sete formatos têm de 3x a 20x de folga.

**A causa, já identificada:** a geração de trecho lê do disco **uma vez por resultado**. Com `limit: 200` são 200 leituras sequenciais, e é isso que domina a latência.

**O que NÃO fazer.** Não afrouxe o teto. `TestRNF04VaultSearchLatencyP95` já repete a medição até 3 vezes antes de reprovar, para separar pico de carga de regressão — e isso **não cria folga**, só evita falso vermelho. Aumentar o teto apagaria o sinal que ele existe para dar.

#### Passos

Reduza o custo. Duas direções, e a escolha é sua **com o número que justifique**:

1. **Leitura concorrente dos trechos** — um pool limitado lendo os N resultados em paralelo. Respeite `ctx`: é I/O de arquivo, e a regra do projeto é que função que pode bloquear recebe `ctx` e o respeita.
2. **Cache de corpo por caminho + mtime**, invalidado pelo watcher.

**Meta.** p95 de `limit: 200` **abaixo de 50 ms** com a máquina ociosa, o que dá 2x de folga e sobrevive à carga que hoje o derruba. Se chegar só a 65 ms, **registre 65 ms** e diga que a meta não foi atingida — não mude a meta.

#### Verificações além dos passos

- Reproduza a condição de carga antes e depois: quatro cópias do binário de teste ao mesmo tempo. Cole os dois conjuntos de números.
- Os outros sete formatos **não podem piorar**. Feature P1 não tem direito de apagar dado P0, e otimização que troca um formato por outro é a mesma coisa.
- Se usar concorrência, `go test -race` tem de passar. O detector já achou uma corrida real neste projeto em 2026-08-01 (`index.MoveNote` contra `service.Search`).
- Os trechos gerados têm de ser **idênticos** aos de antes. Leitura concorrente que muda o resultado não é otimização, é defeito. Compare a saída completa de uma busca antes e depois.

#### Prova de mutação obrigatória

Desligue a otimização (volte ao caminho sequencial) e confirme que um teste nomeia a diferença — nem que seja o próprio teto medido. Cole a saída de `scripts/mutate.ps1`.

#### Regras de execução

Idênticas às da Task 69. Atenção adicional: `ctx` onde há espera real, e nunca `ctx` que nenhum corpo verifica — quando o parâmetro existir só por consistência de assinatura, nomeie-o `_`.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-72-report.md`: os oito formatos antes e depois, ociosa e sob carga de quatro cópias; a comparação dos trechos gerados; a saída de `go test -race`; a prova de mutação; e se a meta de 50 ms foi atingida — com o número, não com uma avaliação.

Responda com no máximo 15 linhas.

**Files:** Modify `internal/search/snippet.go`, `internal/service/search.go`
**Commit:** `perf(search): cut snippet I/O cost for large result limits`

---

