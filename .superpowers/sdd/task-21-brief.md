### Task 21: Backlinks e teste de propriedade

#### Onde isto encaixa

A Task 20 resolveu cada link para um caminho canônico. Esta tarefa constrói o grafo reverso — quem aponta para quem — e o mantém correto sob mutação.

Backlinks alimentam `note_metadata`, `link_graph`, a contagem de notas órfãs em `vault_stats` e o relatório de impacto de `note_delete`. Um backlink fantasma faz o produto afirmar uma referência que não existe.

#### O que já está fechado e vincula esta tarefa

- **Backlinks são derivados, nunca fonte.** Reconstruídos a partir dos links de saída. Perder o índice custa uma reindexação, nunca dados.
- **Cada entrada carrega o caminho da origem.** É isso que permite remover seletivamente as contribuições de uma nota antes de inserir as novas.
- **O contexto textual do backlink é lido do disco na consulta**, não guardado no índice — guardá-lo violaria a decisão de armazenar offsets e não conteúdo.
- **`ctx` só onde bloqueia.** Contabilidade em memória não recebe `ctx`.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Backlink órfão é o defeito clássico aqui, e é de remoção, não de inserção.** Reconstruir é fácil; **remover as contribuições antigas** de uma nota antes de inserir as novas é onde se erra. Sem isso, editar uma nota para tirar um link deixa o backlink pendurado, e o grafo passa a afirmar uma referência que não existe.
- **Invariante de mão dupla.** Testar só "todo link tem backlink" deixa passar exatamente o defeito acima. Teste também "todo backlink corresponde a um link real" — é a direção 2 que o pega.
- **Alvo recém-criado precisa reconciliar.** Ao substituir uma nota, links que apontavam para ela e estavam quebrados podem passar a resolver. Reprocessá-los faz parte da atualização.
- **Teste inerte.** Na Task 8 as fixtures de exclusão usavam extensões que o filtro descartaria de qualquer jeito, então apagar a regra não mudava contagem nenhuma — cobertura reportada que não existia. Aqui o análogo é um teste de propriedade que roda só sobre um cofre imutável: a sequência de mutações **é** o teste.

#### Verificações além dos passos

Faça e **reporte o resultado real de cada uma**:

- Depois de editar uma nota para remover um link, o backlink correspondente desaparece?
- Depois de remover uma nota, os links que apontavam para ela viram alvo inexistente?
- Depois de recriar essa nota, eles voltam a resolver?
- Duas notas apontando para a mesma terceira produzem dois backlinks distinguíveis pela origem?
- Uma nota que aponta para si mesma — o que acontece?
- **Prove que o teste de propriedade pega o defeito:** remova a linha que limpa as contribuições antigas, rode, confirme que a invariante falha na direção 2, restaure, confirme verde. Reporte as duas saídas.

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o código desta seção; não improvise uma variante. Se ele não compilar, corrija o erro mecânico e **diga exatamente o que mudou**. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar. Teste dobrado para passar é como defeito silencioso chega em produção.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** Há trabalho não commitado de outras frentes neste repositório, e um subagente já destruiu trabalho exatamente assim. Para desfazer o que você escreveu, edite de volta ou apague o arquivo específico que você criou.
- **`go mod tidy` está proibido.** Várias dependências fixadas ainda não têm importador, e o `tidy` as removeria — inclusive o pin obrigatório do SDK de MCP. Se o build reclamar de entrada faltando em `go.sum`, **pare e reporte**; não rode `go get`.
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.** Escrita em modo texto converte o arquivo inteiro para CRLF no Windows e o `gofmt` rejeita. Já custou dois commits neste projeto.
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês. Sem arquivos chamados `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave o relatório completo em `.superpowers/sdd/task-21-report.md`, com: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a tabela de verificações extras acima com o resultado real de cada uma; arquivos alterados; achados da auto-revisão; correções mecânicas que fez no código do plano; e preocupações.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, as respostas diretas pedidas acima, e preocupações. O detalhe mora no arquivo de relatório, não na resposta.

**Files:**
- Create: `internal/index/backlinks.go`, `internal/index/update.go`
- Create: `internal/index/backlinks_test.go`

**Interfaces:**
- Consumes: `Index`, `ResolvedLink` (Tasks 19–20)
- Produces: `(*Index).buildBacklinks()`; `(*Index).Backlinks(vault.CanonicalPath) []Backlink`; `(*Index).Replace(ctx, *vault.Vault, vault.CanonicalPath) error`; `(*Index).Remove(vault.CanonicalPath)`

- [ ] **Step 1: Escrever o teste de propriedade**

```go
// Invariante central do indice: para toda nota N e todo link L em N que
// resolve para M, existe um backlink de N em M. E o inverso: todo backlink
// registrado corresponde a um link real.
func TestBacklinkInvariantUnderMutation(t *testing.T) {
	root := t.TempDir()
	for i := range 20 {
		writeFile(t, root, fmt.Sprintf("n%02d.md", i),
			fmt.Sprintf("# N%d\n\n[[n%02d]]\n[[n%02d]]\n", i, (i+1)%20, (i+7)%20))
	}

	v, _ := vault.New(root)
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	assertInvariant(t, idx)

	// Sequencia de mutacoes: modificar, remover, recriar.
	ctx := context.Background()

	writeFile(t, root, "n05.md", "# N5 sem links\n")
	if err := idx.Replace(ctx, v, "n05.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	assertInvariant(t, idx)

	if err := os.Remove(filepath.Join(root, "n07.md")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	idx.Remove("n07.md")
	assertInvariant(t, idx)

	writeFile(t, root, "n07.md", "# N7 de volta\n\n[[n00]]\n")
	if err := idx.Replace(ctx, v, "n07.md"); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	assertInvariant(t, idx)
}

func assertInvariant(t *testing.T, idx *index.Index) {
	t.Helper()

	// Direcao 1: todo link resolvido tem backlink correspondente.
	for _, path := range idx.Paths() {
		note, _ := idx.Get(path)
		for _, link := range note.Links {
			if link.Resolved == "" {
				continue
			}
			found := false
			for _, bl := range idx.Backlinks(link.Resolved) {
				if bl.From == path {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("link %s -> %s sem backlink correspondente", path, link.Resolved)
			}
		}
	}

	// Direcao 2: todo backlink corresponde a um link real.
	for _, target := range idx.Paths() {
		for _, bl := range idx.Backlinks(target) {
			origin, ok := idx.Get(bl.From)
			if !ok {
				t.Errorf("backlink de %s para %s, mas a origem nao esta no indice", bl.From, target)
				continue
			}
			found := false
			for _, link := range origin.Links {
				if link.Resolved == target {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("backlink fantasma: %s -> %s", bl.From, target)
			}
		}
	}
}
```

Adicione `(*Index).Paths() []vault.CanonicalPath` ao índice — ordenado, para que os testes sejam determinísticos.

**Direção 2 é a que pega o bug real.** Reconstruir backlinks é fácil; *remover as contribuições antigas* de uma nota antes de inserir as novas é onde se erra. Sem isso, editar uma nota para remover um link deixa o backlink órfão, e o grafo passa a afirmar uma referência que não existe.

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/index/ -run TestBacklink -v`
Esperado: FAIL.

- [ ] **Step 3: Implementar**

`backlinks.go` mantém `backlinks map[CanonicalPath][]Backlink`. Cada entrada carrega `From`, que é o que permite a remoção seletiva em `Replace`. O contexto textual de cada backlink são até 120 caracteres ao redor do link, lidos do disco no momento da consulta — guardá-los no índice violaria AD-05.

`update.go` implementa `Replace`, que sob `Lock`: remove as contribuições da nota antiga de `backlinks`, `tags`, `byName`, `byAlias` e `lowerPath`; reparseia o arquivo; insere as novas; resolve os links da nota; **e reprocessa os links que apontavam para ela**, porque um alvo recém-criado pode fazer links quebrados passarem a resolver. Incrementa `generation`.

`Remove` faz o simétrico, e marca como `LinkTargetMissing` os links que apontavam para a nota removida.

- [ ] **Step 4: Rodar para confirmar que passa**

Run: `go test -race ./internal/index/ -v`
Esperado: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/index
git commit -m "feat(index): backlink graph with correct removal of stale contributions"
```

---

