### Task 51: O índice de busca precisa chegar ao watcher

Critical. Hoje a busca não é atualizada incrementalmente em produção, e o gate verde não diz nada sobre isso.

#### A evidência medida do defeito

```
cmd/gobsidian/serve.go:112-121   constrói inv, passa a service.New
cmd/gobsidian/serve.go:124       watcher.New(v, idx, debounce, log)   <- inv nao entra
internal/watcher/watcher.go:46   func New(v, idx, debounce, log)      <- sem parametro
type Watcher                      <- nenhum campo de search
internal/watcher/watcher.go:113  Apply(ctx, ..., &w.reconciledRemoved) <- sem o variadico
```

`Apply` declara `inv ...*search.Inverted`. Como `Run` não o passa, `searchInv` é **`nil` em produção sempre**, e todo o bloco de atualização de busca em `apply.go` é código morto: `apply.go:55-58` (rename), `:71-73` (remoção), `:87-89` (replace).

Consequência: nota criada, editada ou removida com o servidor rodando entra no índice de metadados e **não** entra na busca. Fica inencontrável — ou, se removida, continua encontrável — até reiniciar o servidor.

Os dois únicos call sites de teste (`apply_test.go:67`, `overflow_test.go:101`) também não passam `inv`, então o caminho tem **cobertura zero**.

**O parâmetro variádico é o que tornou isso silencioso.** Esquecer de passar compila. É a armadilha "flag não distingue omitida de definida com zero" em forma de dependência.

O brief da Task 45 pedia exatamente o teste que teria pegado: *"Ponta a ponta com o watcher: escreva uma nota nova com o servidor rodando e confirme que ela passa a ser encontrável."* Não foi feito, e as três lentes aprovaram.

#### O que implementar

**1. Acabe com o variádico.** `Apply` recebe `inv *search.Inverted` como parâmetro normal, na posição fixa. `nil` continua sendo válido — há testes que não querem busca —, mas passar `nil` passa a ser **explícito**, e um call site que esquecer não compila.

**2. O `Watcher` ganha o campo e `New` ganha o parâmetro.**

```go
func New(v *vault.Vault, idx *index.Index, inv *search.Inverted, debounce time.Duration, log *slog.Logger) (*Watcher, error)
```

`Run` repassa `w.inv` para `Apply`. Atualize **todos** os chamadores: `grep -rn "watcher.New(" --include=*.go .` antes de commitar. Uma mudança de wiring já foi propagada para `serve` e esquecida em `doctor` neste projeto.

**3. `serve.go` passa `inv` para `watcher.New`.** É a linha que falta.

**4. O erro de `Update` deixa de ser engolido.** Hoje são três `_ = searchInv.Update(...)`. `Update` lê arquivo e pode falhar; quando falha, o índice de metadados atualizou e a busca não — divergência sem sintoma. Logue em `Warn` com o caminho, e conte. Não derrube o laço: um parse ruim numa nota não pode parar a atualização das outras.

#### O teste que fecha a tarefa

```go
// TestWatcherUpdatesSearchIndex e o teste que prova que a busca acompanha o
// cofre. Sem ele, o gate fica verde com search.Update sendo codigo morto.
func TestWatcherUpdatesSearchIndex(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatal(err)
	}
	idx := index.New()
	inv := search.NewInverted()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	w, err := New(v, idx, inv, 10*time.Millisecond, log)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.Run(ctx) }()
	time.Sleep(100 * time.Millisecond) // watcher precisa estar observando

	// CRIACAO: a nota nova tem de passar a ser encontravel.
	if err := os.WriteFile(filepath.Join(tmp, "nova.md"), []byte("prescricao intercorrente"), 0644); err != nil {
		t.Fatal(err)
	}
	esperaTermo(t, inv, "prescricao", "nova.md", true)

	// REMOCAO: a nota removida tem de deixar de ser encontravel. Um teste que
	// so cobre a criacao passa com um Remove que nunca acontece.
	if err := os.Remove(filepath.Join(tmp, "nova.md")); err != nil {
		t.Fatal(err)
	}
	esperaTermo(t, inv, "prescricao", "nova.md", false)
}

// esperaTermo espera em laco com condicao de saida. time.Sleep fixo como
// assercao e o que faz um teste passar sem o mecanismo existir.
func esperaTermo(t *testing.T, inv *search.Inverted, termo, path string, quer bool) {
	t.Helper()
	limite := time.Now().Add(3 * time.Second)
	for time.Now().Before(limite) {
		presente := false
		for _, p := range inv.Postings(termo) {
			if p.Path == path {
				presente = true
			}
		}
		if presente == quer {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("apos 3s, %q em %q: presente=%v, quer %v — a busca nao acompanhou o cofre",
		termo, path, !quer, quer)
}
```

Ajuste `inv.Postings` e o campo `Path` à API real de `search.Inverted`.

#### Verificações além dos passos

- `grep -rn "watcher.New(" --include=*.go .` — liste todos os chamadores e confirme que todos foram atualizados.
- `grep -rn "\.\.\.\*search\." --include=*.go .` devolve vazio? O variádico não pode sobrar.
- Um `Update` que falha aparece no log e no contador, e o laço continua? Prove com um cofre onde uma nota é ilegível.
- Renomear uma nota atualiza a busca nos dois lados — sai do caminho antigo e entra no novo?
- `go test -race` acusa corrida entre `Update` na goroutine do watcher e `Postings` na do MCP?

**Provas de mutação obrigatórias, com saída colada:**

```bash
# 1. a fiacao. Antes desta tarefa esta mutacao nao tem o que mutar — depois
#    dela, remover o repasse tem de reprovar o teste novo.
pwsh -File scripts/mutate.ps1 -Path internal/watcher/watcher.go `
  -Anchor 'w.inv' -Replacement 'nil' `
  -Test TestWatcherUpdatesSearchIndex -Package ./internal/watcher/

# 2. a metade da remocao. Um teste que so cobre a criacao deixa passar um
#    Remove que nunca acontece.
pwsh -File scripts/mutate.ps1 -Path internal/watcher/apply.go `
  -Anchor 'searchInv.Remove(string(path))' -Replacement '_ = path' `
  -Test TestWatcherUpdatesSearchIndex -Package ./internal/watcher/
```

Se a âncora `w.inv` ocorrer mais de uma vez, o script sai `2` — amplie com as linhas vizinhas.

#### Regras de execução

- **O plano é a fonte.** Se um teste falhar por motivo que esta seção não explica, **pare e reporte**.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** **`go mod tidy` está proibido.**
- **Verde obrigatório antes do commit:** `pwsh -File scripts/verify.ps1` **e** `golangci-lint run ./internal/... ./cmd/...`.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

`.superpowers/sdd/task-51-report.md`: RED e GREEN com comando e saída; as **duas** provas de mutação com saída colada; a lista de chamadores de `watcher.New`; a saída dos dois `grep`; a tabela de verificações com resultado real; preocupações.

**Files:** Modify `internal/watcher/watcher.go`, `apply.go`, `apply_test.go`, `overflow_test.go`, novo teste; `cmd/gobsidian/serve.go`
**Commit:** `fix(watcher): the inverted index never reached the watcher`

---

