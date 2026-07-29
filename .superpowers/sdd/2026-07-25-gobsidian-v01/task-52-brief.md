### Task 52: A medição honesta da Q3, e o p95 de verdade

A Q3 do PRD §11 foi **reaberta** em 2026-07-29. Esta tarefa a fecha com número medido.

#### Por que foi reaberta

Duas coisas, ambas verificáveis no repositório:

**O corpus tinha uma nota, rotulada como cem.** `internal/search/persist_test.go:156-158`:

```go
for i := 0; i < 100; i++ {
    inv.Add(filepath.Join("folder", "note.md"), search.Analyze("termo exemplo..."))
}
```

Cem inserções **do mesmo caminho**. O cache continha uma nota, e o log do próprio teste dizia `Notes: 1` ao lado do rótulo `"100 notas"`. O número foi para o PRD como decisão fechada e para o `OPERACAO.md`.

**A comparação que decide a pergunta nunca foi feita.** A Q3 pergunta se vale persistir o índice de busca **ou** reconstruí-lo. O relatório mediu `Save` e `Load` do cache — o custo de persistir. O custo de **reconstruir a partir do índice de metadados já carregado**, que é o outro lado da comparação, não foi medido. "Persistir ambos" foi escolhido, não medido.

**E o p95 do RNF-04 não era p95.** Uma chamada em teste unitário com índice em memória. Um ponto não é percentil.

#### O que implementar

**1. Um corpus de caminhos distintos.** O gerador vai no teste, literal:

```go
// geraCorpus cria N notas com caminhos DISTINTOS e conteudo distinto. O
// defeito que esta tarefa corrige foi um laco que inseria N vezes o mesmo
// caminho: o cache tinha uma nota e o rotulo dizia N.
func geraCorpus(t *testing.T, n int) (*vault.Vault, *index.Index, *search.Inverted) {
	t.Helper()
	root := t.TempDir()
	for i := 0; i < n; i++ {
		dir := filepath.Join(root, fmt.Sprintf("pasta%02d", i%10))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		corpo := fmt.Sprintf("---\ntags: [t%d]\n---\n\n# Nota %d\n\nprescricao intercorrente termo%d civil\n", i%7, i, i)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("nota%04d.md", i)), []byte(corpo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v, err := vault.New(root)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	inv := search.NewInverted()
	for _, p := range idx.NotePaths() {
		data, err := v.ReadAll(context.Background(), p)
		if err != nil {
			t.Fatalf("corpus ilegivel em %s: %v", p, err)
		}
		body, _ := vault.StripBOM(data)
		inv.Add(string(p), search.Analyze(string(body)))
	}
	// A assercao que o teste anterior nao tinha: o corpus e do tamanho que o
	// rotulo diz. Sem ela, todo numero abaixo mede outra coisa.
	if got := idx.NoteCount(); got != n {
		t.Fatalf("corpus tem %d notas, quer %d", got, n)
	}
	return v, idx, inv
}
```

**2. As duas medições, lado a lado.** No mesmo corpus, no mesmo teste:

- **(a)** `LoadInvertedCache` do disco.
- **(b)** reconstruir o índice invertido a partir do `index.Index` **já carregado** — o laço que `serve.go` já faz no caminho de cache-miss.

Reporte os dois números com a contagem de notas. **A decisão sai do número, não da preferência:** se (b) mais o carregamento dos metadados couber em 300 ms, persista **só** os metadados, porque é o formato mais barato de versionar. Se não couber, persista os dois e registre o custo de versionamento como dívida.

**3. O p95 do RNF-04.** Uma medição, não uma chamada: pelo menos 200 consultas, com termos que variam, sobre o corpus gerado. Ordene as durações e reporte o percentil 95. Uma chamada única não pode ser rotulada p95.

**4. Feche a Q3 no PRD §11** com a data, os dois números e a decisão. **Atualize `docs/OPERACAO.md`**, onde RNF-02 e RNF-04 estão hoje como **"não medido"** — a revisão os retirou de propósito, porque número falso esperando tarefa é número falso no repositório.

#### A regra que governa esta tarefa

**Não escreva número que você não mediu, e confira que o que você mediu é o que o rótulo diz.** O defeito anterior não foi inventar um número: foi medir honestamente a coisa errada e rotulá-la certo. Antes de escrever qualquer linha da tabela, pergunte o que exatamente estava no corpus — e afirme isso no teste, como `geraCorpus` faz.

Se não conseguir medir, a resposta é `BLOCKED` com o motivo. **Estimativa apresentada como resultado não é resposta.**

#### Verificações além dos passos

- Qual o tamanho do corpus? Afirmado no teste, não só no rótulo?
- Os dois números da Q3, com a contagem de notas ao lado.
- O p95 do RNF-04, com o número de consultas e a dispersão (mínimo, mediana, p95).
- `TestQ3PerformanceMeasurement`, que existe hoje, foi corrigido ou substituído? Se o laço do mesmo caminho ficou em algum lugar, ele volta a mentir.
- Cache gravado sobre corpus de N notas, relido, tem N notas? `header.NoteCount == n`.

**Prova de mutação obrigatória:** troque `n` por `1` na chamada do gerador e confirme que a asserção de tamanho do corpus reprova. É a prova de que o rótulo e o conteúdo estão amarrados.

#### Regras de execução e contrato de relatório

Idênticos aos da Task 51. Relatório em `.superpowers/sdd/task-52-report.md`, com **as duas medições da Q3**, o **p95 com a dispersão**, o diff do PRD §11 e do `OPERACAO.md`, e a prova de mutação.

**Files:** Modify `internal/search/persist_test.go` (ou novo arquivo de medição), `docs/PRD.md` §11, `docs/OPERACAO.md`
**Commit:** `test(search): measure Q3 on a corpus of distinct paths and a real p95`

---

