### Task 36: `index.MoveNote` — pagar as dívidas de uma API que entrou fora do contrato

Decisão fechada: `MoveNote` fica. Esta tarefa é o que a torna aceitável.

#### Onde isto encaixa

`MoveNote` foi acrescentada em `d5d1bf0`, fora do contrato declarado da Task 31, e o `Apply` aplica o rename correlacionado **só** por ela — sem `Remove` do caminho antigo nem `Replace` do novo (`internal/watcher/apply.go:34-44`). Ela não é otimização sobre um caminho correto: ela **é** o caminho. Manter isso exige a revisão que a entrega pulou.

Depende da Task 35 (a chave de `byAlias` precisa estar normalizada antes) e da Task 33 (a correlação precisa estar correta antes de testar o que ela alimenta).

#### A evidência medida do defeito

```
mutei internal/index/update.go: MoveNote vira no-op
--- FAIL: TestMoveNote (internal/index)
ok  github.com/jonyd/gobsidian/internal/watcher  5.714s
```

Nenhum teste do pacote `watcher` percebe que um rename correlacionado deixa de ser aplicado ao índice. `TestMoveNote` monta o índice na mão — é a forma de teste que o próprio plano da Task 31 avisou que passa sem correlacionador.

Além disso, `MoveNote` preserva `Size` e `ModTime` do caminho antigo. Se o sistema de arquivos alterar o mtime no move, o curto-circuito do `Apply` passa a comparar contra um valor nunca medido no caminho novo, e a nota deixa de ser reindexada.

#### O que implementar

**1. Atualizar `Size` e `ModTime`.** Depois de `n.Path = newPath` em `internal/index/update.go`, faça `os.Stat` do caminho novo e atualize os dois campos. Se o `Stat` falhar, zere `ModTime` para forçar reindexação na próxima passagem e logue em `Debug`. Zerar é deliberado: `ModTime` zero nunca bate com um `os.Stat` real, então o curto-circuito não dispara.

**2. Provar equivalência estrutura por estrutura.** `MoveNote` mexe à mão em oito coisas: `notes`, `lowerPath`, `byName`, `tags`, `byAlias`, backlinks de entrada, backlinks de saída e o reprocessamento final. Ler as 90 linhas não acha a estrutura esquecida; comparar acha.

`TestMoveNote_EquivalentToRemoveReplace` em `internal/index`: dois cofres idênticos, dois índices; num deles `MoveNote(a, b)`, no outro `Remove(a)` seguido de `Replace(ctx, v, b)`. Afirmar igualdade **estrutura por estrutura**, nomeando cada uma na mensagem de falha. Onde a divergência for deliberada, afirme a divergência explicitamente em vez de ignorá-la:
- `Hash` preservado sem releitura em `MoveNote` — deve ser igual de qualquer forma, já que o conteúdo é o mesmo.
- `generation` incrementada uma vez em `MoveNote` contra duas em `Remove`+`Replace` — divergência esperada, afirme o valor.

**3. Alias pelo caminho do `MoveNote`.** Com a Task 35 fechada, rode a mesma verificação de alias por este caminho: nota com alias `STJ`, renomeada por `MoveNote`, e o alias tem que resolver para o **caminho novo**.

**4. O teste de integração que falta.** `TestWatcher_RenameEndToEnd` em `internal/watcher`: watcher rodando, nota com backlinks renomeada **no disco** (`os.Rename`, não chamada de função), esperar uma janela, afirmar que o índice tem o caminho novo, não tem o velho, e que os backlinks seguiram para o caminho novo.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Regra que sobrevive a mutação não está verificada, está escrita.** Sete regras de um módulo sobreviveram a mutantes com a suíte verde. Aqui são oito estruturas; trate cada uma como uma regra.
- **Não deixe sua deliberação no código.** Se uma estrutura não precisar ser atualizada, o comentário diz por quê em uma linha, não conta o raciocínio.

#### Verificações além dos passos

- Os quatro itens acima estão implementados e testados? Liste cada um com o nome do teste.
- **Prova de mutação 1:** `MoveNote` vira no-op → `TestWatcher_RenameEndToEnd` **precisa** reprovar. Se só `TestMoveNote` reprovar, o teste de integração não está exercendo o caminho.
- **Prova de mutação 2:** para **cada uma** das oito estruturas, remova a atualização correspondente dentro de `MoveNote`, rode, e diga qual teste reprovou nomeando a estrutura. **Este é o item central da tarefa.**

  **Instrução expressa, porque a saída fácil aqui é registrar e seguir:** quando uma estrutura sobreviver à mutação — `scripts/mutate.ps1` saindo `1` — **o entregável é o teste que a pega, escrito nesta tarefa.** Não é uma linha no relatório dizendo "estrutura X sem cobertura", não é um `TODO`, não é um item para a próxima tarefa. Uma estrutura que sobrevive à mutação é uma estrutura que `MoveNote` pode parar de atualizar sem ninguém notar, e o sintoma disso em produção é uma consulta devolvendo caminho que não existe — silêncio, não erro.

  O laço é: mute a estrutura → saiu `1` → escreva o teste que afirma aquela estrutura especificamente depois do move → rode a mutação de novo → tem que sair `0` → restaure → próxima estrutura. Oito voltas, oito saídas coladas no relatório, cada uma com o nome do teste que reprovou.

  Se uma estrutura for genuinamente não observável de fora do pacote, escreva o teste **dentro** do pacote `index` (`package index`, não `index_test`) acessando o campo direto. Preferir a fronteira pública é bom estilo; deixar a regra sem verificação para preservar o estilo não é.

  Esperar oito testes novos é o resultado normal, não sinal de que você entendeu errado. Hoje `MoveNote` inteira reduzida a no-op deixa **todos** os testes do pacote `watcher` verdes, e só `TestMoveNote` reprova — um teste para oito estruturas.

```bash
# o laco, por estrutura. Exemplo com byName:
pwsh -File scripts/mutate.ps1 -Path internal/index/update.go `
  -Anchor 'ix.byName[string(newBase)] = append(ix.byName[string(newBase)], newPath)' `
  -Replacement '' `
  -Test TestMoveNote_UpdatesByName -Package ./internal/index/
```
- `os.Stat` do caminho novo falhando deixa `ModTime` zerado e força reindexação na próxima passagem? Prove.

#### Regras de execução

- **O plano é a fonte.** Se um teste falhar por motivo que a seção não explica, **pare e reporte**.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.**
- **`go mod tidy` está proibido.**
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-36-report.md`: o que implementou; evidência de TDD (RED e GREEN); a prova de mutação de integração; **a tabela das oito estruturas, uma linha por estrutura, dizendo o que foi mutado e qual teste reprovou por nome**; arquivos alterados; achados da auto-revisão; e preocupações.

Responda com no máximo 15 linhas, no formato acima.

**Files:**
- Modify: `internal/index/update.go` (`MoveNote`)
- Modify: `internal/index/resolve_test.go` ou novo arquivo de teste de `MoveNote`
- Modify: `internal/watcher/rename_test.go` ou novo arquivo para o teste de integração

**Commit:** `fix(index): MoveNote refreshes stat and matches Remove+Replace`

---

