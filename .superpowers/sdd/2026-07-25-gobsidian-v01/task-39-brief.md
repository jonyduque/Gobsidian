### Task 39: Fechar as quatro verificações da Task 27 que foram afirmadas sem teste

#### Onde isto encaixa

`.superpowers/sdd/task-27-report.md` traz uma tabela de verificações com quatro linhas marcadas "Sim". Nenhuma delas tem teste. `internal/watcher/watcher_test.go` tem 76 linhas e cobre apenas o caminho feliz: cria uma nota, confirma que entra no índice, cancela, fecha.

#### As quatro afirmações sem cobertura

Do relatório da Task 27, textualmente:

| Afirmação do relatório | Cobertura real hoje |
|---|---|
| "Fechar o watcher libera handles? Sim. O OS renomeia e apaga normal depois." | Nenhum `os.Rename` nem `os.RemoveAll` da raiz depois do `Close` em teste algum |
| "Canal de eventos fechado no shutdown? Sim. Validado em teste." | Nenhum teste lê o canal depois do shutdown |
| "Evento fora da raiz (links)? Sim." | Nenhum teste injeta evento fora da raiz |
| "Diretório criado depois gera evento? Sim, interceptado no loop." | `watcher.go:126-134` não tem teste algum |

O último é o mais grave: esse bloco é o compensador da ausência de recursividade do `fsnotify` no Windows, é o código mais frágil da Task 27, e uma falha nele deixa uma subárvore inteira do cofre sem observação — sem erro, sem sintoma.

#### O que implementar

Em `internal/watcher/watcher_test.go` e `filter_test.go`:

1. `TestWatcher_CloseReleasesHandles` — depois de `cancel()` e `Close()`, `os.Rename` da raiz do cofre para um irmão e depois `os.RemoveAll`. Se sobrar handle de diretório, no Windows isso falha. Pule com `runtime.GOOS != "windows"` — é o uso aceitável de `runtime.GOOS` em teste.
2. `TestWatcher_EventsChannelClosedOnShutdown` — afirmar que o receive do canal de eventos devolve `ok == false` depois do shutdown, com timeout para não travar a suíte.
3. `TestWatcher_DirCreatedAfterStartIsWatched` — com o watcher rodando, `os.Mkdir` de um subdiretório novo, esperar, escrever uma nota dentro dele, afirmar que ela entra no índice.
4. `TestFilter_OutsideVaultIsDropped` — evento com `Name` apontando para fora da raiz; afirmar que não é emitido e que o contador de `outside_vault` subiu.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Um teste que não pode falhar é pior que teste ausente.** Se o teste espera com `time.Sleep` generoso, ele passa sem o mecanismo. Espere em laço com condição de saída e afirme a condição.
- **Confinamento de caminho tem duas camadas**, e a regra de ponto/espaço no fim de componente vale **só no Windows**. Não escreva fixture que dependa disso fora do Windows.

#### Verificações além dos passos

- Os quatro testes existem e passam? Liste com o resultado.
- **Prova de mutação obrigatória:** remova o bloco `internal/watcher/watcher.go:126-134` (o watch dinâmico de subdiretório), confirme que `TestWatcher_DirCreatedAfterStartIsWatched` reprova, restaure. Cole a saída.
- **Segunda prova de mutação:** remova o `defer close(w.events)` de `Run`, confirme que `TestWatcher_EventsChannelClosedOnShutdown` reprova (ou trava no timeout, o que também é reprovação legítima — diga qual aconteceu), restaure.
- `TestWatcher_CloseReleasesHandles` roda de verdade nesta máquina, ou foi pulado? Diga qual, e em qual `GOOS`.

#### Regras de execução

- **O plano é a fonte.** Se um teste falhar por motivo que a seção não explica, **pare e reporte** — o teste pode estar certo e o código errado, e nesse caso a correção é do código.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.**
- **`go mod tidy` está proibido.**
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-39-report.md`: o que implementou; evidência de TDD (RED e GREEN); as **duas** provas de mutação com saída colada; a tabela das quatro verificações com o resultado real; se algum teste foi pulado e por quê; arquivos alterados; achados da auto-revisão; e preocupações.

Responda com no máximo 15 linhas, no formato acima.

**Files:**
- Modify: `internal/watcher/watcher_test.go`, `internal/watcher/filter_test.go`

**Commit:** `test(watcher): cover handle release, channel close, and dynamic subdirectory watch`

---

