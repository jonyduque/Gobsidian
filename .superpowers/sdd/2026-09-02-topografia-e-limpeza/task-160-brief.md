### Task 160: Apagar ou consertar os cinco testes verificados que não podem falhar

**Files:**
- Modify: `internal/watcher/overflow_test.go:219` (`TestReconcile_CtxCancelStopsEarly`)
- Modify: `internal/index/build_test.go:143` (`TestBuildSkipsUnreadableFile` — apagar)
- Delete: `internal/index/normalizacao_equivalente_test.go`
- Modify: `internal/mcpsrv/schema_params_test.go:171,:204,:237`
- Modify: `internal/service/read_test.go:115` (apagar o teste com `t.Skip` incondicional)

**Interfaces:** nenhuma.

Cada sítio foi lido e o mecanismo pelo qual o teste não pode falhar está descrito abaixo. O implementador **confirma** o mecanismo antes de editar (o Step 1 de cada item); se a leitura discordar, `BLOCKED` com o motivo, não edição silenciosa.

- [ ] **Step 1: `overflow_test.go:219` — dar ao cancelamento algo para interromper**

Mecanismo: entre `Build` e `Reconcile` nada muda no cofre; o atalho `overflow.go:58-61` devolve `updated == 0` com ou sem cancelamento, logo `updated >= 200` (a asserção "parou cedo") é inalcançável e o teste passa vazio.

Conserto: criar 300 notas, `Build`, **modificar 300 notas** (reescrever com conteúdo diferente e mtime avançado — `os.Chtimes` +2 s), cancelar o `ctx` **antes** de `Reconcile`, e afirmar `updated < 300`. Com o `ctx` já cancelado, `Reconcile` deve parar no primeiro check; sem o check (mutação), processa as 300.

```go
func TestReconcileCtxCanceladoParaAntesDeProcessarTudo(t *testing.T) {
	root := t.TempDir()
	const n = 300
	for i := range n {
		escreverNota(t, root, fmt.Sprintf("n%03d.md", i), "# v1\n")
	}
	idx := construirIndice(t, root) // o helper que o arquivo ja usa para Build
	depois := time.Now().Add(2 * time.Second)
	for i := range n {
		p := escreverNota(t, root, fmt.Sprintf("n%03d.md", i), "# v2\n")
		if err := os.Chtimes(p, depois, depois); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	st := reconcile(ctx, idx, root) // a chamada que o teste original faz
	if st.Updated >= n {
		t.Fatalf("Reconcile processou %d de %d com ctx cancelado antes de comecar; o cancelamento nao e respeitado", st.Updated, n)
	}
}
```

Adaptar os nomes `escreverNota`/`construirIndice`/`reconcile`/`st.Updated` aos que **existem** no arquivo (ler `overflow_test.go:1-60` e o teste original). Apagar `TestReconcile_CtxCancelStopsEarly`.

Prova: `pwsh -File scripts/mutate.ps1 -Path internal/watcher/overflow.go -Anchor "<a linha do select/if ctx.Err() dentro do loop de Reconcile>" -Replacement "<a mesma linha com a condição sempre falsa>" -Test TestReconcileCtxCanceladoParaAntesDeProcessarTudo -Package ./internal/watcher/` — Expected: exit 0.

- [ ] **Step 2: `build_test.go:143` — apagar**

Mecanismo: o teste cria um **diretório** chamado `unreadable.md` esperando que `Build` o pule por erro de leitura; mas `walk.go:152` devolve antes de `Classify` para qualquer diretório — nunca há leitura. O comentário do teste descreve um cenário que não acontece. A cobertura real de "arquivo ilegível é pulado" é `build_descarte_test.go:16` (e a versão Windows com handle exclusivo).

Apagar `TestBuildSkipsUnreadableFile` inteiro (função e comentário). Se algum helper ficar sem uso, apagar também.

- [ ] **Step 3: `normalizacao_equivalente_test.go` — apagar**

Mecanismo: compara `normalizeString` com `text.Normalize`; `query.go:39-41` é `return text.Normalize(s)`. Tautologia. Se, depois de apagar, `normalizeString` ficar sem chamador de teste, tudo bem — é código de produto e tem seus chamadores.

`git rm internal/index/normalizacao_equivalente_test.go`.

- [ ] **Step 4: `schema_params_test.go:171,:204,:237` — checar o erro e o tamanho**

Mecanismo: `json.Unmarshal(…, &got)` com `_` no erro, depois `for _, e := range got.Edges { … }` — um JSON inválido ou uma lista vazia passa sem afirmar nada.

Em cada um dos três sítios:

```go
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("StructuredContent nao e o JSON esperado: %v\n%s", err, raw)
	}
	if len(got.Edges) == 0 {
		t.Fatal("resposta sem arestas; o teste nao exercitou nada")
	}
```

(`got.Edges` é o campo do sítio `:171`; ler os outros dois e usar o campo iterado em cada um.)

Prova por mutação de **teste**: trocar temporariamente `raw` por `[]byte("{}")` num dos três, rodar, ver FAIL em `len(...) == 0`, restaurar. Colar.

- [ ] **Step 5: `read_test.go:115` — apagar**

Mecanismo: `t.Skip` incondicional na primeira linha; a cobertura pretendida (nota somente-nuvem não é aberta) existe em `internal/service/cloudonly_replace_windows_test.go:33` com handle exclusivo. Apagar a função e o comentário.

- [ ] **Step 6: Gate e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.

```bash
git add internal/watcher/overflow_test.go internal/index/build_test.go internal/index/normalizacao_equivalente_test.go internal/mcpsrv/schema_params_test.go internal/service/read_test.go
git commit -m "test: remove five tests that could not fail, and make the ctx-cancel one able to"
```

#### Verificações

- `go test ./internal/index/ -run 'TestBuildSkipsUnreadableFile|TestNormaliza' -v` imprime `no tests to run`.
- `go test ./internal/service/ -run TestReadNoteCloudOnly -v` não imprime `SKIP` (a função foi apagada, não pulada).
- `mutate.ps1` do Step 1 com exit 0 e a saída colada.
- Step 4 com o FAIL colado.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. `git rm` por caminho explícito é permitido (é o que a tarefa pede).
- Se a leitura de um sítio contradisser o mecanismo descrito, `BLOCKED` naquele item e seguir com os outros — não "consertar" um teste que se entendeu mal.
- Nomes de teste novos em português sem acento, `TestVerboObjeto`, como o resto do repositório.

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/watcher/overflow.go -Anchor "<condição de ctx no loop de Reconcile>" -Replacement "<mesma linha, condição sempre falsa>" -Test TestReconcileCtxCanceladoParaAntesDeProcessarTudo -Package ./internal/watcher/` — o implementador preenche o Anchor lendo `overflow.go`, e cola o comando **como rodou** no relatório. Exit esperado: 0.

#### Contrato de relatório

`task-160-report.md`: status, SHA, para cada um dos cinco itens uma linha "mecanismo confirmado: sim/não — <evidência>", saída do `mutate.ps1`, saída do Step 4, última linha do `verify.ps1`.

