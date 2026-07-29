### Task 23: Métodos de leitura do serviço

#### Onde isto encaixa

`internal/index` está completo. Esta tarefa constrói a fachada que as tools MCP chamam: `internal/service`, os métodos de leitura, e o grafo.

É a camada onde erro de domínio vira mensagem que um modelo de linguagem lê para decidir o que fazer em seguida. `note_read` de uma seção acontece aqui, e é onde o orçamento de latência é gasto ou economizado.

#### O que já está fechado e vincula esta tarefa

- **Nenhum tipo do SDK de MCP cruza para dentro de `internal/service`.** O pacote fala tipos de domínio Go e não conhece `mcp.CallToolRequest` nem versão de protocolo. É isso que torna migração de protocolo mudança de um pacote só.
- **A mensagem de erro carrega o dado que permite corrigi-la.** `heading not found` gera uma rodada extra de chamadas; a mesma mensagem listando os headings disponíveis permite o cliente se corrigir sozinho. A taxonomia de códigos está em `docs/TOOLS.md`.
- **Ler uma seção lê só os bytes da seção.** `vault.ReadRange` já existe e faz `ReadAt`. Não leia o arquivo inteiro para recortar depois — é o que sustenta o alvo de p95.
- **Casamento de heading é por slug normalizado**, e `heading_level` desempata. Dois headings de mesmo texto **no mesmo nível** continuam ambíguos, e `heading_level` não resolve isso — o erro precisa dizer.
- **`Heading.Start` é o início da LINHA, não do `#`**, e em bloco de várias linhas os prefixos de continuação ficam dentro da faixa. Está documentado em `internal/parser/types.go`; leia antes de fatiar.
- **`ctx` onde bloqueia**, e respeitado: leitura de arquivo recebe e verifica.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Offset que trunca a nota.** A Task 12 quase shipou um contrato de `bodyOffset` ambíguo que deslocaria toda leitura de seção em exatamente 3 bytes, só em notas com BOM. O sintoma não é erro: é conteúdo cortado no meio de uma palavra. Fatie e compare com o esperado, byte a byte, numa nota com frontmatter.
- **Erro genérico onde cabia erro acionável.** Mensagem sem os dados de correção é dívida que o cliente paga em rodadas extras.
- **Teste que afirma só "não deu erro".** Um `note_read` que devolve a seção errada não dá erro. Afirme sobre o conteúdo devolvido, não sobre a ausência de falha.

#### Verificações além dos passos

Faça e **reporte o resultado real de cada uma**:

- `note_read` de uma seção numa nota **com frontmatter** devolve exatamente os bytes da seção? Compare com o esperado literal.
- A seção de um heading de nível 1 que contém subseções inclui as subseções?
- Heading inexistente produz mensagem que lista os headings do mesmo nível?
- Dois headings de mesmo texto e mesmo nível produzem erro de ambiguidade, ou o primeiro em silêncio?
- Caminho com travessia (`../fora.md`) é rejeitado com o código certo?
- `max_bytes` trunca marcando truncamento, sem erro?
- Ler bloco por `block_id` num item de lista devolve o texto do item?

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o código desta seção; não improvise uma variante. Se ele não compilar, corrija o erro mecânico e **diga exatamente o que mudou**. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar. Teste dobrado para passar é como defeito silencioso chega em produção.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** Há trabalho não commitado de outras frentes neste repositório, e um subagente já destruiu trabalho exatamente assim. Para desfazer o que você escreveu, edite de volta ou apague o arquivo específico que você criou.
- **`go mod tidy` está proibido.** Várias dependências fixadas ainda não têm importador, e o `tidy` as removeria — inclusive o pin obrigatório do SDK de MCP. Se o build reclamar de entrada faltando em `go.sum`, **pare e reporte**; não rode `go get`.
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.** Escrita em modo texto converte o arquivo inteiro para CRLF no Windows e o `gofmt` rejeita. Já custou dois commits neste projeto.
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês. Sem arquivos chamados `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave o relatório completo em `.superpowers/sdd/task-23-report.md`, com: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a tabela de verificações extras acima com o resultado real de cada uma; arquivos alterados; achados da auto-revisão; correções mecânicas que fez no código do plano; e preocupações.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, as respostas diretas pedidas acima, e preocupações. O detalhe mora no arquivo de relatório, não na resposta.

**Files:**
- Modify: `internal/service/service.go` (injeta o índice real)
- Create: `internal/service/read.go`, `internal/service/graph.go`, `internal/service/read_test.go`

**Interfaces:**
- Consumes: `Index` (Tasks 19–22), `vault.Vault` (Task 8)
- Produces: `(*Service).ReadNote(ctx, ReadRequest) (ReadResult, error)`; `ListNotes`; `NoteMetadata`; `LinkGraph`; `TagList`; `VaultStats` completo

- [ ] **Step 1: Escrever o teste**

```go
func TestReadNoteSection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "---\ntitle: A\n---\n# Titulo\n\n## Cap 1\n\ntexto um\n\n## Cap 2\n\ntexto dois\n")

	svc := newTestService(t, root)

	res, err := svc.ReadNote(context.Background(), service.ReadRequest{
		Path:    "A.md",
		Heading: "Cap 1",
	})
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}

	if !strings.Contains(res.Content, "texto um") {
		t.Errorf("conteudo nao contem a secao: %q", res.Content)
	}
	if strings.Contains(res.Content, "texto dois") {
		t.Errorf("conteudo vazou para a secao seguinte: %q", res.Content)
	}
	if res.Hash == "" {
		t.Error("Hash vazio — expected_hash depende dele")
	}
	if res.Section == nil || res.Section.Level != 2 {
		t.Errorf("Section = %+v, quer nivel 2", res.Section)
	}
}

// A mensagem de erro e lida por um modelo que precisa decidir o que fazer em
// seguida. Listar as alternativas evita uma rodada extra de chamadas.
func TestReadNoteHeadingNotFoundListsAlternatives(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# T\n\n## Cap 115\n\n## Cap 116\n\n## Cap 117\n")

	svc := newTestService(t, root)

	_, err := svc.ReadNote(context.Background(), service.ReadRequest{Path: "A.md", Heading: "Cap 118"})
	if err == nil {
		t.Fatal("heading inexistente deveria falhar")
	}
	if service.CodeOf(err) != service.CodeHeadingNotFound {
		t.Errorf("codigo = %v, quer HEADING_NOT_FOUND", service.CodeOf(err))
	}
	for _, want := range []string{"Cap 115", "Cap 116", "Cap 117"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("mensagem nao lista %q: %s", want, err.Error())
		}
	}
}

func TestReadNoteAcceptsAccentInsensitiveHeading(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "A.md", "# T\n\n## Capítulo 118\n\ntexto\n")

	svc := newTestService(t, root)

	for _, query := range []string{"Capítulo 118", "capitulo 118", "CAPITULO 118"} {
		res, err := svc.ReadNote(context.Background(), service.ReadRequest{Path: "A.md", Heading: query})
		if err != nil {
			t.Errorf("ReadNote(%q): %v", query, err)
			continue
		}
		if !strings.Contains(res.Content, "texto") {
			t.Errorf("ReadNote(%q) nao trouxe a secao", query)
		}
	}
}

func TestReadNoteRejectsTraversal(t *testing.T) {
	svc := newTestService(t, t.TempDir())

	_, err := svc.ReadNote(context.Background(), service.ReadRequest{Path: "../fora.md"})
	if service.CodeOf(err) != service.CodePathOutsideVault {
		t.Errorf("codigo = %v, quer PATH_OUTSIDE_VAULT", service.CodeOf(err))
	}
}

func TestReadNoteCloudOnlyFails(t *testing.T) {
	// Placeholder do OneDrive nao e reproduzivel em teste portavel; o caso e
	// coberto pelo doctor e pela verificacao manual de WINDOWS.md §1.1.
	t.Skip("requer arquivo com FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS")
}
```

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/service/ -v`
Esperado: FAIL.

- [ ] **Step 3: Implementar**

`read.go` implementa o fluxo de ARCHITECTURE §5.2, sem alocar o arquivo inteiro:

```
ReadNote(path, heading)
  → index.ResolvePath(path)        confinamento e casing; erro acionavel
  → index.Get(canonical)           RLock; devolve *Note
  → localizar Heading por Slug     comparacao normalizada; heading_level desambigua
  → vault.ReadRange(start, end)    apenas os bytes da secao
  → devolver bytes + metadados + hash
```

Regras que os testes fixam:

- `heading` sem correspondência produz `CodeHeadingNotFound` com a lista dos headings **do mesmo nível** na mensagem.
- Mesmo slug em mais de um lugar sem `heading_level` produz `CodeAmbiguousHeading`.
- `block_id` é mutuamente exclusivo com `heading`; passar ambos é erro de entrada.
- `max_bytes` trunca e marca `truncated: true`, sem erro.
- `include_frontmatter: false` começa a leitura em `bodyOffset`.
- `Hash` é a representação hexadecimal de `Note.Hash`.

`graph.go` implementa `LinkGraph` (BFS limitada por `depth` ≤ 3 e por `limit`), `TagList` (do mapa `tags`, com modo hierárquico opcional) e `VaultStats` completo — contagens, órfãs, links quebrados, âncoras quebradas, anexos, colisões de alias, e, com `include_runtime`, `runtime.MemStats`, número de goroutines e `generation`.

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/service/ -v`
Esperado: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service
git commit -m "feat(service): read methods with actionable heading errors and range reads"
```

---

