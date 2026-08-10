# Task 83 — Buscar as postings de cada termo uma vez

**Tier: modelo barato.**

#### Onde encaixa
Depois da 82. Última das quatro que tocam `bm25.go`/`analyzer.go`.

#### O que vincula esta tarefa

Repetido aqui de propósito: o brief é a unidade que viaja, e decisão citada por
código fica no preâmbulo, que não viaja com ela.

- **Otimização que muda resultado é defeito, não trade-off.** O golden de
  ranking da Task 78 (`testdata/ranking/*.tsv`, teste `TestRankingGolden` em
  `internal/service/`) tem de ficar **idêntico**. Golden que muda exige
  explicação escrita e volta para revisão. **Nunca regenerar com `-update` para
  fazer passar** — `-update` grava o que o código produz, não o que está certo.
- **Ordem de acumulação de ponto flutuante não muda.** `CalculateBM25` soma
  `score += idf * tfScore` num laço. Reordenar a iteração muda o arredondamento
  e faz o golden falhar por motivo legítimo; a reação previsível é regenerar, o
  que apaga o gate. Se parecer necessário reordenar, **pare** e escreva por quê.
- **`benchstat` com `-count=6`, uma mudança por vez.** Baseline antes, mudança,
  baseline depois. `~` (sem diferença significativa) **reverte a mudança**:
  código mais feio sem ganho é dívida pura. Colar a saída, não o resumo dela.
- **Teto de latência não é afirmado sob `-race`** (custa 2× a 6×). Asserção de
  tempo fica atrás da constante `raceEnabled`, padrão já existente em
  `internal/service` e `internal/search`.
- **Nenhum teto de RNF é afrouxado nesta batelada.** RNF-04 está em 181 ms
  contra alvo de 100 ms. Alvo não atingido e registrado é informação; alvo
  afrouxado é ficção.

#### Armadilhas já pagas que se aplicam
- **Teste de fallback que deixa o caminho principal ligado mede o caminho
  principal.** Reincidiu duas vezes neste projeto.
- **Chave derivada calculada em dois lugares diverge**, e a divergência aparece
  no caminho menos usado — `[[STJ]]` continuou resolvendo, com `state=ok`, para
  uma nota já removida. Toda chave passa por **uma** função.
- **Campo com valor fixo mente sempre.** `alias_collisions` era `0` literal.
- **Prova de mutação escrita no condicional não é prova.** Tempo verbal no
  passado, com a saída colada.
- **Script Python que edita `.go` converte a sequencia de escape de quebra
  de linha numa quebra literal**, e corrompe a string Go.
  Use `Edit`, não script, para inserir código com escapes.

#### Regras de execução
Rodar `pwsh -File scripts/verify.ps1` antes de dizer que acabou. Registrar no
ledger (`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`) **antes** de
reportar conclusão. Escopo não encolhe em silêncio: se alguma parte não deu,
entregue o resto inteiro e diga o que ficou de fora e por quê — `BLOCKED` com
motivo é resposta melhor que entrega que parece completa.

#### A evidência medida do defeito
```
371.40MB 11.74%  (*Inverted).Postings
```
`CalculateBM25` chama `ix.Postings(m.term)` no laço de pontuação e depois
`ix.Postings(qTok.Raw)` / `(qTok.Reduced)` de novo para montar `docsWithTerm`
do IDF. Mesmo termo, mesma fatia, duas alocações.

#### A decisão que esta tarefa tem de acertar
Montar `docsWithTerm` **na primeira passada**, guardando as postings já obtidas.

**D-M7-2 vale aqui com força.** A tentação é reordenar o laço para aproveitar a
estrutura. Se a ordem de acumulação de `score` mudar, o golden falha por motivo
legítimo — e a reação previsível é regenerar. **Não reordene.** Se parecer
necessário, pare e escreva por quê.

#### Verificações além dos passos
Golden idêntico. Se não for, **não regenerar**: a explicação vira parte do
relatório e a mudança volta para revisão.

#### Prova de mutação
Esta tarefa **não tem prova de mutação**: remove trabalho duplicado sem criar
invariante nova. O que prova que nada quebrou é o golden inalterado.

#### Contrato de relatório
Três coisas, e a primeira é a que o auditor não consegue inferir:

1. **A frase "sem prova de mutação, e por quê"**, explícita. Sem ela o auditor
   não distingue "não se aplica" de "esqueci", e `audit_reports.ps1` sinaliza
   tarefa sem seção de mutação.
2. `TestRankingGolden` verde, com a saída colada, provando que os seis `.tsv`
   ficaram idênticos.
3. `benchstat` de `BenchmarkSearchLimit200` e `BenchmarkSearchTermoAmplo`,
   `-count=6`, antes e depois. Se der `~`, a mudança é revertida e o relatório
   diz isso.

**Files:** `internal/search/bm25.go`
**Commit:** `perf(search): fetch each term's postings once per query`

---

