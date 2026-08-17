# Relatório Task 116 — `expected_hash` contra os bytes lidos

## Status
DONE

## Commit
`f76eeaf` fix(service): validate expected_hash against disk bytes instead of index

## Mudanças por arquivo
- `internal/service/write.go`: `currentHash` passa a ser derivado de `hashDoConteudo(raw)` (`xxhash.Sum64(raw)`), e os retornos `CreateNoteResult`, `AppendNoteResult` e `PatchNoteResult` passam a devolver a propriedade `Hash` recalculada do conteúdo gravado.
- `internal/service/expected_hash_test.go`: criado com `TestExpectedHashPegaEdicaoExternaAindaNaoIndexada` e `TestExpectedHashCorretoAindaPassa`.

## Evidência de TDD

### RED (Antes do fix em `write.go`)
Comando: `go test -v ./internal/service/ -run TestExpectedHash`
Saída literal:
```
=== RUN   TestExpectedHashPegaEdicaoExternaAindaNaoIndexada
=== RUN   TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_append
    expected_hash_test.go:79: a escrita aceitou expected_hash obsoleto e sobrescreveu a edicao externa — o controle otimista falhou no seu unico caso
=== RUN   TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_patch
    expected_hash_test.go:79: a escrita aceitou expected_hash obsoleto e sobrescreveu a edicao externa — o controle otimista falhou no seu unico caso
--- FAIL: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada (0.05s)
    --- FAIL: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_append (0.04s)
    --- FAIL: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_patch (0.01s)
=== RUN   TestExpectedHashCorretoAindaPassa
--- PASS: TestExpectedHashCorretoAindaPassa (0.01s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.941s
FAIL
```

### GREEN (Após o fix em `write.go`)
Comando: `go test -v ./internal/service/ -run TestExpectedHash`
Saída literal:
```
=== RUN   TestExpectedHashPegaEdicaoExternaAindaNaoIndexada
=== RUN   TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_append
=== RUN   TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_patch
--- PASS: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada (0.04s)
    --- PASS: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_append (0.02s)
    --- PASS: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_patch (0.01s)
=== RUN   TestExpectedHashCorretoAindaPassa
--- PASS: TestExpectedHashCorretoAindaPassa (0.03s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	1.403s
```

## Prova de Mutação

### Mutação 1: `AppendNote`
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor ... -Replacement ... -Test TestExpectedHashPegaEdicaoExternaAindaNaoIndexada -Package ./internal/service/`

Saída literal:
```
[...] Mutando internal/service/write.go
      - 	raw, err := os.ReadFile(absPath)
	if err != nil {
		return AppendNoteResult{}, Errorf(CodeInternal, "lendo nota %q: %v", req.Path, err)
	}

	currentHash := hashDoConteudo(raw)
      + 	raw, err := os.ReadFile(absPath)
	if err != nil {
		return AppendNoteResult{}, Errorf(CodeInternal, "lendo nota %q: %v", req.Path, err)
	}

	currentHash := fmt.Sprintf("%016x", note.Hash)

[...] go test -race -run TestExpectedHashPegaEdicaoExternaAindaNaoIndexada ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada (0.08s)
    --- FAIL: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_append (0.06s)
        expected_hash_test.go:79: a escrita aceitou expected_hash obsoleto e sobrescreveu a edicao externa — o controle otimista falhou no seu unico caso
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.377s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```
Resultado: `EXIT=0`. Asserção que falhou: `expected_hash_test.go:79` (`a escrita aceitou expected_hash obsoleto e sobrescreveu a edicao externa...`).

### Mutação 2: `PatchNote`
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor ... -Replacement ... -Test TestExpectedHashPegaEdicaoExternaAindaNaoIndexada -Package ./internal/service/`

Saída literal:
```
[...] Mutando internal/service/write.go
      - 	raw, err := os.ReadFile(absPath)
	if err != nil {
		return PatchNoteResult{}, Errorf(CodeInternal, "lendo nota %q: %v", req.Path, err)
	}

	currentHash := hashDoConteudo(raw)
      + 	raw, err := os.ReadFile(absPath)
	if err != nil {
		return PatchNoteResult{}, Errorf(CodeInternal, "lendo nota %q: %v", req.Path, err)
	}

	currentHash := fmt.Sprintf("%016x", note.Hash)

[...] go test -race -run TestExpectedHashPegaEdicaoExternaAindaNaoIndexada ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada (0.04s)
    --- FAIL: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_patch (0.02s)
        expected_hash_test.go:79: a escrita aceitou expected_hash obsoleto e sobrescreveu a edicao externa — o controle otimista falhou no seu unico caso
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.753s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```
Resultado: `EXIT=0`. Asserção que falhou: `expected_hash_test.go:79` (`a escrita aceitou expected_hash obsoleto e sobrescreveu a edicao externa...`).

## Verificações do Brief

1. `sdd base 116`:
```
[i] Plano: 2026-08-16-revisao-fixes
[OK] Base da Task 116: 8de4390
```

2. `go test` executado e subtestes rodaram sem pular:
```
=== RUN   TestExpectedHashPegaEdicaoExternaAindaNaoIndexada
=== RUN   TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_append
=== RUN   TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_patch
--- PASS: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada (0.04s)
    --- PASS: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_append (0.02s)
    --- PASS: TestExpectedHashPegaEdicaoExternaAindaNaoIndexada/note_patch (0.01s)
=== RUN   TestExpectedHashCorretoAindaPassa
--- PASS: TestExpectedHashCorretoAindaPassa (0.03s)
PASS
```

3. `note_read` (`ReadNote`) Hash origin check:
Observado em `internal/service/read.go:240`: `ReadNote` retorna `Hash` diretamente de `fmt.Sprintf("%016x", note.Hash)` (do índice de metadados). Como `ReadNote` é operação somente-leitura e pode ler apenas faixas (`ReadRange`), reler e recalcular o hash de arquivos inteiros do disco a cada leitura traria custo computacional e I/O desnecessário. Nenhuma alteração feita em `read.go`, conforme instruído em `USER_REQUEST`.

4. Bateria de verificação (`pwsh -File scripts/verify.ps1`):
12 das 13 etapas executadas com `[OK]`. A etapa 11 (`check_tool_params`) exibiu alerta pré-existente (`tagListInput.Sort` e `tagListInput.Hierarchical` declarados e não lidos), que é o objeto da Task 104 e não desta tarefa.

5. `golangci-lint version`:
```
golangci-lint has version 2.12.2 built with go1.26.4
```

## O que ficou de fora
Vazio (nada ficou de fora).

## `git status --porcelain`
```
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
```
(Apenas a atualização do ledger de progresso, pronta para o commit final).
