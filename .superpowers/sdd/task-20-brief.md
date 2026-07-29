### Task 20: Resolução de wikilink, aliases, anexos e âncoras

#### Onde isto encaixa

A Task 19 encheu o índice com o resultado do parser. Mas um wikilink **não é um caminho**: `[[PONTO 03]]` pode se referir a qualquer nota chamada `PONTO 03.md` em qualquer pasta, e `[[P3]]` pode se referir a uma nota que declara `aliases: [P3]`. Esta tarefa traduz alvo bruto em caminho canônico.

É o que decide se o grafo é correto ou apenas plausível. A Task 21 constrói backlinks a partir do que você resolver, e a Task 25 compara tudo contra o Obsidian real — divergência aqui aparece lá como falha de paridade.

#### O que já está fechado e vincula esta tarefa

- **A ordem de resolução é a do Obsidian e não pode ser reordenada:** caminho explícito → nome de nota → nome de anexo → alias do frontmatter → desempate por proximidade → não encontrado.
- **Alias é fallback, nunca override.** Se existe `P3.md` e outra nota declara `aliases: [P3]`, `[[P3]]` aponta para o arquivo. Inverter produz um grafo que diverge do Obsidian de forma invisível.
- **Link quebrado permanece no grafo**, não é descartado. É o que permite `vault_stats` reportá-lo e o que faz um link passar a resolver sozinho quando a nota alvo for criada depois.
- **O estado do link tem três valores, não dois:** resolvido, alvo inexistente, e alvo existente com âncora inexistente. O terceiro é o que aparece depois de renomear um heading, e é invisível até alguém clicar.
- **`CanonicalPath` não garante a grafia do disco** — quem produz grafia real é `vault.Walk`. A resolução insensível a maiúsculas é o que conserta entrada de tool com casing divergente, e mais de um candidato insensível devolve erro de ambiguidade em vez de escolher.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Empate resolvido de forma não determinística.** Se dois candidatos empatam em proximidade, o desempate precisa ser estável entre execuções — caminho lexicograficamente menor, por exemplo. Sem isso o índice muda de resposta entre boots e a paridade fica impossível de reproduzir.
- **Teste que não pode falhar.** Na Task 5 a defesa contra reciclagem de PID não tinha nenhum teste que a exercitasse: todos passavam pelo caminho de erro, e a comparação de identidade nunca era chamada. Aqui o risco é o mesmo — um teste que só verifica "resolveu" não distingue resolveu-por-caminho de resolveu-por-alias. Afirme sobre **como** resolveu, não só que resolveu.
- **Colisão virando escolha arbitrária.** Aliases declarados por notas diferentes colidem. Escolher em silêncio é pior que reportar: registre os candidatos e conte a colisão para `vault_stats`.

#### Verificações além dos passos

Faça e **reporte o resultado real de cada uma**:

- Um alias que colide com o nome de um arquivo real resolve para o arquivo?
- Dois aliases iguais em notas diferentes — o que acontece, e a colisão é contável?
- `[[nota#heading-que-nao-existe]]` marca âncora quebrada mantendo o alvo resolvido?
- `[[nota#^bloco-que-nao-existe]]` idem?
- Um embed para anexo (`![[diagrama.png]]`) resolve pelo caminho de anexo, em vez de virar link quebrado?
- Resolução insensível a maiúsculas com mais de um candidato devolve erro listando os candidatos?
- Um link cujo alvo é criado depois passa a resolver quando o índice é atualizado?

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o código desta seção; não improvise uma variante. Se ele não compilar, corrija o erro mecânico e **diga exatamente o que mudou**. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar. Teste dobrado para passar é como defeito silencioso chega em produção.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** Há trabalho não commitado de outras frentes neste repositório, e um subagente já destruiu trabalho exatamente assim. Para desfazer o que você escreveu, edite de volta ou apague o arquivo específico que você criou.
- **`go mod tidy` está proibido.** Várias dependências fixadas ainda não têm importador, e o `tidy` as removeria — inclusive o pin obrigatório do SDK de MCP. Se o build reclamar de entrada faltando em `go.sum`, **pare e reporte**; não rode `go get`.
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.** Escrita em modo texto converte o arquivo inteiro para CRLF no Windows e o `gofmt` rejeita. Já custou dois commits neste projeto.
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês. Sem arquivos chamados `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave o relatório completo em `.superpowers/sdd/task-20-report.md`, com: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a tabela de verificações extras acima com o resultado real de cada uma; arquivos alterados; achados da auto-revisão; correções mecânicas que fez no código do plano; e preocupações.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, as respostas diretas pedidas acima, e preocupações. O detalhe mora no arquivo de relatório, não na resposta.

**Files:**
- Create: `internal/index/resolve.go`, `internal/index/alias.go`, `internal/index/assets.go`, `internal/index/anchors.go`
- Create: `internal/index/resolve_test.go`

**Interfaces:**
- Consumes: `Index` (Task 19), `parser.Link`, `parser.Slug`
- Produces: `(*Index).buildAliasMap()`, `(*Index).resolveAllLinks()`; `(*Index).ResolvePath(input string) (vault.CanonicalPath, error)` com `ErrAmbiguousPath`

- [ ] **Step 1: Escrever o teste**

`internal/index/resolve_test.go`:

```go
func TestResolutionOrder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Civil/PONTO 03.md", "# Ponto 3\n\n## Cap 1\n\ntexto ^blk1\n")
	writeFile(t, root, "Penal/PONTO 03.md", "# Outro\n")
	writeFile(t, root, "P3.md", "# Arquivo P3\n")
	writeFile(t, root, "Apelidada.md", "---\naliases: [P3, Terceiro]\n---\n# Apelidada\n")
	writeFile(t, root, "Anexos/diagrama.png", "\x89PNG")
	writeFile(t, root, "Origem.md", strings.Join([]string{
		"# Origem",
		"[[Civil/PONTO 03]]",       // 0: caminho explicito
		"[[P3]]",                   // 1: nome de arquivo vence alias
		"[[Terceiro]]",             // 2: alias, sem arquivo homonimo
		"![[diagrama.png]]",        // 3: anexo
		"[[Civil/PONTO 03#Cap 1]]", // 4: ancora existente
		"[[Civil/PONTO 03#Cap 9]]", // 5: ancora inexistente
		"[[Civil/PONTO 03#^blk1]]", // 6: bloco existente
		"[[Nao Existe]]",           // 7: alvo inexistente
	}, "\n\n")+"\n")

	v, _ := vault.New(root)
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	note, ok := idx.Get("Origem.md")
	if !ok {
		t.Fatal("Origem.md ausente")
	}
	if len(note.Links) != 8 {
		t.Fatalf("links = %d, quer 8: %+v", len(note.Links), note.Links)
	}

	want := []struct {
		resolved vault.CanonicalPath
		via      index.ResolveVia
		state    index.LinkState
	}{
		{"Civil/PONTO 03.md", index.ViaPath, index.LinkOK},
		{"P3.md", index.ViaName, index.LinkOK},
		{"Apelidada.md", index.ViaAlias, index.LinkOK},
		{"Anexos/diagrama.png", index.ViaAsset, index.LinkOK},
		{"Civil/PONTO 03.md", index.ViaPath, index.LinkOK},
		{"Civil/PONTO 03.md", index.ViaPath, index.LinkAnchorMissing},
		{"Civil/PONTO 03.md", index.ViaPath, index.LinkOK},
		{"", index.ViaNone, index.LinkTargetMissing},
	}

	for i, w := range want {
		got := note.Links[i]
		if got.Resolved != w.resolved {
			t.Errorf("link %d Resolved = %q, quer %q", i, got.Resolved, w.resolved)
		}
		if got.Via != w.via {
			t.Errorf("link %d Via = %v, quer %v", i, got.Via, w.via)
		}
		if got.State != w.state {
			t.Errorf("link %d State = %v, quer %v", i, got.State, w.state)
		}
	}
}

func TestNameCollisionPicksNearest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Civil/A.md", "# A civil\n")
	writeFile(t, root, "Penal/A.md", "# A penal\n")
	writeFile(t, root, "Civil/Origem.md", "# O\n\n[[A]]\n")

	v, _ := vault.New(root)
	idx := index.New()
	_ = idx.Build(context.Background(), v)

	note, _ := idx.Get("Civil/Origem.md")
	if note.Links[0].Resolved != "Civil/A.md" {
		t.Errorf("Resolved = %q, quer Civil/A.md — o mais proximo da origem",
			note.Links[0].Resolved)
	}
}

func TestResolvePathCaseInsensitiveAndAmbiguous(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Civil/PONTO 03.md", "# A\n")

	v, _ := vault.New(root)
	idx := index.New()
	_ = idx.Build(context.Background(), v)

	// Casing divergente resolve.
	got, err := idx.ResolvePath("civil/ponto 03.md")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if got != "Civil/PONTO 03.md" {
		t.Errorf("ResolvePath = %q, quer a grafia do disco", got)
	}

	// Caminho inexistente falha, nao adivinha.
	if _, err := idx.ResolvePath("Civil/Nada.md"); err == nil {
		t.Error("ResolvePath de caminho inexistente deveria falhar")
	}
}
```

Adicione `"strings"` aos imports.

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/index/ -run "TestResolution|TestName|TestResolvePath" -v`
Esperado: FAIL.

- [ ] **Step 3: Implementar**

A ordem de resolução, implementada em `resolve.go`, é a de ARCHITECTURE §3.4 e não pode ser reordenada:

1. **Caminho explícito** — o alvo contém `/`. Casa contra `notes` e `assets`, com `.md` implícito quando não há extensão.
2. **Nome de arquivo de nota** — consulta `byName`, com `.md` implícito.
3. **Nome de arquivo de anexo** — consulta `byName` restrito a `assets`, exigindo extensão explícita.
4. **Alias** — consulta `byAlias`. É fallback, nunca override: se existe `P3.md` e outra nota declara `aliases: [P3]`, o arquivo vence. Inverter produz um grafo que diverge do Obsidian de forma invisível.
5. **Desempate por proximidade** — entre candidatos, o de maior prefixo de diretório comum com a nota de origem. Empate persistente resolve pelo caminho lexicograficamente menor, para que o resultado seja determinístico.
6. **Não encontrado** — `State = LinkTargetMissing`, `Via = ViaNone`. O link permanece no grafo: é o que permite `vault_stats` reportá-lo e o que faz um link passar a resolver sozinho quando a nota alvo for criada depois.

`anchors.go` roda depois, sobre links já resolvidos: se `Anchor` começa com `^`, procura em `Blocks` do alvo; caso contrário compara `parser.Slug(anchor)` contra o `Slug` de cada heading. Sem correspondência, `State = LinkAnchorMissing` — o alvo continua resolvido, e é essa distinção que dá valor ao estado.

`alias.go` popula `byAlias` a partir de `Note.Aliases`, com chave em minúsculas. Aliases declarados por mais de uma nota registram todos os candidatos e incrementam o contador de colisão exposto em `vault_stats`.

`ResolvePath` (usado pelas tools, não pela resolução de wikilink) tenta correspondência exata em `notes`, depois insensível via `lowerPath`. Mais de um candidato insensível devolve `ErrAmbiguousPath` listando os candidatos — escolher arbitrariamente seria pior que falhar.

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/index/ -v`
Esperado: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/index
git commit -m "feat(index): obsidian-order link resolution with aliases, assets, and anchor validation"
```

---

