### Task 172: c1 PR2 — chamadores migram, caches usam `ReplaceFile`, encaminhadores somem

**Files:**
- Modify: `internal/service/write.go:149,:248,:378,:619,:751,:832` (`writer.WriteAtomic` → `vault.WriteAtomic`)
- Modify: `cmd/gobsidian/servico.go:78` (`writer.SweepStaleTempFiles` → `vault.SweepStaleTempFiles`)
- Modify: `internal/index/persist.go:124-146` (`SaveIndexCache` usa `vault.ReplaceFile`)
- Modify: `internal/search/persist.go:87-116` (`SaveInvertedCache` usa `vault.ReplaceFile`; `promoverArenaSePresente(inv)` **antes** de `ExportForCache`)
- Delete: `internal/writer/atomic.go` (os encaminhadores)
- Modify: testes de `service` que referenciam `writer.TempFilePrefix` (encontrar com `gopls references`)
- Modify: `docs/ESTADO.md` (formato de cache: nota de que a gravação agora tem `fsync`)

**Interfaces:**
- Consumes: `vault.ReplaceFile`, `vault.WriteAtomic`, `vault.SweepStaleTempFiles`, `vault.TempFilePrefix` (Task 171).
- Consumes: `TestSaveOverwritesMappedCache` (`internal/search/persist_test.go:65`) — é o teste que prende a ordem `promover → rename`.

- [ ] **Step 1: `gopls rename`? Não — é troca de pacote.** Substituir por `sed` guiado pela lista de **Files** e confirmar com `go build ./...`. Import de `writer` em `service/write.go` permanece (usa `PathLocker`, seções); em `servico.go`, se `writer` ficar sem uso, remover o import.

- [ ] **Step 2: `SaveIndexCache` com `ReplaceFile`**

```go
	finalPath := filepath.Join(cacheDir, indexCacheFileName)
	if err := vault.ReplaceFile(ctx, finalPath, func(f *os.File) error {
		return escreveIndexCache(f, header, notes, assets)
	}); err != nil {
		return fmt.Errorf("gravando cache de indice em %q: %w", finalPath, err)
	}
	return nil
```

`index` já importa `vault` — nenhuma aresta nova.

- [ ] **Step 3: `SaveInvertedCache` com `ReplaceFile`**

A promoção da arena precisa vir **antes** do rename; com `ReplaceFile` o rename está dentro, então a promoção vem antes da chamada inteira — e antes do `ExportForCache`, para que os slices exportados já apontem para o heap:

```go
	// Promove a arena ANTES de exportar e gravar: ReplaceFile faz o rename
	// por dentro, e o rename falha no Windows enquanto o alvo esta mapeado
	// (ver promoverArenaSePresente, em mmap.go). Exportar depois de promover
	// garante que os slices gravados nao apontam para o mapeamento fechado.
	promoverArenaSePresente(inv)
	termos, docLengths := inv.ExportForCache()

	finalPath := filepath.Join(cacheDir, "inverted_cache.gob")
	if err := vault.ReplaceFile(ctx, finalPath, func(f *os.File) error {
		return escreveCache(f, header, termos, docLengths)
	}); err != nil {
		return fmt.Errorf("gravando cache de busca em %q: %w", finalPath, err)
	}
	return nil
```

Run: `go test -race ./internal/search/ -run 'TestSaveOverwritesMappedCache|TestSaveAndLoadInvertedCache|TestIndiceRecarregadoEIdenticoAoConstruido' -v` — Expected: PASS nos três.

Prova de que a ordem importa: mover temporariamente `promoverArenaSePresente(inv)` para **depois** do `ReplaceFile`, rodar `TestSaveOverwritesMappedCache` — Expected: FAIL (o rename falha com o arquivo mapeado). Restaurar. Colar.

- [ ] **Step 4: Apagar os encaminhadores**

`git rm internal/writer/atomic.go`. `go build ./...` — Expected: limpo (ninguém mais usa). `grep -rn "writer\.\(WriteAtomic\|SweepStaleTempFiles\|TempFilePrefix\|SweepResult\)" --include=*.go .` — Expected: vazio.

- [ ] **Step 5: `benchstat` da gravação**

`SaveIndexCacheReal` e `SaveInvertedCacheReal` antes (binário `antes_index`/`antes_search`) × depois, 7 intercalados. Esperado: `sec/op` sobe para perto das linhas `ComFsync` da Baseline (41,87 ms / 259,1 ms) — é o custo decidido no ruling da Task 171; **colar e registrar em `docs/ESTADO.md`** como medição publicada. Se subir **além** disso com `p < 0.05`, algo além do fsync entrou: investigar antes de commitar.

- [ ] **Step 6: Gate, órfãos e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.
Run: `pwsh -File scripts/test_orphans.ps1` — Expected: quatro `[OK]` (o boot foi tocado).

```bash
git add internal/service/write.go cmd/gobsidian/servico.go internal/index/persist.go internal/search/persist.go internal/writer/atomic.go <testes tocados> docs/ESTADO.md
git commit -m "refactor: callers use vault atomic write; caches replace through vault.ReplaceFile with fsync"
```

#### Verificações

- `grep` do Step 4 vazio, colado.
- FAIL do Step 3 (ordem da promoção) colado.
- `benchstat` do Step 5 colado; números publicados em `ESTADO.md`.
- `go list -f '{{.Imports}}' ./internal/index/ ./internal/search/` não ganha `writer`.
- `test_orphans.ps1` quatro `[OK]`; `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. `git rm` por caminho é permitido.
- Formato de cache (`CacheFormatVersion`, `IndexCacheFormatVersion`) **não** muda: o que muda é como o arquivo chega ao disco, não o que tem dentro. Prova: `TestSaveAndLoadInvertedCache` e o equivalente de `index` carregam um cache gravado pelo binário `antes` (gerar um com `antes_search.test.exe -run TestSaveAndLoad… ` e apontar o teste para ele se houver mecanismo; se não, registrar "compatibilidade inferida do formato intacto, não testada" — honesto).
- Nenhuma aresta nova: `index`, `search`, `service` já importam `vault`.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: a regra provada é a ordem `promover → gravar`, e a prova é a inversão manual do Step 3 com o FAIL colado.

#### Contrato de relatório

`task-172-report.md`: status, SHA, `grep` vazio, FAIL do Step 3, `benchstat`, `go list`, `test_orphans.ps1`, última linha do `verify.ps1`.

---

