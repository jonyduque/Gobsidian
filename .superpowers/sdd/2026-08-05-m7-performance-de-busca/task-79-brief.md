# Task 79 — Checador de artefato citado na doc que não existe no código

**Tier: modelo barato.** Transcrição: a regra está decidida abaixo.

#### Onde encaixa
Segundo. Vale para a doc que a Task 84 vai escrever.

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


#### A evidência medida da lacuna
```
$ grep -rn 'index_cache' --include=*.go .
(vazio)
$ ls "$LOCALAPPDATA/gobsidian/560a6a08c9fa8602/"
inverted_cache.gob
```
`docs/PRD.md` Q3 decidiu persistir `index_cache.gob` **e** `inverted_cache.gob`.
Só o segundo existe. A decisão está escrita como fechada desde 2026-07-29.

#### A decisão que esta tarefa tem de acertar
O checador procura, em `docs/*.md` e `README.md`, tokens em crase que **parecem
identificador de código** e não aparecem em nenhum `.go`:

- nome de arquivo `*.gob`, `*.go`
- token `snake_case` dentro de bloco de parâmetro/JSON
- identificador `CamelCase` com parêntese, ex.: `` `MoveNote()` ``

**Saída é lista, não veredito.** Cada achado é uma frase que precisa de uma
pessoa confirmando — igual a `audit_reports.ps1`. Sai `1` com achados.

**Ruído mata checador.** Rodar contra o repositório inteiro e olhar o volume
antes de aceitar a regra: a primeira versão do checador de briefs sinalizou uma
linha que **negava** ter placeholder. Se passar de ~20 achados legítimos-mas-
irrelevantes, restringir o padrão, não aceitar o barulho.

#### Verificações além dos passos
Prova de que o instrumento pega: acrescente `` `create_dirs` `` a
`docs/TOOLS.md`, rode, confirme que aparece, **remova**. Colar as duas saídas.

#### Contrato de relatório
Volume total de achados no repositório hoje, e a lista. Prova de disparo acima.
Esta tarefa **não tem prova de mutação** — o entregável é um script PowerShell,
e `mutate.ps1` roda teste Go. A prova equivalente é o disparo controlado acima.

**Files:** `scripts/check_doc_refs.ps1`
**Commit:** `test(docs): flag doc references to code artifacts that do not exist`

---

