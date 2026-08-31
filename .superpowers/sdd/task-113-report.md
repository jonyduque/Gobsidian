# Relatório da Task 113: vault.Resolve como portão único de escrita

- **Status**: DONE
- **Commit**: `84aadde` (`fix(service): vault.Resolve as single write gate and checkWritable`)

---

## 1. Alterações Realizadas

1. **Refatoração em `internal/service/write.go`**:
   - `CreateNote`: Utiliza `vault.Resolve(s.vault.Root(), req.Path)` como portão único para validação e resolução do caminho canonical. Removida conversão explícita `vault.CanonicalPath(...)`.
   - `MoveNote`: Utiliza `vault.Resolve(s.vault.Root(), req.To)` e `vault.Resolve(s.vault.Root(), toInput)` para sanitização e validação de caminhos do destino. Removidas conversões explícitas `vault.CanonicalPath(...)`.
   - `DeleteNote` e `AppendNote`: Atualizados para utilizar `vault.Resolve` onde aplicável.
   - `checkWriteAllowed` foi renomeada para `checkWritable()` com responsabilidade única de validar o modo somente-leitura (`s.opts.ReadOnly`).
   - Criada a função `mapVaultErr(err error) *Error` para mapear de forma desacoplada sentinelas de erro do pacote `vault` (`ErrOutsideVault`, `ErrAbsolutePath`, `ErrEmptyPath`, `ErrInvalidPath`) para os códigos MCP correspondentes (`CodePathOutsideVault`, `CodeInvalidArgument`).

2. **Criação da Suíte de Testes `internal/service/write_traversal_test.go`**:
   - Transcrito o teste `TestEscritaRecusaTravessiaComSeparadorDoWindows` em `package service`.

---

## 2. Evidência de TDD

### RED

Comando executado antes da resolução completa dos caminhos de escrita:
```bash
go test ./internal/service/ -run TestEscritaRecusaTravessiaComSeparadorDoWindows
```

Saída:
```
--- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows (0.01s)
    --- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows/COM1 (0.00s)
        write_traversal_test.go:51: MoveNote(to="COM1") devolveu sucesso
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.816s
```

### GREEN

Comando executado após refatoração com `vault.Resolve`:
```bash
go test ./internal/service/ -run TestEscritaRecusaTravessiaComSeparadorDoWindows
```

Saída:
```
ok  	github.com/jonyd/gobsidian/internal/service	1.567s
```

---

## 3. Medição Inicial de Cobertura (Antes da Correção)

Remoção temporária das verificações de travessia e execução da suíte completa de testes (`go test ./internal/...`):

```
--- FAIL: TestMoveNote_OutsideVaultAndAlreadyExists (0.01s)
    move_test.go:201: MoveNote() erro = <nil>, querCode = CodePathOutsideVault
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.854s
```

**Resultado:** Exatamente **1 teste falhou** (`TestMoveNote_OutsideVaultAndAlreadyExists`). O `CreateNote` não possuía cobertura prévia para tentativas de travessia de diretório, comprovando a lacuna de cobertura que a Task 113 corrigiu.

---

## 4. Provas de Mutação (`scripts/mutate.ps1`)

### Mutação 1 (`CreateNote` `vault.Resolve` mutado)

**Comando:**
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor "absPath, canonical, err := vault.Resolve(s.vault.Root(), req.Path)" -Replacement "canonical, err := vault.CanonicalPath(filepath.ToSlash(filepath.Clean(req.Path))), error(nil); absPath := s.vault.Abs(canonical)" -Test TestEscritaRecusaTravessiaComSeparadorDoWindows -Package ./internal/service/
```

**Saída:**
```
[...] Mutando internal/service/write.go
      - absPath, canonical, err := vault.Resolve(s.vault.Root(), req.Path)
      + canonical, err := vault.CanonicalPath(filepath.ToSlash(filepath.Clean(req.Path))), error(nil); absPath := s.vault.Abs(canonical)

[...] go test -race -run TestEscritaRecusaTravessiaComSeparadorDoWindows ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows (0.15s)
    --- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows/..\..\x.md (0.01s)
        write_traversal_test.go:38: CreateNote("..\..\x.md") devolveu sucesso
    --- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows/..\x.md (0.01s)
        write_traversal_test.go:38: CreateNote("..\x.md") devolveu sucesso
    --- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows/sub\..\..\x.md (0.00s)
        write_traversal_test.go:45: CreateNote("sub\..\..\x.md") gravou "x.md" fora do cofre
    --- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows/../../x.md (0.00s)
        write_traversal_test.go:45: CreateNote("../../x.md") gravou "x.md" fora do cofre
    --- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows//etc/passwd (0.00s)
        write_traversal_test.go:45: CreateNote("/etc/passwd") gravou "x.md" fora do cofre
    --- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows/C:\Windows\Temp\x.md (0.00s)
        write_traversal_test.go:45: CreateNote("C:\Windows\Temp\x.md") gravou "x.md" fora do cofre
    --- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows/COM1 (0.01s)
        write_traversal_test.go:38: CreateNote("COM1") devolveu sucesso
    --- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows/nota\x00.md (0.11s)
        write_traversal_test.go:45: CreateNote("nota\x00.md") gravou "x.md" fora do cofre
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	2.407s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

`EXIT=0`.

---

### Mutação 2 (`MoveNote` `vault.Resolve` mutado)

**Comando:**
```powershell
pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor "_, canonicalTo, err := vault.Resolve(s.vault.Root(), req.To)" -Replacement "canonicalTo, err := vault.CanonicalPath(filepath.ToSlash(filepath.Clean(req.To))), error(nil)" -Test TestEscritaRecusaTravessiaComSeparadorDoWindows -Package ./internal/service/
```

**Saída:**
```
[...] Mutando internal/service/write.go
      - _, canonicalTo, err := vault.Resolve(s.vault.Root(), req.To)
      + canonicalTo, err := vault.CanonicalPath(filepath.ToSlash(filepath.Clean(req.To))), error(nil)

[...] go test -race -run TestEscritaRecusaTravessiaComSeparadorDoWindows ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows (0.02s)
    --- FAIL: TestEscritaRecusaTravessiaComSeparadorDoWindows/COM1 (0.01s)
        write_traversal_test.go:51: MoveNote(to="COM1") devolveu sucesso
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	2.059s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

`EXIT=0`.

---

## 5. Verificações do Ambiente

### Execução de `verify.ps1`:
```
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 4. go vet (windows)
[OK] go vet (windows)
[...] 5. go vet (linux)
[OK] go vet (linux)
[...] 6. go vet (darwin)
[OK] go vet (darwin)
[...] 7. gofmt
[OK] gofmt
[...] 8. golangci-lint
[OK] golangci-lint
[...] 9. golangci-lint (linux)
[OK] golangci-lint (linux)
[...] 10. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 11. check_tool_params
[OK] check_tool_params
[...] 12. check_doc_refs
[OK] check_doc_refs
[...] 13. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

### Execução de `check_net.ps1`:
```
[OK] Nenhum pacote de internal/ ou cmd/ importa net/* ou abre socket que saia da maquina (verificado via netcheck vettool em windows, linux, darwin)
```

### Versão do `golangci-lint`:
```
golangci-lint has version 2.12.2 built with go1.26.4
```

---

## 6. O que ficou de fora

Nenhum item do escopo ficou de fora.

---

## 7. Estado do Git

`git status --porcelain` retorna vazio.
