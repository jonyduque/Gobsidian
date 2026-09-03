### Task 173: d — `service` recebe `*index.Index`; a interface `service.Index` some

**Files:**
- Modify: `internal/service/service.go:11-24` (apagar `type Index interface {…}`), `:84` (`New(v *vault.Vault, idx *index.Index, inv *search.Inverted, w WatchStats, opts Options)`), campo `Service.index *index.Index`
- Modify: `internal/service/search.go:202-204` (apagar a asserção `s.index.(*index.Index)`), `:208` e `:298` (passar `s.index` direto a `search.CalculateBM25` e `search.GenerateSnippet`)
- Modify: `docs/ESTADO.md:136` (única menção à interface)
- Test: nenhum novo. Os existentes de `service`, `mcpsrv`, `daemon` e `cmd/gobsidian` já passam `*index.Index` a `New`.

**Interfaces:**
- Consumes: Task 158 (índice `nil` → `VAULT_UNAVAILABLE`): o teste `if s.index == nil` continua verdadeiro com ponteiro tipado — não há o alçapão de interface-com-ponteiro-nil, porque não há mais interface.
- Produces: `service.New(v *vault.Vault, idx *index.Index, inv *search.Inverted, w WatchStats, opts Options) *Service`. Tasks 174–177 chamam esta assinatura.

- [ ] **Step 1: Confirmar que a interface tem uma implementação e nenhum fake**

Run: `gopls references internal/service/service.go:#<offset de Index>` (ou `grep -rn "service\.Index\b\|Index interface" --include=*.go .`).
Expected: definição em `service.go:11`, parâmetro em `New`, campo `index`, asserção em `search.go:202`. Nenhum `type fakeIndex` em `_test.go`. Se aparecer um fake, **parar**: a Task muda de "apagar interface" para "manter interface e apagar asserção", e isso é ruling do orquestrador — reportar `BLOCKED` com o arquivo.

- [ ] **Step 2: Apagar a interface e tipar o campo**

`internal/service/service.go`:

```go
// Service e a fachada das tools sobre o dominio. Recebe o indice concreto:
// a interface que existia aqui tinha uma implementacao e nenhum fake, e
// search.CalculateBM25 e search.GenerateSnippet exigem *index.Index, o que
// obrigava uma assercao de tipo em cada busca.
type Service struct {
	vault    *vault.Vault
	index    *index.Index
	inverted *search.Inverted
	watch    WatchStats
	opts     Options
	// ... demais campos inalterados
}

func New(v *vault.Vault, idx *index.Index, inv *search.Inverted, w WatchStats, opts Options) *Service {
```

Apagar o bloco `type Index interface { … }` inteiro (`:11-24`). O import de `internal/index` já existe no pacote.

- [ ] **Step 3: Remover a asserção em `search.go`**

Substituir `:202-204`:

```go
	var idxImpl *index.Index
	if realIdx, ok := s.index.(*index.Index); ok {
		idxImpl = realIdx
	}
```

por nada, e trocar `idxImpl` por `s.index` nas chamadas `search.CalculateBM25(queryTokens, s.inverted, s.index)` (`:208`) e `search.GenerateSnippet(ctx, s.vault, s.inverted, s.index, …)` (`:298`). Se `idxImpl` aparecer em mais lugares, `gopls references` lista todos — trocar cada um.

Run: `go build ./... && go vet ./...` — Expected: limpo.
Run: `go test -race ./internal/service/ ./internal/mcpsrv/ ./internal/daemon/ ./cmd/...` — Expected: PASS.

- [ ] **Step 4: Prova de que o `nil` da Task 158 ainda é tratado**

Run: `go test -race ./internal/service/ -run 'Nil|Unavailable|Indisponivel' -v` — Expected: os testes da Task 158 passam com a assinatura nova (o nome exato está no relatório da 158; se nenhum casar com o padrão, `grep -n VAULT_UNAVAILABLE internal/service/*_test.go` mostra qual é).

- [ ] **Step 5: `benchstat` — a asserção sumiu; nada mais mudou**

`antes_service.test.exe` (Baseline) × `go test -c -o depois_service.test.exe ./internal/service/`, 7 intercalados, `-bench 'SearchDoisTermos|SearchFraseExata|SearchLimit200Cache|SearchTermoAmploCache' -benchmem`; `IndexBuild` em `antes_index` × `depois_index` (não muda — é a prova de que não muda). Expected: `~` em todos. Colar.

- [ ] **Step 6: Documentação**

`docs/ESTADO.md:136`: trocar a frase que descreve `service.Index` por "`service.New` recebe `*index.Index`; a interface foi removida em 2026-09 (Task 173) — tinha uma implementação e nenhum fake."

- [ ] **Step 7: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/service/service.go internal/service/search.go docs/ESTADO.md
git commit -m "refactor(service): take *index.Index directly; the interface had one implementation and no fake"
```

#### Verificações

- `grep -rn "service\.Index\b\|idxImpl" --include=*.go .` vazio, colado.
- `benchstat` do Step 5 colado — `~` em todos.
- `verify.ps1` verde (última linha colada).

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Estrutural puro: nenhum comportamento muda, nenhum teste é editado. Se um teste precisar de edição para compilar, é porque existe um fake — voltar ao Step 1.
- `service` **não** ganha import: `index` já era importado.

#### Comando de mutação

Esta tarefa não tem prova de mutação: é estrutural, e a prova é `go build`, `go vet`, os testes da Task 158 e o `benchstat` com `~`.

#### Contrato de relatório

`task-173-report.md`: status, SHA, saída do Step 1, `grep` vazio, saída do Step 4, `benchstat`, última linha do `verify.ps1`.

---

