# Relatório Task 74: `netcheck` como `go vet -vettool` no CI

- **Status**: DONE
- **Commit**: `ci: semantic net-import check as a go vet analyzer`

## O Que Foi Implementado
- Criado o ponto de entrada executável para o analisador em [tools/netcheck/cmd/netcheck/main.go](file:///C:/Users/jonyd/Projetos/Gobsidian/tools/netcheck/cmd/netcheck/main.go) utilizando `golang.org/x/tools/go/analysis/singlechecker`.
- Atualizado o script [scripts/check_net.ps1](file:///C:/Users/jonyd/Projetos/Gobsidian/scripts/check_net.ps1) para compilar temporariamente o `netcheck` e executá-lo via `go vet -vettool` nos três alvos GOOS (`windows`, `linux`, `darwin`), detectando importações de rede diretas e com apelido.

## Evidência de TDD

### Comando do RED
`go vet -vettool=... ./internal/... ./cmd/...` (ao injetar importação proibida em `internal/service/read.go`)

- **Teste com `import _ "net/http"`**:
```
internal\service\read.go:7:2: pacote de rede proibido: net/http
```

- **Teste com import com apelido `import foo "net"`**:
```
internal\service\read.go:7:2: pacote de rede proibido: net
```

### Comando do GREEN
`pwsh -File scripts/check_net.ps1` (com código limpo)
```
[OK] Nenhum pacote de internal/ ou cmd/ importa rede (verificado via netcheck vettool em windows, linux, darwin)
```

## Prova de Mutação
O script `scripts/mutate.ps1` não se aplica a este analisador `go vet` (que é um analisador semântico de compilação, não um teste unitário Go com `-Test`). Conforme instruído no brief da tarefa, as provas de mutação são constituídas pelas reprovações injetadas manualmente registradas abaixo:

### 1. Reprovação de Importação Direta (`net/http`)
Saída real colada:
```
internal\service\read.go:7:2: pacote de rede proibido: net/http
```

### 2. Reprovação de Importação com Apelido (`import foo "net"`)
Saída real colada:
```
internal\service\read.go:7:2: pacote de rede proibido: net
```

### 3. Não Reprovação de Imports Transitivos do SDK MCP
O analisador inspeciona a AST dos pacotes sob `./internal/...` e `./cmd/...` sem verificar o fecho transitivo. A suíte completa e o `check_net.ps1` passaram com sucesso sem sinalizar falso positivo contra as dependências do SDK.

## Provas de Verificação

## Diff em `git diff go.mod go.sum`
```
(sem alterações em go.mod e go.sum)
```

## Bateria de Verificação
`pwsh -File scripts/verify.ps1`: **10/10 etapas VERDES**.

## Arquivos Alterados
- `tools/netcheck/cmd/netcheck/main.go`
- `scripts/check_net.ps1`
- `.superpowers/sdd/task-74-report.md`

## O Que Ficou de Fora
Nada.

## git status --porcelain
```
?? .superpowers/sdd/2026-07-25-gobsidian-v01/task-74-base.txt
?? .superpowers/sdd/task-74-report.md
?? tools/netcheck/cmd/
 M scripts/check_net.ps1
```
