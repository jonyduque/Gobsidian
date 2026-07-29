### Task 22: Consultas por metadados

#### Onde isto encaixa

O índice (19), a resolução (20) e os backlinks (21) estão prontos. Esta tarefa expõe consultas estruturais sobre eles: por tag, por pasta, por glob de caminho, por campo de frontmatter.

É a metade barata da busca. Consulta de metadados é servida direto da memória, sob `RLock`, sem tocar disco nem índice de texto — latência de microssegundos. `note_list` inteiro se apoia nisso, e usar `vault_search` para o mesmo trabalho é ordens de grandeza mais caro.

#### O que já está fechado e vincula esta tarefa

- **A semântica de casamento de `frontmatter` está especificada em `docs/TOOLS.md`, seção "Convenções gerais"**, numa tabela: escalar contra escalar, escalar contra lista, lista contra lista, `null` significando presença, chave com ponto navegando em objeto aninhado, e campo ausente. Implemente aquela tabela, não uma interpretação dela.
- **`total` é a contagem antes de `limit` e `offset`**, não depois. Devolver a contagem já cortada torna a paginação inútil — o cliente não tem como saber que há mais.
- **A ordem de filtragem importa para o custo:** tag primeiro (consulta de mapa invertido), depois pasta (prefixo), depois glob, e frontmatter por último, varrendo só os sobreviventes.
- **`ctx` só onde bloqueia.** Nada aqui toca disco; nada aqui recebe `ctx`.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Teste que passaria contra um stub.** Foi o padrão mais recorrente das revisões deste projeto: fixtures que o filtro descartaria de qualquer jeito, asserções só de contagem, casos cujo valor esperado é zero contra uma implementação que devolve zero. Para cada regra de casamento, pergunte se apagá-la faria algum teste falhar. Se não, o teste não testa a regra.
- **Comparação insensível a acentos e caixa.** É requisito, não conveniência: um cofre em português tem `Ação` e `acao` no mesmo campo. Mas insensibilidade tem que valer só onde a tabela diz.
- **Zero significando duas coisas.** `config.Flags` precisou de companheiros `ReadOnlySet` porque um zero não distingue "omitido" de "definido como zero". Se alguma opção de consulta tiver o mesmo formato, resolva igual.

#### Verificações além dos passos

Faça e **reporte o resultado real de cada uma**:

- Cada linha da tabela de `frontmatter` de `docs/TOOLS.md` tem um caso? Liste linha por linha.
- `total` reflete a contagem antes do corte? Prove com um conjunto maior que `limit`.
- Filtro por tag pega tag hierárquica pai quando se busca o filho, e vice-versa? Qual é o comportamento pretendido?
- Glob com `*` e com `?` casa o que se espera, e não atravessa separador de pasta quando não deveria?
- Ordenação por cada valor de `sort`, nas duas direções, é estável para valores iguais?
- Uma consulta sem nenhum filtro devolve tudo, ou nada?

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o código desta seção; não improvise uma variante. Se ele não compilar, corrija o erro mecânico e **diga exatamente o que mudou**. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar. Teste dobrado para passar é como defeito silencioso chega em produção.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** Há trabalho não commitado de outras frentes neste repositório, e um subagente já destruiu trabalho exatamente assim. Para desfazer o que você escreveu, edite de volta ou apague o arquivo específico que você criou.
- **`go mod tidy` está proibido.** Várias dependências fixadas ainda não têm importador, e o `tidy` as removeria — inclusive o pin obrigatório do SDK de MCP. Se o build reclamar de entrada faltando em `go.sum`, **pare e reporte**; não rode `go get`.
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.** Escrita em modo texto converte o arquivo inteiro para CRLF no Windows e o `gofmt` rejeita. Já custou dois commits neste projeto.
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês. Sem arquivos chamados `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave o relatório completo em `.superpowers/sdd/task-22-report.md`, com: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a tabela de verificações extras acima com o resultado real de cada uma; arquivos alterados; achados da auto-revisão; correções mecânicas que fez no código do plano; e preocupações.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, as respostas diretas pedidas acima, e preocupações. O detalhe mora no arquivo de relatório, não na resposta.

**Files:**
- Create: `internal/index/query.go`, `internal/index/query_test.go`

**Interfaces:**
- Consumes: `Index` (Tasks 19–21)
- Produces: `index.Query{Folder, Glob string, Tags []string, TagMode string, Frontmatter map[string]any, Recursive bool, Sort, Order string, Limit, Offset int}`; `(*Index).List(Query) ([]*Note, int)`; `(*Index).Tags(prefix string, minCount int) []TagCount`

- [ ] **Step 1: Escrever o teste**

Cobre, um subteste por linha, a tabela de semântica de `frontmatter` de `docs/TOOLS.md` (Convenções gerais): escalar contra escalar com comparação insensível a acentos e caixa; escalar contra lista; lista contra lista exigindo todos; `null` casando com presença; chave com ponto navegando em objeto aninhado; campo ausente nunca casando. Mais: `folder` com e sem `recursive`, `glob` com `*` e `?`, `tags` com `tag_mode` `all` e `any`, ordenação por cada valor de `sort` nas duas ordens, e `limit`/`offset` devolvendo `total` correto.

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/index/ -run TestQuery -v`
Esperado: FAIL.

- [ ] **Step 3: Implementar**

Consultas de metadados são servidas direto do índice em memória, sob `RLock`, sem tocar disco. Latência de microssegundos. A filtragem aplica, nesta ordem — da mais seletiva e barata para a mais cara: `tags` (lookup no mapa invertido), `folder` (prefixo de caminho), `glob` (`path.Match`), `frontmatter` (varredura dos sobreviventes).

`total` é a contagem **antes** de `limit`/`offset`. É o que permite ao cliente saber que há mais resultados; devolver a contagem depois do corte torna a paginação inútil.

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/index/ -v`
Esperado: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/index
git commit -m "feat(index): metadata queries with documented frontmatter matching rules"
```

---

