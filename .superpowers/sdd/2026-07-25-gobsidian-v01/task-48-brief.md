### Task 48: Tool `vault_search`

RF-20 a RF-25. É onde a busca vira produto.

#### Onde isto encaixa

`docs/TOOLS.md` já define o schema inteiro — `query`, `folder`, `tags`, `frontmatter`, `modified_after`, `modified_before`, `snippet_chars`, `limit`, `offset` — e o retorno: `path`, `title`, `score`, `snippet`, `matched_headings`, `modified`. **O contrato existe antes do código.**

#### O que já está fechado

- **Nenhum tipo do SDK de MCP cruza para fora de `internal/mcpsrv`.** `internal/service` expõe método com tipos de domínio.
- **`query` vazia com filtros redireciona para o caminho de metadados** (RF-25), muito mais barato, servido do índice em memória sem tocar o índice de texto. `docs/TOOLS.md` já promete isso.
- **Aspas duplas delimitam frase exata** (RF-24, P1). A busca por frase usa **só a forma crua** — nunca a reduzida.
- **`internal/index` já sabe filtrar por pasta, tag e frontmatter.** As consultas da Task 22 existem. Não reimplemente filtro.
- **Handler que devolve `error` Go faz o SDK montar `IsError` sem `StructuredContent`.** Devolver resultado de erro com a saída zerada manda `{"results":[]}` junto, e o cliente não distingue falha de "nada encontrado" no canal que ele lê primeiro.

#### A armadilha específica desta tarefa, que já aconteceu aqui

**`note_list` declarava `fields` no schema e o descartava.** O modelo do outro lado pede três campos, recebe tudo, e não tem como saber que o pedido não fez nada — o schema é justamente o que ele lê para decidir.

Portanto, para **cada** parâmetro do schema: um teste que o exercita e afirma que ele mudou a resposta. `limit`, `offset`, `snippet_chars`, cada filtro. Não um teste que confirma que o campo é aceito.

E para **cada campo do retorno**: um teste que afirma o valor. `score` não pode ser constante, `matched_headings` não pode vir sempre vazio. **Campo de API com valor fixo mente sempre** — `alias_collisions: 0` era literal e apareceu na resposta por semanas.

#### Verificações além dos passos

- Cada parâmetro do schema tem teste que prova que ele age? **Liste os nove com o nome do teste ao lado.**
- Cada campo do retorno tem teste que afirma o valor? Liste os seis.
- `query` vazia com filtro **não** toca o índice de texto? Prove contando, não observando o tempo.
- `limit` acima do máximo do schema é recusado ou truncado? Decida, documente em `TOOLS.md` se divergir.
- `offset` além do fim devolve lista vazia, e não erro?
- Frase entre aspas casa sequência, e não os termos soltos? É o teste que distingue RF-24 de RF-20.
- Consulta que não casa nada devolve `results: []` **e** não um erro. Cofre vazio e consulta sem resultado não podem produzir a mesma resposta que uma falha.
- `docs/TOOLS.md` descreve exatamente os nomes JSON que o código emite? Compare campo a campo, contra a saída real, não de memória.
- **RNF-04: `vault_search` p95 ≤ 100 ms.** Meça no cofre de teste disponível, com a contagem de notas ao lado. Se não mediu, escreva **"não medido"** — mas meça, porque este é o número que a Task 50 vai precisar.

**Prova de mutação obrigatória:** para o redirecionamento de `query` vazia, e para pelo menos três parâmetros do schema, mute e confirme que um teste nomeado reprova.

#### Regras de execução e contrato de relatório

Idênticos aos da Task 43. Relatório em `.superpowers/sdd/task-48-report.md`, com a **tabela dos nove parâmetros** e a **tabela dos seis campos de retorno**, cada uma com o nome do teste, mais o p95 medido ou "não medido".

**Files:** Modify `internal/service/` (método de busca), `internal/mcpsrv/` (registro da tool), `docs/TOOLS.md` se algo divergir
**Commit:** `feat(mcpsrv): vault_search tool with filters and phrase queries`

---

