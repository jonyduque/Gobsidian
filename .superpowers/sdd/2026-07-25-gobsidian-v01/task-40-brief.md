### Task 40: Higiene — erros engolidos, sentinela comparada por string, e sujeira commitada

Tarefa mecânica. Nenhuma decisão de projeto; tudo abaixo é troca ponto a ponto.

#### Onde isto encaixa

São achados que `golangci-lint` teria pegado e que passaram porque o gate não foi rodado nesta frente. O `go.mod` declara `go 1.25.0` e o CI fixa `golangci-lint v2.12.2`; **confira `golangci-lint version` antes de confiar num zero**, porque um binário compilado com Go mais antigo recusa o config antes de analisar linha nenhuma.

#### O que implementar, item a item

**1. `internal/watcher/watcher.go:52`** — o retorno de `filepath.WalkDir` é descartado. Se a varredura inicial de subdiretórios falhar, o watcher nasce cego numa parte do cofre sem dizer nada. Colete o erro: falha na **raiz** é erro de `New` (a Task 27 é explícita — watcher que não observa o cofre é erro de inicialização, não laço mudo); falha em subdiretório vira `log.Warn` mais incremento do contador de descarte.

**2. `internal/watcher/watcher.go:64` e `:131`** — o erro de `fsWatcher.Add` é descartado nos dois pontos. Mesmo tratamento do item 1.

**3. `internal/watcher/watcher.go:105-108`** — o código faz:

```go
if err == fsnotify.ErrEventOverflow || (err != nil && err.Error() == fsnotify.ErrEventOverflow.Error()) {
```

e o comentário logo acima diz *"Usamos Is"*, o que é falso. Troque por `errors.Is(err, fsnotify.ErrEventOverflow)`, apague a comparação de string, e apague o trecho truncado do comentário (`"macOS reporta overflow the event/err layer dependendo da versao"`, que além de truncado está errado — ver a lacuna de kqueue registrada na Task 34).

**4. `internal/watcher/watcher_test.go:66`** — `err != context.Canceled` vira `!errors.Is(err, context.Canceled)`.

**5. Apague `scratch_fsnotify.go` da raiz do repositório.** Entrou em `0bce29f`. É `package main` no diretório do módulo — além de ser sonda de medição já respondida em `docs/WINDOWS.md:156`, disputa o `main` do módulo. O resultado da medição fica no relatório e no doc; o script não fica.

**6. Apague os comentários de deliberação:**
- `internal/watcher/watcher.go:44` — `// root can be fetched directly or via getter.`
- `internal/watcher/debounce.go:44` — `// It creates a new map instead of modifying the existing one, but doing it in-place is fine.` Além de deliberação, descreve o oposto do que o código faz: `flush` apaga com `delete(dirty, path)`, não cria mapa novo. Se um comentário for útil ali, ele diz **por que** o esvaziamento é in-place, em uma linha.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Não deixe sua deliberação no código.** Comentário explica por que o código é assim; raciocínio sobre o que fazer não é comentário.
- **`golangci-lint` local verde não significa CI verde.** Confira a versão.

#### Verificações além dos passos

- `golangci-lint run ./...` sai limpo? Cole a saída **e** a saída de `golangci-lint version`.
- `grep -rn "scratch" --include=*.go .` devolve vazio?
- `go build ./...` continua limpo depois de remover o `main` da raiz?
- Falha de `fsWatcher.Add` na raiz agora é erro de `New`? Prove com um teste que passa um caminho inobservável, ou diga que não conseguiu construir o cenário e por quê.
- `grep -rn "Wait,\|For the sake of\|we can let it be\|Actually\|TODO" --include=*.go internal/ cmd/` — cole a saída. Deve estar vazia.

#### Regras de execução

- **O plano é a fonte.** Se um teste falhar por motivo que a seção não explica, **pare e reporte**.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** Para remover `scratch_fsnotify.go`, apague o arquivo específico — não use `git clean`.
- **`go mod tidy` está proibido.**
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-40-report.md`: os seis itens com o antes e o depois de cada um; a saída de `golangci-lint version` e de `golangci-lint run ./...`; a saída dos três `grep`; arquivos alterados; e preocupações.

Responda com no máximo 15 linhas, no formato acima.

**Files:**
- Modify: `internal/watcher/watcher.go`, `internal/watcher/watcher_test.go`, `internal/watcher/debounce.go`
- Delete: `scratch_fsnotify.go`

**Commit:** `fix(watcher): stop swallowing Add errors, use errors.Is, drop committed scratch`

---

