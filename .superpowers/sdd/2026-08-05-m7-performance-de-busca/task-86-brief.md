# Task 86 — Re-resolução de links dirigida, não global

**Tier: modelo forte.** Maior risco de correção da batelada. Mexe em resolução de link, que já custou caro aqui.

#### Onde encaixa
Última das de código: maior risco, e quer golden e benchmarks já estáveis
para conseguir atribuir regressão.

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
`internal/index/update.go:299`, com o comentário do próprio código:
> `reprocessLinksLocked` *"roda em **todo** evento do watcher, sobre **todas**
> as notas"*

**RNF-06 não atingido**: 20,35 ms contra 20 ms para reindexar um arquivo.


#### A decisão que esta tarefa tem de acertar
O índice reverso cobre links **não resolvidos** também. Um link `[[foo]]`
quebrado tem de estar no mapa sob `foo`, senão criar `foo.md` não conserta nada
— e o sintoma é um link que fica quebrado para sempre até reiniciar.

Aliases contam: criar uma nota com `aliases: [STJ]` afeta todo `[[STJ]]`.

#### Verificações além dos passos
Diferencial contra o caminho global: para uma sequência de eventos (criar,
renomear, apagar, criar de novo com alias), o índice resultante da re-resolução
dirigida tem de ser **idêntico** ao da global. O caminho global fica no teste
como referência, não no produto.

#### Prova de mutação
```
pwsh -File scripts/mutate.ps1 -Path internal/index/resolve.go `
  -Anchor 'for _, alias := range n.Aliases {' -Replacement 'for _, alias := range []string(nil) {' `
  -Test TestReresolucaoDirigidaCobreAliases -Package ./internal/index/

pwsh -File scripts/mutate.ps1 -Path internal/index/update.go `
  -Anchor 'ix.citantesPorNome[nomeChave(alvo)]' -Replacement 'ix.citantesPorNome[alvo]' `
  -Test TestReresolucaoDirigidaIgualAGlobal -Package ./internal/index/
```
A segunda muta exatamente a armadilha do `aliasKey`: chave crua num lado,
normalizada no outro. Se o teste não reprovar, ele não cobre a divergência.

**As âncoras acima nomeiam código que ainda não existe, e isso é deliberado:
elas são o contrato de nomes desta tarefa.** O mapa reverso chama-se
`citantesPorNome`, e a função única que deriva a chave chama-se `nomeChave`. Se
a implementação usar outros nomes, as duas provas não casam âncora e o
`mutate.ps1` sai `2` (inconclusivo) — o que se lê como "não provado", e é.

#### Contrato de relatório
`benchstat` de reindexação de arquivo único no cofre de 5.000 notas.
Dizer se RNF-06 passou a ser atingido, com o número.

**Files:** `internal/index/update.go`, `internal/index/resolve.go`, testes
**Commit:** `perf(index): re-resolve only the links a change can affect`

---

