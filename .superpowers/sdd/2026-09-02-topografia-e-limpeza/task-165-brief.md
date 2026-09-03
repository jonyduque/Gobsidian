### Task 165: Buracos de contrapeso — limite 50 aceito, ponte host→daemon, trava do kernel, `AliasCollisions`

**Files:**
- Modify: `internal/mcpsrv/tools_read_test.go:324` (acrescentar o par: 50 aceito)
- Modify: `cmd/gobsidian/ponte_test.go:195` (host→daemon exercitado; half-close M8)
- Create: `internal/daemon/trava_kernel_windows_test.go`, `internal/daemon/trava_kernel_other_test.go`
- Modify: `internal/daemon/*_test.go` (cobrir `EscutarComLock`, `TravaEmUso`)
- Modify: `internal/index/*_test.go` (cobrir `AliasCollisions`)
- Modify: `internal/ipc/*_test.go` (handshake com `ReadOnly=true`)

**Interfaces:**
- Consumes: `vaulttest.Prazo` (Task 159); `daemon.EhArquivoDeTrava` (Task 153) não é usado aqui; o offset `1<<62` de `internal/daemon/trava_windows.go:19-21`, lido por `internal/doctor/daemon.go:219`.

Cada item é um teste que **afirma o lado aceito** de uma regra que hoje só tem o lado recusado, ou que exercita um caminho que nenhum teste percorre. O critério de pronto de cada um é a mutação nomeada.

- [ ] **Step 1: `tools_read_test.go:324` — 50 aceito**

Hoje: 51 caminhos recusados. Mutação `>` → `>=` no produto sobrevive. Acrescentar o subteste "50 aceito" ao lado:

```go
	t.Run("cinquenta caminhos sao aceitos", func(t *testing.T) {
		paths := make([]string, 50)
		for i := range paths {
			paths[i] = fmt.Sprintf("n%02d.md", i)
		}
		res := chamar(t, sess, "note_metadata", map[string]any{"paths": paths})
		if res.IsError {
			t.Fatalf("50 caminhos e o limite, nao acima dele: %s", textoDoErro(res))
		}
	})
```

(Adaptar `chamar`/`textoDoErro`/o nome da tool ao que `:324` usa.)

Prova: `mutate.ps1 -Path <arquivo do limite> -Anchor "len(req.Paths) > 50" -Replacement "len(req.Paths) >= 50" -Test <nome do teste> -Package ./internal/mcpsrv/` — exit 0.

- [ ] **Step 2: `ponte_test.go:195` — host→daemon**

Hoje: `stdinHost, _ := io.Pipe()` descarta o writer, logo nada flui do host para o daemon; apagar `io.Copy(conn, teed)` (`ponte.go:177-180`) e o half-close M8 (`:198-215`) passa.

Conserto: guardar o writer, escrever um `initialize` JSON-RPC válido nele depois de a ponte estar de pé, e afirmar que o daemon **recebeu** (ler do lado do daemon — o fake que o teste monta — até ver o `"method":"initialize"`, com `vaulttest.Prazo`). Depois fechar o writer e afirmar que o daemon vê EOF dentro do prazo (o half-close M8).

Prova: duas mutações manuais — comentar `io.Copy(conn, teed)`; comentar o half-close — cada uma deve fazer o teste falhar. Colar as duas.

- [ ] **Step 3: `daemon` — trava do kernel entre processos**

Hoje `ComLockDeEscuta` só é testado in-process; trocar `LockFileEx` por `sync.Mutex` passaria. E o offset `1<<62` (`trava_windows.go:19-21`), que `doctor/daemon.go:219` **lê**, não tem teste que o prenda.

`internal/daemon/trava_kernel_windows_test.go` (`package daemon_test`, `//go:build windows`):

1. Criar o arquivo de trava, chamar `daemon.EscutarComLock` (ou a função pública que adquire — ler `trava.go`) no processo de teste.
2. Lançar `os.Executable()` com `-test.run=TestAjudanteTravaEmUso` e env `GOBSIDIAN_TRAVA_AJUDANTE=<caminho>` (padrão de processo-ajudante do `os/exec` da stdlib; o ajudante retorna imediatamente se a env não está setada).
3. O ajudante tenta `daemon.TravaEmUso(caminho)` e sai com código 3 se a trava está em uso, 4 se não.
4. O teste afirma exit 3. Depois libera a trava e relança: exit 4.
5. Offset: o ajudante também tenta `LockFileEx` no offset `1<<62` diretamente (`windows.LockFileEx` com `OverlappedOffset` = os 64 bits do offset) e sai com 5 se **conseguiu** — o que prova que o produto **não** usa esse offset; esperado: falha (o produto está segurando). Se o produto mudar o offset, este teste é o que avisa o `doctor`.

`trava_kernel_other_test.go`: mesmo desenho com `flock` (`unix.Flock`), sem o offset.

Prova: `mutate.ps1 -Path internal/daemon/trava_windows.go -Anchor "1 << 62" -Replacement "1 << 61" -Test TestTravaDoKernelEntreProcessos -Package ./internal/daemon/` — exit 0.

- [ ] **Step 4: `AliasCollisions`**

O campo que substituiu o `0` literal da armadilha está a 0 % de cobertura. Teste em `internal/index/`: duas notas com o mesmo alias no frontmatter; `Build`; afirmar `stats.AliasCollisions == 1` (ou o valor que a semântica de `chave.go` definir — ler e dizer). Mutação: trocar o incremento por nada — exit 0.

- [ ] **Step 5: `ipc` handshake com `ReadOnly=true`**

Toda ponte de teste passa `ReadOnly=false`. Acrescentar um teste que faz o handshake com `ReadOnly=true` e afirma que o servidor do outro lado o recebe assim (ler `internal/ipc/handshake.go` para ver como o campo viaja e onde é observável). Mutação: forçar `ReadOnly=false` no encode — exit 0.

- [ ] **Step 6: Gate, órfãos e commit**

Run: `pwsh -File scripts/verify.ps1` — Expected: verde.
Run: `pwsh -File scripts/test_orphans.ps1` — Expected: os quatro cenários `[OK]` (a ponte foi tocada).

```bash
git add internal/mcpsrv/tools_read_test.go cmd/gobsidian/ponte_test.go internal/daemon/trava_kernel_windows_test.go internal/daemon/trava_kernel_other_test.go internal/daemon/*_test.go internal/index/*_test.go internal/ipc/*_test.go
git commit -m "test: counterweights for the accepted limit, the host-to-daemon bridge, the kernel lock offset, alias collisions and read-only handshake"
```

(Antes do `git add` com glob: `git status --short internal/daemon/ internal/index/ internal/ipc/` e conferir que **só** arquivos desta tarefa aparecem; se houver outro, adicionar por nome.)

#### Verificações

- Cinco saídas de `mutate.ps1` (Steps 1, 3, 4, 5 e — para o Step 2 — as duas mutações manuais) coladas, todas exit 0 / FAIL.
- `go test -cover ./internal/daemon/ ./internal/index/ ./internal/ipc/` antes e depois, colado: `EscutarComLock`, `TravaEmUso`, `AliasCollisions` deixam de estar a 0 % (usar `go tool cover -func` e colar as três linhas).
- `test_orphans.ps1` com os quatro `[OK]`.
- `verify.ps1` verde.

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`.
- Processo-ajudante: matar só o PID que o teste lançou, nunca por nome; `t.Cleanup` com `cmd.Process.Kill()` se ainda vivo.
- Nenhum `net.Listen` fora de `ipc`; o teste do daemon usa `ipc.Listen`.
- Nenhuma alteração de produto. Se um contrapeso revelar defeito, relatório "defeito encontrado — fora de escopo".
- Não rodar `test_orphans.ps1` em paralelo com outra coisa.

#### Comando de mutação

`pwsh -File scripts/mutate.ps1 -Path internal/daemon/trava_windows.go -Anchor "1 << 62" -Replacement "1 << 61" -Test TestTravaDoKernelEntreProcessos -Package ./internal/daemon/` — exit 0. Mais os de Steps 1, 4 e 5, colados como rodaram.

#### Contrato de relatório

`task-165-report.md`: status, SHA, as cinco provas, as três linhas de `cover -func`, saída de `test_orphans.ps1`, última linha do `verify.ps1`.

---

