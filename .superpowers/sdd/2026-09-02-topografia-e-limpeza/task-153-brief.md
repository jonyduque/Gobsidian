### Task 153: `doctor` enxerga a trava de escuta, e os sufixos das travas têm uma conta só (1.7 + 1.8)

`daemon/lock.go:118` deriva `<sock>.lock`; `:188` deriva `<sock>.listen.lock`;
`doctor/daemon.go:203` filtra `HasSuffix(nome, ".sock.lock")` — e
`x.sock.listen.lock` NÃO termina em `.sock.lock`. O `doctor` nunca lista a trava
de escuta, que é justamente a que um daemon morto sem fechar o listener deixa
para trás. `doctor/daemon.go:148` ainda faz `sock + ".log"` por conta própria
quando `daemon.CaminhoDoLog` já existe para isso (é o que o comentário de
`daemon/log.go:11-19` diz que a função veio resolver — e sobrou uma cópia).

**Files:**
- Modify: `internal/daemon/lock.go:113-119`, `:181-189`
- Modify: `internal/doctor/daemon.go:141-148`, `:203`
- Test: `internal/daemon/lock_test.go` (novo ou acrescentar ao existente, pacote `daemon`)
- Test: `internal/doctor/daemon_test.go` (acrescentar, pacote `doctor`)

**Interfaces:**
- Produces: `func EhArquivoDeTrava(nome string) bool` em `internal/daemon/lock.go`, exportada — reconhece os DOIS sufixos. Constantes `sufixoTrava = ".lock"` e `sufixoTravaDeEscuta = ".listen.lock"`, não exportadas.

- [ ] **Step 1: Teste em `daemon`**

```go
func TestEhArquivoDeTravaCobreAsDuasTravas(t *testing.T) {
	sock := "abc.sock"
	casos := map[string]bool{
		sock + sufixoTrava:         true,
		sock + sufixoTravaDeEscuta: true,
		sock:                       false,
		sock + ".log":              false,
		"abc.lock.txt":             false,
	}
	for nome, quer := range casos {
		if got := EhArquivoDeTrava(nome); got != quer {
			t.Errorf("EhArquivoDeTrava(%q) = %v, quer %v", nome, got, quer)
		}
	}
	// As constantes sao o que lockPath e ComLockDeEscuta usam de verdade.
	p, err := lockPath(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !EhArquivoDeTrava(filepath.Base(p)) {
		t.Fatalf("lockPath produz %q, que EhArquivoDeTrava nao reconhece", p)
	}
}
```

- [ ] **Step 2: Rodar; `undefined: sufixoTrava`**

- [ ] **Step 3: Constantes e conta em `lock.go`**

Acima de `lockPath`:

```go
// Sufixos das duas travas do daemon, derivadas do caminho do socket. O doctor
// lista as travas pelo mesmo par — ate 2026-09-02 ele filtrava por ".sock.lock"
// e a trava de escuta (".sock.listen.lock") era invisivel para ele.
const (
	sufixoTrava         = ".lock"
	sufixoTravaDeEscuta = ".listen.lock"
)

// EhArquivoDeTrava diz se um nome de arquivo no diretorio de runtime e uma
// das duas travas de daemon. E a unica conta; o doctor a consome.
func EhArquivoDeTrava(nome string) bool {
	return strings.HasSuffix(nome, sufixoTrava) // ".listen.lock" tambem termina em ".lock"
}
```

`lockPath` devolve `sock + sufixoTrava`; `ComLockDeEscuta` usa `sock + sufixoTravaDeEscuta`. Acrescente `"strings"` aos imports se faltar.

- [ ] **Step 4: Teste no `doctor`** (pacote `doctor`, em `daemon_test.go`):

```go
func TestCheckLocksDeDaemonEnxergaListenLock(t *testing.T) {
	cfg := config.Config{VaultPath: t.TempDir()}
	err := daemon.ComLockDeEscuta(cfg.VaultPath, func() error {
		r := checkLocksDeDaemon(context.Background(), cfg)
		if !strings.Contains(r.Detail, "listen.lock") {
			t.Errorf("com a trava de escuta tomada, doctor disse: %q", r.Detail)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCheckDaemonLogUsaACaminhoDoLogDoDaemon(t *testing.T) {
	cfg := config.Config{VaultPath: t.TempDir()}
	esperado, err := daemon.CaminhoDoLog(cfg.VaultPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(esperado), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(esperado, []byte("daemon iniciado\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(esperado) })
	r := checkDaemonLog(context.Background(), cfg)
	if strings.Contains(r.Detail, "ainda nao existe") {
		t.Fatalf("o log existe em %q e o doctor nao o achou: %q", esperado, r.Detail)
	}
}
```

- [ ] **Step 5: `doctor/daemon.go`**

`:141-148`: substitua `sock, err := ipc.SocketPath(...)` + `path := sock + ".log"` por `path, err := daemon.CaminhoDoLog(cfg.VaultPath)` (mesmo tratamento de erro). Se `ipc` ficar sem uso no arquivo, tire o import — confira que `doctor` continua importando `ipc` em outro arquivo (o grafo do `CLAUDE.md` diz `doctor → ipc`; se a aresta sumir de vez, atualize o grafo no mesmo commit).
`:203`: `if e.IsDir() || !daemon.EhArquivoDeTrava(e.Name()) {`.

- [ ] **Step 6: Rodar `go test ./internal/daemon ./internal/doctor -run 'EhArquivoDeTrava|ListenLock|CaminhoDoLog' -v`; passa. Mutação: volte `:203` para `".sock.lock"`, rode, `ListenLock` falha; restaure.**

- [ ] **Step 7: Gate, commit**

```bash
git add internal/daemon/lock.go internal/daemon/lock_test.go internal/doctor/daemon.go internal/doctor/daemon_test.go CLAUDE.md
git commit -m "fix(doctor): list the listen lock too, deriving both lock suffixes and the log path from daemon"
```

(`CLAUDE.md` só entra se o grafo mudou.)

#### Verificações
Além dos passos:
1. Os dois testes do `doctor` usam o diretório de runtime REAL (`ipc.SocketPath` não tem override por env em Windows); o `t.TempDir()` como cofre garante chave única, e o `t.Cleanup` remove o log. Confira que nada ficou em `%LOCALAPPDATA%\gobsidian\run\` com o nome do cofre de teste depois da execução — cole o `ls`.
2. `grep -rn '".lock"\|".listen.lock"\|".log"' internal/daemon internal/doctor` só encontra as constantes em `lock.go` e `log.go`.
3. Se `internal/doctor` deixou de importar `ipc` em todos os arquivos, o grafo em `CLAUDE.md` perde a aresta `doctor → ipc` NO MESMO COMMIT; confira com `go list -f '{{.Imports}}' ./internal/doctor`.

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde antes do commit, com a contagem de etapas colada no relatório. `-SkipCross -SkipNet` só para iterar.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`. Há trabalho não commitado no repositório (`test-vault/`, `.claude/skills/`, `Resume-Claude.ps1`): `git diff <caminho>` antes de `git add <caminho>`; nunca `git add -A`.
- Nunca `go mod tidy`.
- Nunca despache subagentes. Nunca mate processo por nome (`Stop-Process -Name`); só por PID que você lançou.
- Referências e renames por gopls (LSP), não por grep. `grep` só para confirmar tags e reflection.
- Se um teste falhar por motivo que este brief não explica, **pare e reporte `BLOCKED`**; não ajuste a expectativa para o código passar.
- Este commit é de UMA categoria (o prefixo do assunto diz qual). Se você se pegar corrigindo outra coisa no caminho, anote em "O que ficou de fora" e não corrija.

#### Comando de mutação
Copie a âncora **do arquivo**, não de memória — âncora digitada sai `EXIT=2`. `0` = o teste reprovou sob mutação (o que se quer); `1` = a regra está escrita e não verificada; `2` = âncora ambígua ou build quebrado.

```bash
pwsh -File scripts/mutate.ps1 -Path internal/doctor/daemon.go `
  -Anchor '!daemon.EhArquivoDeTrava(e.Name())' `
  -Replacement '!strings.HasSuffix(e.Name(), ".sock.lock")' `
  -Test TestCheckLocksDeDaemonEnxergaListenLock -Package ./internal/doctor/
```
(Se `strings` não estiver importado em `daemon.go`, a mutação sai `EXIT=2` por build; use `-Replacement '!(len(e.Name()) > 10 && e.Name()[len(e.Name())-10:] == ".sock.lock")'`.)

#### Contrato de relatório
Escreva o relatório completo no arquivo de relatório indicado no despacho; devolva no chat só status, SHA, uma linha de testes e as preocupações. O relatório traz:
- **Status** — `DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`.
- **Commit** — SHA curto e assunto (`git log -1 --oneline`).
- **Evidência de TDD** — comando do RED com a saída falhando; comando do GREEN com a saída passando. Não "segui TDD".
- **Prova de mutação** — para cada regra reivindicada: o comando `mutate.ps1` (ou a mutação manual, com o diff), qual teste reprovou **pelo nome e pela linha**, a saída colada, e a confirmação de restauro (`git diff --stat` limpo no arquivo mutado).
- **As verificações do brief** — cada uma com o resultado real, inclusive as que deram certo.
- **`verify.ps1`** — a última linha, com a contagem de etapas.
- **O que ficou de fora** — e por quê. Vazio é resposta aceitável; ausente não é.
- **`git status --porcelain`** — colado. Nenhum arquivo do usuário tocado.

---

