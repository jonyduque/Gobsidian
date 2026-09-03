### Task 171: c1 PR1 — `vault.ReplaceFile`, `vault.WriteAtomic`, `vault.TempFilePrefix`, `vault.SweepStaleTempFiles`; `writer` encaminha

**Files:**
- Create: `internal/vault/atomic.go` (movido de `internal/writer/atomic.go`, menos `CleanStaleTempFiles` já apagada na Task 166)
- Create: `internal/vault/syncdir_unix.go`, `internal/vault/syncdir_windows.go` (movidos)
- Create: `internal/vault/atomic_test.go`, `internal/vault/durabilidade_test.go`, `internal/vault/sweep_profundo_windows_test.go` (movidos de `writer`)
- Modify: `internal/writer/atomic.go` (vira só encaminhadores com alias)
- Modify: `internal/vault/walk.go:74` (literal `".gobsidian-tmp-"` → `TempFilePrefix`)
- Modify: `internal/vault/longpath_windows.go:26` (comentário cita `writer.SweepStaleTempFiles` → `SweepStaleTempFiles`)
- Modify: `docs/ESTRUTURA.md`, `docs/ARCHITECTURE.md` (onde descrevem `writer`/`vault`)

**Interfaces:**
- Produces (pacote `vault`):
  - `const TempFilePrefix = ".gobsidian-tmp-"`
  - `func ReplaceFile(ctx context.Context, targetPath string, escrever func(*os.File) error) error` — temp no mesmo diretório com `TempFilePrefix`, chmod do alvo, chama `escrever(tmp)`, `Sync`, `Close`, rename com retry e `sincronizarDiretorio`. É o corpo atual de `WriteAtomic` com o `tmpFile.Write(data)` substituído por `escrever(tmpFile)`.
  - `func WriteAtomic(ctx context.Context, targetPath string, data []byte) error` — `ReplaceFile(ctx, targetPath, func(f *os.File) error { _, err := f.Write(data); return err })`.
  - `type SweepResult` (movido), `func SweepStaleTempFiles(ctx context.Context, root string) (SweepResult, error)`.
- `writer` durante PR1: `const TempFilePrefix = vault.TempFilePrefix`; `type SweepResult = vault.SweepResult`; `func WriteAtomic(ctx, p, data) error { return vault.WriteAtomic(ctx, p, data) }`; `func SweepStaleTempFiles(ctx, root) (SweepResult, error) { return vault.SweepStaleTempFiles(ctx, root) }` — cada um com `// Deprecated: use vault.X. Removido na Task 172.`

**Por que `ReplaceFile` com callback:** os dois caches (`index/persist.go:124-146`, `search/persist.go:87-116`) codificam **em streaming** para o temporário (`escreveIndexCache(w io.Writer, …)`), não têm `[]byte`. Sem o callback, a Task 172 teria de materializar 25 MiB (`SaveInvertedCacheReal`: 24,95 MiB B/op) para chamar `WriteAtomic`. Com ele, `WriteAtomic` é o caso particular.

**Ruling do orquestrador (registrado no ledger):** os caches passam a ter `Sync()` como as notas — uma conta, um comportamento. Custo medido na Baseline: `SaveIndexCacheReal` 21,76 → 41,87 ms com fsync; `SaveInvertedCacheReal` 227,3 → 259,1 ms. Fora do caminho de consulta; acontece uma vez por construção. Se errado, custa 20 + 32 ms por gravação de cache num cofre de 5 000 notas.

- [ ] **Step 1: Mover com `git mv` e ajustar o pacote**

```bash
git mv internal/writer/atomic.go internal/vault/atomic.go
git mv internal/writer/syncdir_unix.go internal/vault/syncdir_unix.go
git mv internal/writer/syncdir_windows.go internal/vault/syncdir_windows.go
git mv internal/writer/atomic_test.go internal/vault/atomic_test.go
git mv internal/writer/durabilidade_test.go internal/vault/durabilidade_test.go
git mv internal/writer/sweep_profundo_windows_test.go internal/vault/sweep_profundo_windows_test.go
```

Trocar `package writer` → `package vault` (e `writer_test` → `vault_test`) nos seis. Conferir que nenhum dos testes movidos usa símbolo de `writer` que ficou (`PathLocker`, `NormalizeEOL` de `section.go`) — se usar, o teste não migra inteiro: separar o subteste que depende de `writer` num arquivo que fica.

- [ ] **Step 2: Extrair `ReplaceFile`**

Em `internal/vault/atomic.go`, renomear `WriteAtomic` → `ReplaceFile` com o parâmetro `escrever func(*os.File) error`, substituir `tmpFile.Write(data)` por:

```go
	if err := escrever(tmpFile); err != nil {
		return fmt.Errorf("escrevendo no temporario %q: %w", tmpName, err)
	}
```

e criar `WriteAtomic` como o wrapper de quatro linhas. Docstring de `ReplaceFile` explica o callback (os caches em streaming) e mantém os cinco passos numerados que a docstring atual tem.

- [ ] **Step 3: Encaminhadores em `writer`**

Novo `internal/writer/atomic.go` só com os quatro encaminhadores acima e o `// Deprecated`. `writer` já importa `vault` — nenhuma aresta nova.

- [ ] **Step 4: `walk.go:74` e o comentário**

`strings.HasPrefix(name, ".gobsidian-tmp-")` → `strings.HasPrefix(name, TempFilePrefix)`.

Teste que prende a conta, em `internal/vault/walk_test.go`:

```go
// O arquivo e criado com o LITERAL, nao com a constante: e o que o disco tem
// de um binario antigo. Se a constante do filtro mudar, este teste e o que
// avisa que o lixo antigo passaria a entrar no indice.
func TestWalkIgnoraTemporarioDeBinarioAntigo(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gobsidian-tmp-abc123"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# a
"), 0o644); err != nil {
		t.Fatal(err)
	}
	var vistos []string
	err := vault.Walk(context.Background(), root, func(rel string, d fs.DirEntry) error {
		vistos = append(vistos, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(vistos) != 1 || vistos[0] != "a.md" {
		t.Fatalf("o temporario entrou no walk: %v", vistos)
	}
}
```

(Adaptar a assinatura de `vault.Walk` à real — ler `walk.go`.) Com a mutação da constante (`tmp` → `tmq`), o filtro deixa de casar o literal e o teste falha.

- [ ] **Step 5: Teste de `ReplaceFile` com callback que falha**

Em `internal/vault/atomic_test.go`:

```go
func TestReplaceFileCallbackFalhaNaoTocaOAlvo(t *testing.T) {
	alvo := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(alvo, []byte("antes"), 0o644); err != nil {
		t.Fatal(err)
	}
	quero := errors.New("codec falhou")
	err := vault.ReplaceFile(context.Background(), alvo, func(*os.File) error { return quero })
	if !errors.Is(err, quero) {
		t.Fatalf("err = %v, quero %v embrulhado", err, quero)
	}
	got, _ := os.ReadFile(alvo)
	if string(got) != "antes" {
		t.Fatalf("alvo mudou para %q", got)
	}
	restos, _ := filepath.Glob(filepath.Join(filepath.Dir(alvo), vault.TempFilePrefix+"*"))
	if len(restos) != 0 {
		t.Fatalf("temporario ficou para tras: %v", restos)
	}
}
```

- [ ] **Step 6: Gate, órfãos e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.
Run: `go test -race -count=3 ./internal/vault/ ./internal/writer/` — Expected: `ok` (os 1 000 ciclos de `atomic_test.go` agora em `vault`).

```bash
git add internal/vault/atomic.go internal/vault/syncdir_unix.go internal/vault/syncdir_windows.go internal/vault/atomic_test.go internal/vault/durabilidade_test.go internal/vault/sweep_profundo_windows_test.go internal/writer/atomic.go internal/vault/walk.go internal/vault/longpath_windows.go internal/vault/walk_test.go docs/ESTRUTURA.md docs/ARCHITECTURE.md
git commit -m "refactor(vault): atomic replace, write and temp sweep move to vault; writer forwards"
```

#### Verificações

- `git diff --stat -M <base>..HEAD` mostra os seis arquivos como **rename** (similaridade alta), não delete+add.
- `grep -rn "writer.WriteAtomic\|writer.SweepStaleTempFiles" --include=*.go internal cmd | grep -v _test | wc -l` = 7 (os seis de `service/write.go` + `servico.go:78`) — inalterado nesta PR.
- `mutate.ps1` do Step 4, exit 0, colado.
- Grafo: `go list -f '{{.Imports}}' ./internal/vault/` **não** ganha import de pacote interno (continua folha).
- `verify.ps1` verde; `-count=3` colado.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. `git mv` é permitido (é o que a tarefa pede).
- Nenhum chamador de `service` ou `cmd` muda nesta PR — é o que torna a PR2 revisável sozinha.
- `vault` continua folha: nenhum import de `internal/*`.
- Código de plataforma (`syncdir_*`) permanece atrás de build tag.

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/vault/atomic.go -Anchor 'TempFilePrefix = ".gobsidian-tmp-"' -Replacement 'TempFilePrefix = ".gobsidian-tmq-"' -Test TestWalkIgnoraTemporarioDeBinarioAntigo -Package ./internal/vault/` — exit 0.

#### Contrato de relatório

`task-171-report.md`: status, SHA, `git diff --stat -M`, o `grep` com 7, `go list` de `vault`, `mutate.ps1`, última linha do `verify.ps1`.

