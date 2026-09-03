### Task 166: Código morto — apagar o que ninguém chama, e o doc que promete o que ninguém emite

**Files:**
- Modify: `internal/console/console.go:91` (`Stream.Writer`)
- Modify: `internal/service/service.go:102` (`Inverted()`)
- Modify: `internal/vault/eol.go:71,:79` (`AddBOM`, `NormalizeEOL` — **não** confundir com `writer/section.go:54 NormalizeEOL`, que é viva)
- Modify: `internal/writer/atomic.go:34` (`CleanStaleTempFiles`) e o comentário em `cmd/gobsidian/servico.go:103` que a cita
- Modify: `internal/mcpsrv/server.go:29,:42` (`cfg` escrito e nunca lido)
- Modify: `internal/service/errors.go:36` (`CodePathTooLong`) + `docs/TOOLS.md:503` (5.6)
- Modify: `internal/index/persist_codec.go:639,:682` (guardas `l.uvarint(math.MaxUint64, …)` que nunca disparam)
- Modify: `internal/search/inverted.go:25,:275,:298` (`Posting.Frequency` — só leitores em teste; **não** está no formato de cache)
- Modify: `internal/doctor/doctor.go:61-67` (`halting` constante)
- Modify: `internal/lifecycle/lifecycle.go:28-69` (`Options.ParentCheckInterval` nunca definido em produção — ver Step 2)
- Modify: `internal/doctor/checks.go:277-278` (contadores escritos em toda plataforma, lidos só em `checks_windows.go` — ver Step 2)
- Modify: `internal/service/write.go:423-426` (`vault.Resolve` com três resultados descartados → `if req.To == ""`)
- Modify: os `_test.go` que leem `Posting.Frequency` (encontrar com `gopls references`)

**Interfaces:** nenhuma nova. `writer.CleanStaleTempFiles` some antes da Task 171 mover o resto do arquivo.

- [ ] **Step 1: Para cada símbolo, `gopls references` antes de apagar**

Regra: apagar só o que tem **zero** referências fora da própria declaração, contando `_test.go`. Se um teste referencia, o teste é apagado junto **só se** ele testa exclusivamente o símbolo morto (ler o teste). Registrar no relatório a contagem de referências por símbolo (`gopls references` ou `grep -rn`).

Para `Posting.Frequency`: confirmar com `grep -n Frequency internal/search/persist_codec.go internal/search/soa.go` vazio (não está no formato — apagar o campo não muda o cache). Apagar o campo, os `:275,:298` que o escrevem, e as leituras em teste.

- [ ] **Step 2: Dois casos que não são "apagar"**

`lifecycle.Options.ParentCheckInterval`: nunca definido em produção → o padrão é a única conta. **Não apagar o campo** se um teste o usa para encurtar a vigília (`grep -rn ParentCheckInterval --include=*_test.go`); se nenhum teste o usa, apagar o campo e deixar a constante. Dizer qual foi no relatório.

`doctor/checks.go:277-278`: contadores escritos em toda plataforma e lidos só em `checks_windows.go`. Mover a escrita para o arquivo `_windows.go` que os lê (código de plataforma atrás de build tag). Se isso exigir um ponto de extensão, `BLOCKED` com a proposta — não `if runtime.GOOS`.

- [ ] **Step 3: `write.go:423-426`**

Hoje: `_, _, err := vault.Resolve(root, req.To)` (ou equivalente) só para rejeitar `to: ""`. Trocar por:

```go
	if req.To == "" {
		return MoveNoteResult{}, Errorf(CodeInvalidArgument, "to e obrigatorio")
	}
```

e conferir que a resolução real de `req.To` acontece mais adiante (ela acontece — é o que torna a chamada de `:423` redundante; se **não** acontecer, a chamada não era morta: `BLOCKED`).

Prova: `mutate.ps1 -Path internal/service/write.go -Anchor 'if req.To == ""' -Replacement 'if false && req.To == ""' -Test <teste que já cobre to vazio; se não existir, criar TestMoveNoteSemDestinoEInvalidArgument> -Package ./internal/service/` — exit 0.

- [ ] **Step 4: `TOOLS.md:503` — `PATH_TOO_LONG`**

Retirar a linha da tabela de erros. Se `check_doc_refs.ps1` reclamar de referência solta, é sinal de que outro doc cita — retirar lá também.

- [ ] **Step 5: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde (o `golangci-lint` com `unused` é quem confirma que nada ficou órfão).

```bash
git add internal/console/console.go internal/service/service.go internal/vault/eol.go internal/writer/atomic.go cmd/gobsidian/servico.go internal/mcpsrv/server.go internal/service/errors.go docs/TOOLS.md internal/index/persist_codec.go internal/search/inverted.go internal/doctor/doctor.go internal/lifecycle/lifecycle.go internal/doctor/checks.go internal/doctor/checks_windows.go internal/service/write.go <testes tocados>
git commit -m "refactor: delete dead code and the error code no path produces"
```

#### Verificações

- Tabela no relatório: símbolo, referências antes (número), ação (apagado / mantido porque …).
- `go build ./... && go vet ./...` nos três GOOS (o `verify.ps1` faz) — `checks_windows.go` compila.
- `mutate.ps1` do Step 3, exit 0, colado.
- `git diff --stat`: só remoção líquida fora de `write.go` e `checks_windows.go`.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Nunca `if runtime.GOOS ==`.
- Um símbolo com referência viva **não** é apagado nesta tarefa — vai para o relatório como "não morto: <quem chama>".
- Sem alteração de comportamento observável além de `to: ""` responder `INVALID_ARGUMENT` (que já respondia, por outro caminho).

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/service/write.go -Anchor 'if req.To == ""' -Replacement 'if false && req.To == ""' -Test TestMoveNoteSemDestinoEInvalidArgument -Package ./internal/service/` — exit 0.

#### Contrato de relatório

`task-166-report.md`: status, SHA, a tabela de símbolos, a saída do `mutate.ps1`, `git diff --stat`, última linha do `verify.ps1`.

