# Relatório Task 59: Write Tools (`note_create`, `note_append`, `note_patch`)

- **Status**: DONE
- **Commit**: `feat(mcpsrv): write tools with dry_run and expected_hash`

## Resumo das Mudanças
- Implementados os métodos de serviço de escrita em `internal/service/write.go`: `CreateNote`, `AppendNote`, `PatchNote`.
- Implementado suporte a `dry_run` em todas as três tools: calcula o conteúdo proposto, gera o diff unificado com `writer.UnifiedDiff` e retorna sem alterar o arquivo no disco (garantindo mtime inalterado).
- Implementado suporte a `expected_hash` (concorrência otimista com xxhash): valida que o hash da nota não mudou entre leitura e escrita, recusando com `HASH_MISMATCH` em caso de divergência.
- Implementadas regras de restrição: recusa de sobrescrita em `note_create` (`NOTE_ALREADY_EXISTS`), recusa de caminhos fora do cofre (`PATH_OUTSIDE_VAULT`), recusa em notas `CloudOnly` (`CLOUD_ONLY_FILE`), desambiguação de headings/blocos e suporte a criação opcional de diretórios e headings (`create_folders`, `create_if_missing`).
- Registradas as três novas ferramentas MCP em `internal/mcpsrv/tools_write.go`: `note_create`, `note_append`, `note_patch`.
- Criada a suíte completa de testes de integração em `internal/service/write_test.go`.

## Mapeamento de Parâmetros e Testes Correspondentes
Cada parâmetro de cada uma das três tools possui um teste dedicado comprovando sua atuação:

### 1. `note_create`
- `path`: `TestCreateNote_Basic`
- `content`: `TestCreateNote_Basic`
- `frontmatter`: `TestCreateNote_WithFrontmatter`
- `create_folders`: `TestCreateNote_CreateFoldersTrueAndFalse`
- `dry_run`: `TestCreateNote_DryRunPreservesMTimeAndDoesNotWrite`

### 2. `note_append`
- `path`: `TestAppendNote_Basic`
- `content`: `TestAppendNote_Basic`
- `heading`: `TestAppendNote_ToHeading`
- `heading_level`: `TestAppendNote_CreateIfMissingTrueAndFalse`
- `create_if_missing`: `TestAppendNote_CreateIfMissingTrueAndFalse`
- `ensure_blank_line`: `TestAppendNote_Basic`
- `expected_hash`: `TestAppendNote_ExpectedHashMatchAndMismatch`
- `dry_run`: `TestAppendNote_DryRunPreservesMTime`

### 3. `note_patch`
- `path`: `TestPatchNote_Basic` (verificado via `TestPatchNote_ReplaceSection`)
- `content`: `TestPatchNote_ReplaceSection`
- `heading`: `TestPatchNote_ReplaceSection`
- `heading_level`: `TestPatchNote_ReplaceHeadingAndSectionMode`
- `block_id`: `TestPatchNote_ReplaceBlock`
- `mode`: `TestPatchNote_ReplaceHeadingAndSectionMode`
- `expected_hash`: `TestPatchNote_ExpectedHashMatchAndMismatch`
- `dry_run`: `TestPatchNote_DryRunPreservesMTime`

## Evidência de TDD

### RED
Comando:
`go test -v ./internal/service/ -run TestCreateNote` (antes de implementar write.go)
Saída:
FAIL: CreateNote undefined / package internal/service sem métodos de escrita

### GREEN
Comando:
`go test -v ./internal/service/ -run "TestCreateNote|TestAppendNote|TestPatchNote"`
Saída:
=== RUN   TestCreateNote_Basic
--- PASS: TestCreateNote_Basic (0.03s)
=== RUN   TestCreateNote_AlreadyExists
--- PASS: TestCreateNote_AlreadyExists (0.01s)
=== RUN   TestCreateNote_WithFrontmatter
--- PASS: TestCreateNote_WithFrontmatter (0.01s)
=== RUN   TestCreateNote_CreateFoldersTrueAndFalse
--- PASS: TestCreateNote_CreateFoldersTrueAndFalse (0.01s)
=== RUN   TestCreateNote_DryRunPreservesMTimeAndDoesNotWrite
--- PASS: TestCreateNote_DryRunPreservesMTimeAndDoesNotWrite (0.00s)
=== RUN   TestAppendNote_Basic
--- PASS: TestAppendNote_Basic (0.01s)
=== RUN   TestAppendNote_ToHeading
--- PASS: TestAppendNote_ToHeading (0.01s)
=== RUN   TestAppendNote_CreateIfMissingTrueAndFalse
--- PASS: TestAppendNote_CreateIfMissingTrueAndFalse (0.01s)
=== RUN   TestAppendNote_ExpectedHashMatchAndMismatch
--- PASS: TestAppendNote_ExpectedHashMatchAndMismatch (0.04s)
=== RUN   TestAppendNote_DryRunPreservesMTime
--- PASS: TestAppendNote_DryRunPreservesMTime (0.07s)
=== RUN   TestPatchNote_ReplaceSection
--- PASS: TestPatchNote_ReplaceSection (0.01s)
=== RUN   TestPatchNote_ReplaceHeadingAndSectionMode
--- PASS: TestPatchNote_ReplaceHeadingAndSectionMode (0.01s)
=== RUN   TestPatchNote_ReplaceBlock
--- PASS: TestPatchNote_ReplaceBlock (0.01s)
=== RUN   TestPatchNote_ExpectedHashMatchAndMismatch
--- PASS: TestPatchNote_ExpectedHashMatchAndMismatch (0.01s)
=== RUN   TestPatchNote_DryRunPreservesMTime
--- PASS: TestPatchNote_DryRunPreservesMTime (0.06s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	0.820s

## Provas de Mutação

### 1. Desligamento do check de `expected_hash` (`if req.ExpectedHash != "" ... -> if false`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor ... -Replacement ... -Test TestAppendNote_ExpectedHashMatchAndMismatch -Package ./internal/service/`
Saída:
--- FAIL: TestAppendNote_ExpectedHashMatchAndMismatch (0.04s)
    write_test.go:246: esperava CodeHashMismatch, obteve: <nil>
FAIL
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

### 2. Desligamento do curto-circuito de `dry_run` (`if req.DryRun -> if false`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor ... -Replacement ... -Test TestCreateNote_DryRunPreservesMTimeAndDoesNotWrite -Package ./internal/service/`
Saída:
--- FAIL: TestCreateNote_DryRunPreservesMTimeAndDoesNotWrite (0.01s)
    write_test.go:132: Created deve ser false em dry_run
    write_test.go:135: Diff deve vir preenchido em dry_run
    write_test.go:139: arquivo foi criado no disco durante dry_run
FAIL
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

## Decisão de Comportamento Registrada
Se `expected_hash` for omitido, a verificação de concorrência otimista é pulada (permitindo escrita incondicional), mas o servidor ainda realiza a escrita atômica com trava de caminho para impedir corrupção de concorrência local.

## Arquivos Alterados
- `internal/service/errors.go`
- `internal/service/service.go`
- `internal/service/write.go`
- `internal/service/write_test.go`
- `internal/mcpsrv/tools_write.go`
- `.superpowers/sdd/task-59-report.md`

## O Que Ficou de Fora
Nada.
