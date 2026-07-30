# Relatório Task 55: `internal/writer/atomic.go` — Escrita Atômica e Gate RNF-11

- **Status**: DONE
- **Commit**: `feat(writer): atomic write with fsync and rename retry`

## Resumo das Mudanças
- Implementada a função `WriteAtomic(targetPath string, data []byte) error` em `internal/writer/atomic.go`.
- Garante criação do arquivo temporário no **mesmo diretório do alvo** com prefixo `.gobsidian-tmp-*`, essencial para suportar `os.Rename` atômico no mesmo volume.
- Incluído `tmpFile.Sync()` obrigatório antes do fechamento do temporário para flush físico ao disco contra falhas de energia.
- Implementado loop de retry no `os.Rename` (10 tentativas com intervalo de 10ms) para contornar bloqueios temporários de arquivo comuns em ambiente Windows (antivírus, indexador).
- Criada a rotina `CleanStaleTempFiles(dir)` para higienização de temporários órfãos de escritas interrompidas.
- Criado a suíte de testes de estresse `TestRNF11NoCorruptionUnder1000Crashes` em `internal/writer/atomic_test.go`, disparando 1.000 iterações com término forçado de subprocesso (`cmd.Process.Kill()`) em instantes variados.

## Evidência de TDD

### RED
Comando:
`go test -v ./internal/writer/ -run TestWriteAtomic` (antes de criar atomic.go)
Saída:
FAIL: WriteAtomic undefined / package internal/writer sem atomic.go

### GREEN
Comando:
`go test -v ./internal/writer/ -run TestRNF11NoCorruptionUnder1000Crashes`
Saída:
=== RUN   TestRNF11NoCorruptionUnder1000Crashes
    atomic_test.go:92: RNF-11: 0 corrompidas em 1000 iteracoes
--- PASS: TestRNF11NoCorruptionUnder1000Crashes (32.21s)
PASS
ok  	github.com/jonyd/gobsidian/internal/writer	33.235s

Comando (suíte curta completa):
`go test -v ./internal/writer/ -short`
Saída:
=== RUN   TestRNF11NoCorruptionUnder1000Crashes
    atomic_test.go:50: 1000 iteracoes; roda no gate, nao no ciclo curto
--- SKIP: TestRNF11NoCorruptionUnder1000Crashes (0.00s)
=== RUN   TestWriteAtomic_TempInSameDir
--- PASS: TestWriteAtomic_TempInSameDir (0.02s)
=== RUN   TestWriteAtomic_CreatesTempInTargetDir
--- PASS: TestWriteAtomic_CreatesTempInTargetDir (0.01s)
=== RUN   TestWriteAtomic_PreservesBOMAndCRLF
--- PASS: TestWriteAtomic_PreservesBOMAndCRLF (0.04s)
=== RUN   TestWriteAtomic_RenameRetryOnLock
--- PASS: TestWriteAtomic_RenameRetryOnLock (0.06s)
=== RUN   TestPathLocker_SamePathLostUpdate
--- PASS: TestPathLocker_SamePathLostUpdate (0.09s)
=== RUN   TestPathLocker_SamePathCasing
--- PASS: TestPathLocker_SamePathCasing (0.05s)
=== RUN   TestPathLocker_DifferentPathsParallel
--- PASS: TestPathLocker_DifferentPathsParallel (0.00s)
=== RUN   TestPathLocker_NoMemoryLeak
--- PASS: TestPathLocker_NoMemoryLeak (0.11s)
PASS
ok  	github.com/jonyd/gobsidian/internal/writer	1.100s

## Provas de Mutação

### 1. Remoção do `Sync()` (`tmpFile.Sync() -> if false`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/writer/atomic.go -Anchor 'if err := tmpFile.Sync(); err != nil {' -Replacement 'if false {' -Test TestWriteAtomic_TempInSameDir -Package ./internal/writer/`
Saída:
[!] O teste PASSOU com a regra mutada.
    TestWriteAtomic_TempInSameDir nao consegue reprovar sem essa regra: ela esta escrita, nao verificada.
*Nota de Lacuna Técnica:* A gravação com `Sync()` protege contra perda física de energia (kernel flush para o dispositivo NVMe/SATA). A interrupção abrupta de processo via `Process.Kill()` no espaço do usuário testa o isolamento do diretório/rename, mas não simula desalimentação elétrica da controladora de disco. A regra é mantida obrigatoriamente conforme o plano arquitetural.

### 2. Temporário fora do diretório do alvo (`os.CreateTemp(dir) -> os.CreateTemp("")`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/writer/atomic.go -Anchor 'tmpFile, err := os.CreateTemp(dir, TempFilePrefix+"*")' -Replacement 'tmpFile, err := os.CreateTemp("", TempFilePrefix+"*")' -Test TestWriteAtomic_CreatesTempInTargetDir -Package ./internal/writer/`
Saída:
--- FAIL: TestWriteAtomic_CreatesTempInTargetDir (2.01s)
    atomic_test.go:143: temporario nao foi criado no mesmo diretorio do alvo (...), criando risco de rename nao-atomico entre volumes
FAIL
[OK] internal/writer/atomic.go restaurado byte a byte (SHA-256 confere).

### 3. Remoção do Retry no Rename (`maxRetries := 10 -> maxRetries := 1`)
Comando:
`pwsh -File scripts/mutate.ps1 -Path internal/writer/atomic.go -Anchor 'maxRetries := 10' -Replacement 'maxRetries := 1' -Test TestWriteAtomic_RenameRetryOnLock -Package ./internal/writer/`
Saída:
--- FAIL: TestWriteAtomic_RenameRetryOnLock (0.04s)
    atomic_test.go:158: WriteAtomic falhou apos soltar o lock: falha ao renomear ... Access is denied.
FAIL
[OK] internal/writer/atomic.go restaurado byte a byte (SHA-256 confere).

## Verificações Exigidas pelo Brief
| Verificação | Resultado | Evidência |
|---|---|---|
| 1.000 iterações com zero notas corrompidas? | SIM | 0 corrompidas em 1000 iterações (32.21s) |
| Nenhum `.gobsidian-tmp-*` sobrou ao fim das iterações? | SIM | Verificado com `filepath.Glob` |
| Temporário criado no mesmo diretório? | SIM | `TestWriteAtomic_CreatesTempInTargetDir` |
| Retry de rename funciona sob bloqueio temporário? | SIM | `TestWriteAtomic_RenameRetryOnLock` |
| Preservação de BOM e CRLF byte a byte? | SIM | `TestWriteAtomic_PreservesBOMAndCRLF` |

## Arquivos Alterados
- `internal/writer/atomic.go`
- `internal/writer/atomic_test.go`
- `.superpowers/sdd/task-55-report.md`

## O Que Ficou de Fora
Nada.
