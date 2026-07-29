### Task 24: Tools de leitura e resources

#### Onde isto encaixa

`internal/service` expõe os métodos; esta tarefa os publica como tools MCP e liga o servidor ao índice. É o ponto em que o produto passa a ser usável de verdade dentro do Claude Desktop — **é o corte da v0.1**.

#### O que já está fechado e vincula esta tarefa

- **Nenhum tipo do SDK sai de `internal/mcpsrv`.** Os handlers traduzem e delegam; lógica de domínio nenhuma mora aqui.
- **stdout pertence ao JSON-RPC.** Todo log vai para stderr. Um único byte estranho em stdout corrompe a sessão, e o sintoma é o servidor sumir do host sem erro nenhum.
- **`Serve` usa `mcp.IOTransport`, não `StdioTransport`**, e recebe um leitor espelhado. `StdioTransport` lê `os.Stdin` direto e colide com o monitor de EOF do ciclo de vida; dois leitores no mesmo descritor repartem os bytes e corrompem a sessão.
- **`--read-only` remove as tools de escrita da lista anunciada**, não as rejeita na chamada. Host que vê a tool tenta usá-la, e a recusa desperdiça uma rodada.
- **Toda chamada a `config.Load` preenche os companheiros das flags** (`ReadOnlySet`, `DebounceMSSet`) via `cmd.Flags().Changed(nome)`. Esquecer em um subcomando faz a flag virar no-op silencioso.
- **Panic em handler vira resultado de erro**, nunca derruba o servidor. Já há middleware; use-o em toda tool nova.
- **O servidor só sobe depois de indexado.** Anunciar disponibilidade antes faz a primeira chamada responder com dado incompleto — pior que um boot mais lento.
- **O esquema de resource é `gobsidian://`**, não `obsidian://` — o segundo pertence ao aplicativo e é registrado no sistema operacional.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Descrição de schema é o que o modelo lê para decidir se chama a tool.** Copie as descrições de `docs/TOOLS.md` literalmente em vez de reescrevê-las; elas foram redigidas para serem lidas por um modelo.
- **Resultado de erro com saída zerada junto.** Devolver `IsError` com a struct de saída zerada manda `{"notes":0,...}` no mesmo pacote, e o cliente não distingue falha de cofre vazio no canal que lê primeiro.
- **Teste que só chama e confere que não deu erro.** Afirme sobre a estrutura devolvida.
- **Handshake manual com stdin fechado no mesmo instante** devolve zero bytes — o ciclo de vida vê EOF e corre contra a escrita da resposta. Não é defeito: segure o stdin aberto, como `docs/WINDOWS.md` §8.3 mostra.

#### Verificações além dos passos

Faça e **reporte o resultado real de cada uma**:

- Cada uma das cinco tools aparece em `ListTools` e responde a uma chamada válida com a estrutura de `docs/TOOLS.md`?
- Uma chamada inválida devolve `IsError` com o código correto **e sem** estrutura de saída zerada junto?
- O servidor continua respondendo depois de um erro de tool e depois de um panic?
- Grep no módulo inteiro por `fmt.Print`, `os.Stdout` e `println`: o que é alcançável a partir de `serve`? (`doctor` e `version` imprimem em stdout de propósito — são comandos CLI.)
- Handshake manual, com o stdin segurado aberto, produz **uma única linha JSON** em stdout?
- `ListResources` devolve URIs `gobsidian://`, respeita o limite e ordena por data de modificação decrescente?

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o código desta seção; não improvise uma variante. Se ele não compilar, corrija o erro mecânico e **diga exatamente o que mudou**. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar. Teste dobrado para passar é como defeito silencioso chega em produção.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** Há trabalho não commitado de outras frentes neste repositório, e um subagente já destruiu trabalho exatamente assim. Para desfazer o que você escreveu, edite de volta ou apague o arquivo específico que você criou.
- **`go mod tidy` está proibido.** Várias dependências fixadas ainda não têm importador, e o `tidy` as removeria — inclusive o pin obrigatório do SDK de MCP. Se o build reclamar de entrada faltando em `go.sum`, **pare e reporte**; não rode `go get`.
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.** Escrita em modo texto converte o arquivo inteiro para CRLF no Windows e o `gofmt` rejeita. Já custou dois commits neste projeto.
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês. Sem arquivos chamados `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave o relatório completo em `.superpowers/sdd/task-24-report.md`, com: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a tabela de verificações extras acima com o resultado real de cada uma; arquivos alterados; achados da auto-revisão; correções mecânicas que fez no código do plano; e preocupações.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, as respostas diretas pedidas acima, e preocupações. O detalhe mora no arquivo de relatório, não na resposta.

**Files:**
- Modify: `internal/mcpsrv/server.go` (registra as cinco tools restantes)
- Create: `internal/mcpsrv/tools_read.go`, `internal/mcpsrv/resources.go`
- Create: `internal/mcpsrv/tools_read_test.go`
- Modify: `cmd/gobsidian/serve.go` (constrói e injeta o índice antes de servir)

**Interfaces:**
- Consumes: `Service` (Task 23)
- Produces: tools `note_read`, `note_list`, `note_metadata`, `link_graph`, `tag_list` registradas; resources sob `gobsidian://`

- [ ] **Step 1: Escrever o teste ponta a ponta**

Um subteste por tool, cada um atravessando o transporte em memória como na Task 9, verificando que: a tool aparece em `ListTools`; uma chamada válida devolve `IsError: false` com a estrutura documentada em `docs/TOOLS.md`; uma chamada inválida devolve `IsError: true` com o código correto no início da mensagem; e o servidor continua respondendo depois do erro.

Mais um teste para os resources: `ListResources` devolve URIs `gobsidian://`, respeita o limite padrão de 200 e ordena por data de modificação decrescente; `ReadResource` de uma URI devolve `text/markdown`.

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/mcpsrv/ -v`
Esperado: FAIL.

- [ ] **Step 3: Implementar**

Cada tool em `tools_read.go` segue o mesmo formato: uma struct de entrada com tags `json` e `jsonschema` espelhando exatamente o schema de `docs/TOOLS.md`, um handler embrulhado em `guard`, e conversão de erro por `toolError`. Nenhum handler contém lógica de domínio — todos delegam ao `Service` em uma linha e traduzem o resultado.

O `jsonschema` de cada campo importa mais do que parece: é a descrição que o modelo lê para decidir se e como chamar a tool. Copie as descrições de `docs/TOOLS.md` literalmente em vez de reescrevê-las.

Em `serve.go`, a ordem do boot é a de ARCHITECTURE §5.1, e o passo do servidor é o **último**:

```
1. Analisar flags, resolver e validar o caminho do cofre
2. Iniciar lifecycle
3. Construir o indice (varredura completa; cache fica para M3)
4. Construir o Service com o indice
5. Iniciar o servidor MCP em stdio
```

Um servidor que aceita `initialize` antes de estar indexado responde a primeira chamada com erro ou com dados incompletos — pior que um boot 200 ms mais lento.

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/mcpsrv/ -v`
Esperado: PASS.

- [ ] **Step 5: Verificar no Claude Desktop**

```powershell
.\scripts\build.ps1
```

Registre com o script de `docs/WINDOWS.md` §8.1, ajustando `$BinaryPath` para `bin\gobsidian.exe`. Reinicie o Claude Desktop e confirme que as seis tools aparecem e que `note_list` devolve notas reais do cofre.

Se o servidor não aparecer, rode o handshake manual do Step 8 da Task 9 antes de qualquer outra hipótese: em quase todos os casos a causa é algo escrito em stdout.

- [ ] **Step 6: Commit**

```bash
git add internal/mcpsrv cmd/gobsidian
git commit -m "feat(mcpsrv): read tools and gobsidian:// resources"
```

---

