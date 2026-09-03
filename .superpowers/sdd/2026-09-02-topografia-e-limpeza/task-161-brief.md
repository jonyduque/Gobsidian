### Task 161: Auditoria dos treze testes reportados pelo sweep — cada um vira conserto, apagamento ou "falsifica, mantido"

**Files:**
- Modify (conforme veredito): `internal/mcpsrv/tools_read_test.go:96-133`, `internal/service/limites_enums_test.go:44`, `internal/mcpsrv/server_test.go:116`, `internal/search/bm25_test.go:213`, `internal/search/persist_test.go:232`, `internal/search/pool_test.go:9`, `internal/service/write_test.go:242,:368,:384`, `internal/index/slug_persistido_test.go:84`, `cmd/gobsidian/cli_subcommands_test.go:168`, `cmd/gobsidian/console_saida_test.go:100`, `internal/watcher/filter_test.go:117`, `internal/search/inverted_test.go:118`

**Interfaces:** nenhuma.

Estes treze foram apontados por uma varredura e são coerentes com o código, mas **não foram re-lidos um a um** (o relatório de 2026-09-02 diz isso explicitamente). Esta tarefa é a re-leitura. O produto é uma tabela no relatório com **um veredito por sítio**, e o diff que os vereditos exigem.

- [ ] **Step 1: Para cada sítio, o mesmo protocolo**

1. Ler o teste inteiro e a função de produto que ele exercita.
2. Escrever, em uma frase, **qual regra de produto** o teste afirma.
3. Apagar essa regra no produto (edição temporária: inverter a condição, devolver zero, devolver `nil`), rodar **só aquele teste**, anotar PASS ou FAIL. Restaurar.
4. Veredito:
   - **FAIL sob mutação** → "falsifica, mantido". Sem diff.
   - **PASS sob mutação, e a regra tem outro teste que falsifica** → apagar o teste (dizer qual é o outro).
   - **PASS sob mutação, e a regra não tem outro teste** → consertar: acrescentar a asserção que falta, repetir o passo 3 até FAIL.

Pistas por sítio (a leitura decide, não a pista):

| Sítio | Pista |
|---|---|
| `tools_read_test.go:96-133` | 6 de 7 subtestes só checam `!IsError && StructuredContent != nil`; zero hits passa. Consertar: afirmar sobre o conteúdo (`len(results) > 0`, o path esperado). |
| `limites_enums_test.go:44` | Resultado descartado; o clamp de `MaxResults` é inobservável. Consertar: afirmar `len(res.Results) <= limite`. |
| `server_test.go:116` | Ver o que afirma. |
| `bm25_test.go:213` | Testa filtro de NaN, mas `bm25.go:193` já filtra antes. Provável "apagar" se outro teste cobre `:193`, senão mover a asserção para onde o NaN pode entrar. |
| `persist_test.go:232` | Não chama código de produção. Provável apagar. |
| `pool_test.go:9` | Idem. |
| `write_test.go:242,:368,:384` | Hash esperado calculado a partir do mesmo índice que o produto usa — tautologia. Consertar: hash esperado calculado **do arquivo** (`sha256` do conteúdo lido do disco), não do índice. |
| `slug_persistido_test.go:84` | Ver o que afirma. |
| `cli_subcommands_test.go:168` | Ver o que afirma. |
| `console_saida_test.go:100` | Aborta antes de chegar em `serve`; o que quer provar (stdout limpo?) não é exercitado. Consertar ou apagar. |
| `filter_test.go:117` | `root` hardcoded `C:\` — o próprio arquivo registra 4 dias de CI vermelho por isso. Consertar: `t.TempDir()`. |
| `inverted_test.go:118` | `DocCount() < 0` — `int` sem sinal de negativo possível? Ver o tipo; se `int`, a asserção é vazia. Consertar com o valor esperado exato. |

- [ ] **Step 2: Rodar o pacote de cada arquivo tocado**

Run: `go test -race ./internal/mcpsrv/ ./internal/service/ ./internal/search/ ./internal/index/ ./internal/watcher/ ./cmd/... 2>&1 | tail -10` — Expected: `ok`.

- [ ] **Step 3: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add <cada arquivo tocado, por caminho>
git commit -m "test: audit thirteen sweep-flagged tests; fix the ones that could not fail, drop the redundant ones"
```

#### Verificações

- A tabela do relatório tem **treze linhas** com: sítio, regra afirmada (uma frase), mutação aplicada (uma linha), resultado (PASS/FAIL), veredito, ação.
- Para cada "consertado": saída do teste **falhando** sob a mutação, colada.
- Para cada "apagado": nome do outro teste que falsifica a mesma regra.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. Mutação temporária é edição manual seguida de edição manual de volta; conferir com `git diff` que o produto ficou intacto **antes** do commit (o diff só pode tocar `_test.go`).
- Um veredito por sítio. "Provavelmente" não é veredito.
- Não consertar produto nesta tarefa. Se a mutação revelar um defeito real de produto, registrar no relatório como "defeito encontrado — fora de escopo" e seguir.

#### Comando de mutação

Cada linha da tabela é uma mutação manual; a tarefa não usa `mutate.ps1` porque as mutações são de asserção de teste, não de uma regra única de produto. As saídas coladas são a prova.

#### Contrato de relatório

`task-161-report.md`: status, SHA, a tabela de treze linhas, saídas coladas, `git diff --stat` do commit confirmando só `_test.go`.

