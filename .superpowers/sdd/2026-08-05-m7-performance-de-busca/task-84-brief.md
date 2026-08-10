# Task 84 — `note_read` aceitando vários caminhos

**Tier: modelo forte.** Muda contrato público de tool: schema, doc e erro parcial. O modo de falha barato é decidir sozinho o que acontece quando **um** dos caminhos falha.

#### Onde encaixa
Depois das otimizações de busca. Não conflita em arquivo nenhum com elas.

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

#### Onde encaixa
Depois das otimizações de busca; não conflita com elas em arquivo nenhum.

#### A evidência da lacuna
`note_read` aceita `path` (um). `note_read` p95 medido em **345 µs**. Um fluxo
de pesquisa que lê dez notas paga dez idas e voltas de protocolo por 3,5 ms de
trabalho.

#### A decisão que esta tarefa tem de acertar
**Pré-decidido, não re-litigar:**

1. O campo novo é `paths: []string`, **e `path` continua existindo**. Os dois
   preenchidos é erro de validação (`INVALID_ARGUMENT`), não precedência
   silenciosa.
2. **Falha parcial não derruba o lote.** Cada item do retorno carrega o próprio
   `error` opcional. Uma nota inexistente no meio de dez não pode custar as nove
   boas — mas **também não pode sumir**: o item aparece com o erro, na mesma
   posição. Uma lista que encolhe sem dizer é a falha silenciosa desta tarefa.
3. `max_bytes` aplica-se **por nota**, não ao lote. Um teto de lote faria a
   décima nota truncar por causa das nove anteriores, o que depende da ordem.
4. Limite de `len(paths)`: **50**. Acima disso, erro. Sem teto, uma chamada pede
   o cofre inteiro e o servidor materializa 100 MB em memória para uma resposta
   que o cliente não consegue ler.

#### Armadilhas específicas desta tarefa
- **Schema que promete e código que ignora é pior que parâmetro ausente.**
  `note_list.fields` já foi declarado e descartado. `scripts/check_tool_params.ps1`
  roda no `verify.ps1` e vai reprovar se `paths` não for lido.
- **Handler que devolve `error` Go faz o SDK montar `IsError` sem
  `StructuredContent`.** Erro de validação (os dois campos, ou lote grande
  demais) sai como resultado de erro **com** `Out` preenchido.

#### Verificações além dos passos
- Teste com dez caminhos, um deles inexistente: os nove voltam, o décimo volta
  **na posição certa** com erro.
- Teste com `path` e `paths` juntos: erro de validação.
- Teste com 51 caminhos: erro.
- `docs/TOOLS.md` atualizado, e `scripts/check_doc_refs.ps1` (Task 79) limpo.

#### Prova de mutação
Duas regras, duas provas — uma por regra, não uma por tarefa:
```
pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/tools_read.go `
  -Anchor 'if len(req.Paths) > maxPathsPorLote {' -Replacement 'if false {' `
  -Test TestNoteReadRecusaLoteAcimaDoTeto -Package ./internal/mcpsrv/

pwsh -File scripts/mutate.ps1 -Path internal/service/read.go `
  -Anchor 'out[i] = ReadNoteItem{Path: p, Err: err}' `
  -Replacement 'continue' `
  -Test TestNoteReadMantemPosicaoNoErroParcial -Package ./internal/service/
```
A segunda é a que importa: substituir o item de erro por `continue` faz a lista
encolher em silêncio, que é exatamente o defeito.

#### Contrato de relatório
Saída de `check_tool_params.ps1` e de `check_doc_refs.ps1`, mais as duas provas
de mutação acima com a saída colada.

**Files:** `internal/mcpsrv/tools_read.go`, `internal/service/read.go`,
`docs/TOOLS.md`, testes
**Commit:** `feat(note_read): read several notes in one call`

---

