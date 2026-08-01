# Relatório Task 75: Subcomandos `index`, `search` e `inspect` (`cmd/gobsidian`)

- **Status**: DONE
- **Commit**: `feat(cmd): index, search and inspect subcommands (RF-52)`

## O Que Foi Implementado
- Criados os subcomandos CLI `index`, `search` e `inspect` em [cmd/gobsidian/index.go](file:///C:/Users/jonyd/Projetos/Gobsidian/cmd/gobsidian/index.go), [cmd/gobsidian/search.go](file:///C:/Users/jonyd/Projetos/Gobsidian/cmd/gobsidian/search.go) e [cmd/gobsidian/inspect.go](file:///C:/Users/jonyd/Projetos/Gobsidian/cmd/gobsidian/inspect.go) conforme o requisito RF-52 do PRD.
- Todos os três subcomandos suportam saída em texto plano (com marcadores ASCII puros: `[OK]`, `[*]`, `[!]`, `[i]`) e saída estruturada via a flag `--json`.
- Registrados e conectados em [cmd/gobsidian/main.go](file:///C:/Users/jonyd/Projetos/Gobsidian/cmd/gobsidian/main.go).
- Preenchidas obrigatoriamente as flags `ReadOnlySet` e `DebounceMSSet` através de `cmd.Flags().Changed(...)` antes da chamada a `config.Load(flags)`.
- Adicionado comentário explicativo em todos os três comandos documentando que a escrita em `stdout` é feita deliberadamente por se tratar de comandos CLI e não de servidores MCP JSON-RPC.
- Criada a suíte de testes em [cmd/gobsidian/cli_subcommands_test.go](file:///C:/Users/jonyd/Projetos/Gobsidian/cmd/gobsidian/cli_subcommands_test.go).

## Evidência de TDD

### Comando do RED
`go test -v ./cmd/gobsidian/... -run "TestIndexCmd|TestSearchCmd|TestInspectCmd"` (antes de criar os arquivos de subcomando)
```
# github.com/jonyd/gobsidian/cmd/gobsidian [github.com/jonyd/gobsidian/cmd/gobsidian.test]
cmd\gobsidian\cli_subcommands_test.go:34:10: undefined: newIndexCmd
cmd\gobsidian\cli_subcommands_test.go:50:14: undefined: newIndexCmd
cmd\gobsidian\cli_subcommands_test.go:76:10: undefined: newSearchCmd
cmd\gobsidian\cli_subcommands_test.go:92:14: undefined: newSearchCmd
cmd\gobsidian\cli_subcommands_test.go:119:10: undefined: newInspectCmd
cmd\gobsidian\cli_subcommands_test.go:135:14: undefined: newInspectCmd
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian [build failed]
```

### Comando do GREEN
`go test -v ./cmd/gobsidian/...` (após implementar)
```
=== RUN   TestIndexCmd_StdoutAndJSON
--- PASS: TestIndexCmd_StdoutAndJSON (0.11s)
=== RUN   TestSearchCmd_StdoutAndJSON
--- PASS: TestSearchCmd_StdoutAndJSON (0.09s)
=== RUN   TestInspectCmd_StdoutAndJSON
--- PASS: TestInspectCmd_StdoutAndJSON (0.12s)
=== RUN   TestSubcommands_FlagsSetPopulated
--- PASS: TestSubcommands_FlagsSetPopulated (0.05s)
=== RUN   TestIndexCmd_DebounceMSFlagZeroRejected
--- PASS: TestIndexCmd_DebounceMSFlagZeroRejected (0.04s)
PASS
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	1.760s
```

## Prova de Mutação

### Comando de Mutação
`pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/index.go -Anchor 'flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")' -Replacement 'flags.DebounceMSSet = false' -Test TestIndexCmd_DebounceMSFlagZeroRejected -Package ./cmd/gobsidian/`

### Saída Real do Script
```
[...] Mutando cmd/gobsidian/index.go
      - flags.DebounceMSSet = cmd.Flags().Changed("debounce-ms")
      + flags.DebounceMSSet = false

[...] go test -race -run TestIndexCmd_DebounceMSFlagZeroRejected ./cmd/gobsidian/
----------------------------------------------------------------------
--- FAIL: TestIndexCmd_DebounceMSFlagZeroRejected (0.06s)
    cli_subcommands_test.go:209: esperava erro ao passar --debounce-ms=0, mas obteve nil
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	2.224s
FAIL
----------------------------------------------------------------------
[OK] cmd/gobsidian/index.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

## Verificações Exigidas pelo Brief

| Verificação | Resultado Real |
|---|---|
| Um teste por subcomando em `cmd/gobsidian` | OK (testados `index`, `search`, `inspect` e flags) |
| `stdout` e `stderr` capturados separadamente (`stderr` vazio em execução normal) | OK (`stderr` validado com tamanho 0 em todos os testes) |
| `--json` produz JSON válido (`json.Unmarshal`) | OK (`json.Unmarshal` validado com sucesso para os 3 comandos) |
| Saída de console em ASCII puro no modo não-JSON | OK (`[OK]`, `[*]`, `[i]`) |
| Nenhum subcomando novo quebrou `serve` (100 ciclos de órfãos) | OK (100/100 ciclos sem órfãos) |

## Auditoria de Relatório
`pwsh -File scripts/audit_reports.ps1 75` executado (0 achados).

## Bateria de Verificação
`pwsh -File scripts/verify.ps1`: **9/9 etapas VERDES**.
`pwsh -File scripts/build.ps1`: **Binário gerado com sucesso**.

## Arquivos Alterados
- `cmd/gobsidian/index.go`
- `cmd/gobsidian/search.go`
- `cmd/gobsidian/inspect.go`
- `cmd/gobsidian/main.go`
- `cmd/gobsidian/cli_subcommands_test.go`
- `.superpowers/sdd/task-75-report.md`

## O Que Ficou de Fora
Nada.

## git status --porcelain
```
?? .superpowers/sdd/2026-07-25-gobsidian-v01/task-75-base.txt
?? .superpowers/sdd/task-75-report.md
 M cmd/gobsidian/cli_subcommands_test.go
 M cmd/gobsidian/index.go
 M cmd/gobsidian/inspect.go
 M cmd/gobsidian/main.go
 M cmd/gobsidian/search.go
```
