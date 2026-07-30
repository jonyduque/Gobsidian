# Relatório Task 60: `--read-only` verificado no `ListTools`

- **Status**: DONE
- **Commit**: `feat(mcpsrv): write tools absent from ListTools under read-only`

## Resumo das Mudanças
- Verificado que as ferramentas de escrita (`note_create`, `note_append`, `note_patch`) ficam **ausentes** do retorno de `ListTools` quando o servidor roda com `--read-only` (comprovando o requisito RF-51 e a decisão de não anunciar ferramentas que seriam rejeitadas).
- Verificado que as ferramentas de leitura (`vault_stats`, `note_read`, `vault_search`, etc.) continuam normalmente presentes tanto em modo leitura quanto escrita.
- Verificado que o watcher continua ativo quando `--read-only` está ligado (verificado via `vault_stats` com `include_runtime: true`).
- Verificado que o campo `ReadOnlySet` em `config.Flags` está corretamente preenchido via `cmd.Flags().Changed("read-only")` nos subcomandos `serve` e `doctor`.
- Criados os testes unitários e de integração em `internal/mcpsrv/tools_write_test.go`.

## As Verificações do Brief
- Com `--read-only`, `ListTools` **não** contém `note_create`, `note_append`, `note_patch` (verificado via `TestListTools_ReadOnlyTrue`).
- Sem `--read-only`, `ListTools` **contém** as três ferramentas de escrita (verificado via `TestListTools_ReadOnlyFalse`).
- As ferramentas de leitura continuam presentes em ambos os modos.
- O watcher continua ativo com `--read-only` (verificado via `TestWatcherActiveUnderReadOnly`).
- `ReadOnlySet` preenchido em `serve` e `doctor` (`cmd/gobsidian/serve.go:27` e `cmd/gobsidian/doctor.go:26`).

## Evidência de TDD

### RED
Comando:
`pwsh -Command '$a = @"\n\tif !cfg.ReadOnly {\n\t\ts.registerWriteTools()\n\t}\n"@; $r = @"\n\tif true {\n\t\ts.registerWriteTools()\n\t}\n"@; pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/server.go -Anchor $a -Replacement $r -Test TestListTools_ReadOnlyTrue -Package ./internal/mcpsrv/'`
Saída:
--- FAIL: TestListTools_ReadOnlyTrue (0.08s)
    tools_write_test.go:66: tool de escrita "note_create" NAO deveria estar presente sob --read-only
    tools_write_test.go:66: tool de escrita "note_append" NAO deveria estar presente sob --read-only
    tools_write_test.go:66: tool de escrita "note_patch" NAO deveria estar presente sob --read-only
FAIL

### GREEN
Comando:
`go test -v ./internal/mcpsrv/ -run "TestListTools|TestWatcherActive"`
Saída:
=== RUN   TestListTools_ReadOnlyTrue
--- PASS: TestListTools_ReadOnlyTrue (0.01s)
=== RUN   TestListTools_ReadOnlyFalse
--- PASS: TestListTools_ReadOnlyFalse (0.01s)
=== RUN   TestWatcherActiveUnderReadOnly
--- PASS: TestWatcherActiveUnderReadOnly (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/mcpsrv	0.065s

## Prova de Mutação

### Ignorar `--read-only` no registro (`if !cfg.ReadOnly` -> `if true`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/server.go -Anchor ... -Replacement ... -Test TestListTools_ReadOnlyTrue -Package ./internal/mcpsrv/`
Saída:
--- FAIL: TestListTools_ReadOnlyTrue (0.08s)
    tools_write_test.go:66: tool de escrita "note_create" NAO deveria estar presente sob --read-only
    tools_write_test.go:66: tool de escrita "note_append" NAO deveria estar presente sob --read-only
    tools_write_test.go:66: tool de escrita "note_patch" NAO deveria estar presente sob --read-only
FAIL
[OK] internal/mcpsrv/server.go restaurado byte a byte (SHA-256 confere).

## Arquivos Alterados
- `internal/mcpsrv/tools_write_test.go`
- `.superpowers/sdd/task-60-report.md`

## O Que Ficou de Fora
Nada.
