### Task 147: Falha de `ReadDir` na raiz varrida é falha da raiz (1.1)

`filepath.WalkDir(dir, fn)` chama `fn` para a raiz DUAS vezes quando `Lstat`
passa mas `ReadDir` falha: a primeira com `d != nil, err == nil`, a segunda com
`d != nil, caminho == dir, err != nil`. Os dois callbacks que tratam "falha da
raiz" só reconhecem `d == nil` (`Lstat` falhou), então uma raiz que EXISTE mas
não pode ser LIDA vira "entrada ilegível", é engolida, e a varredura devolve
sucesso com zero entradas — no `vault.Walk` isso é "cofre vazio"; no
`watcher.varreDiretorioNovo` é "diretório novo sem nada dentro".

Provado nesta máquina em 2026-09-02 com um handle exclusivo
(`CreateFile(..., dwShareMode=0, FILE_FLAG_BACKUP_SEMANTICS)`) sobre o
diretório: `os.Lstat(dir)` → `nil`; `os.ReadDir(dir)` → `The process cannot
access the file because it is being used by another process`. É exatamente o
cenário do antivírus segurando a pasta recém-movida.

**Files:**
- Modify: `internal/vault/walk.go:123-146` (callback de `Walk`)
- Modify: `internal/watcher/watcher.go:228-252` (callback de `varreDiretorioNovo`)
- Test: `internal/vault/walk_raiz_test.go` (novo, puro)
- Test: `internal/vault/walk_raiz_windows_test.go` (novo, `//go:build windows`)
- Test: `internal/watcher/varredura_raiz_test.go` (novo, `//go:build !windows`)

**Interfaces:**
- Produces: `func FalhaNaRaiz(raiz, caminho string, d fs.DirEntry) bool` em `internal/vault/walk.go`, exportada — a ÚNICA conta de "este erro do WalkDir é da raiz". `watcher` já importa `vault`; nenhuma aresta nova.

- [ ] **Step 1: Teste puro da conta, em `internal/vault/walk_raiz_test.go`**

```go
package vault

import (
	"io/fs"
	"testing"
)

// entradaFalsa satisfaz fs.DirEntry sem tocar o disco. So Name e usado
// pela conta; o resto existe para compilar.
type entradaFalsa struct{ nome string }

func (e entradaFalsa) Name() string               { return e.nome }
func (e entradaFalsa) IsDir() bool                { return true }
func (e entradaFalsa) Type() fs.FileMode          { return fs.ModeDir }
func (e entradaFalsa) Info() (fs.FileInfo, error) { return nil, fs.ErrNotExist }

func TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir(t *testing.T) {
	const raiz = `C:\cofre`
	casos := []struct {
		nome    string
		caminho string
		d       fs.DirEntry
		quer    bool
	}{
		{"Lstat da raiz falhou: d == nil", raiz, nil, true},
		{"ReadDir da raiz falhou: d != nil, caminho == raiz", raiz, entradaFalsa{"cofre"}, true},
		{"entrada comum ilegivel", `C:\cofre\sub`, entradaFalsa{"sub"}, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := FalhaNaRaiz(raiz, c.caminho, c.d); got != c.quer {
				t.Fatalf("FalhaNaRaiz(%q, %q, %v) = %v, quer %v", raiz, c.caminho, c.d, got, c.quer)
			}
		})
	}
}
```

- [ ] **Step 2: Rodar; deve falhar por símbolo ausente**

Run: `go test ./internal/vault -run TestFalhaNaRaiz`
Expected: `undefined: FalhaNaRaiz`.

- [ ] **Step 3: Escrever a conta e usá-la em `Walk`**

Em `internal/vault/walk.go`, antes de `Walk`:

```go
// FalhaNaRaiz diz se um erro entregue pelo callback de filepath.WalkDir e da
// PROPRIA raiz varrida, e nao de uma entrada dentro dela.
//
// WalkDir tem duas formas de falhar na raiz, e so uma delas vem com d == nil:
//
//   - Lstat(raiz) falhou: um unico callback, d == nil.
//   - Lstat passou e ReadDir(raiz) falhou: DOIS callbacks — o primeiro normal,
//     o segundo com d != nil, caminho == raiz e o erro do ReadDir.
//
// Tratar so d == nil deixa a segunda forma passar como "entrada ilegivel":
// engolida, logada, e a varredura devolve sucesso com zero entradas. Um
// diretorio que existe mas nao pode ser lido (antivirus segurando a pasta,
// no Windows) virava cofre vazio. E a unica conta dessa distincao — Walk e
// watcher.varreDiretorioNovo usam a mesma.
func FalhaNaRaiz(raiz, caminho string, d fs.DirEntry) bool {
	return d == nil || caminho == raiz
}
```

E no callback de `Walk`, troque `if d == nil {` por `if FalhaNaRaiz(v.walkRoot, abs, d) {` — o comentário acima do `if` ganha a frase "Ver FalhaNaRaiz para a segunda forma, que ReadDir produz." e perde a afirmação de que `d == nil` é a única.

- [ ] **Step 4: Usar a conta em `varreDiretorioNovo`**

Em `internal/watcher/watcher.go:245`, troque `if d == nil {` por `if vault.FalhaNaRaiz(dir, caminho, d) {`. O comentário de :230-244 fica; acrescente uma linha: "A segunda forma — ReadDir da raiz falhou, d != nil — e a que vault.FalhaNaRaiz cobre."

- [ ] **Step 5: Teste no watcher, `internal/watcher/varredura_raiz_test.go`**

O que se prova aqui é que o watcher USA a conta. A versão portátil tira a permissão de leitura por `chmod`, que no Windows não faz nada — por isso o build tag:

```go
//go:build !windows

package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestVarreDiretorioNovoNaoEngoleRaizIlegivel(t *testing.T) {
	w, cancel, root, _ := setupTestWatcher(t)
	defer cancel()
	dir := filepath.Join(root, "chegou")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("ReadDir nao falhou com 0o000 (root?); o cenario nao se reproduz aqui")
	}
	err := w.varreDiretorioNovo(context.Background(), dir)
	if err == nil {
		t.Fatal("varreDiretorioNovo devolveu nil para uma raiz que ReadDir nao consegue ler: a varredura reportou sucesso com zero entradas")
	}
}
```

`setupTestWatcher` é o construtor de `internal/watcher/counters_test.go:68` — devolve `(*Watcher, context.CancelFunc, string, *index.Index)`, com o watcher já rodando sobre um `t.TempDir()`. O evento `Create` do `MkdirAll` vai disparar `varreDiretorioNovo` pelo `Run` também; não importa — o teste chama a função diretamente e julga só o retorno dela.

- [ ] **Step 6: Teste Windows em `internal/vault/walk_raiz_windows_test.go`**

```go
//go:build windows

package vault

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// travarDiretorioExclusivo abre dir com dwShareMode = 0. Enquanto o handle
// vive, ReadDir(dir) falha com ERROR_SHARING_VIOLATION e Lstat(dir) passa —
// a segunda forma de falha da raiz que FalhaNaRaiz existe para reconhecer.
// Provado nesta maquina em 2026-09-02; o teste confere de novo e pula, com o
// motivo, se o SO desta vez deixar ler.
func travarDiretorioExclusivo(t *testing.T, dir string) {
	t.Helper()
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatalf("CreateFile exclusivo em %q: %v", dir, err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("handle exclusivo NAO impediu ReadDir nesta maquina; o cenario nao se reproduz")
	}
}

func TestWalkNaoEngoleRaizQueExisteMasNaoLe(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	travarDiretorioExclusivo(t, root)

	var vistos int
	err = v.Walk(context.Background(), func(Entry) error { vistos++; return nil })
	if err == nil {
		t.Fatalf("Walk devolveu nil com %d entradas para uma raiz que ReadDir nao le: cofre inacessivel virou cofre vazio", vistos)
	}
}
```

`golang.org/x/sys` já está no `go.mod` (`v0.47.0`, indireto); se `go build` reclamar de `go.sum`, `go get golang.org/x/sys/windows@v0.47.0` — caminho do pacote, nunca `go mod tidy`.

- [ ] **Step 7: Rodar os três; ver o do Windows FALHAR antes do fix**

Run: `go test ./internal/vault ./internal/watcher -run 'FalhaNaRaiz|RaizQueExiste|RaizIlegivel' -v`
Expected ANTES do Step 3/4: `TestWalkNaoEngoleRaizQueExisteMasNaoLe` FAIL com "Walk devolveu nil com 0 entradas". Se der SKIP, registre o motivo e a Task fica `BLOCKED`: o cenário não se reproduz e não há prova. DEPOIS: PASS. O teste `!windows` não roda nesta máquina; o gate só o compila (`go vet` com `GOOS=linux`) — diga isso no relatório em vez de "testes passam".

- [ ] **Step 8: Prova de mutação**

Troque `return d == nil || caminho == raiz` por `return d == nil`, rode o Step 7, cole a saída (`TestFalhaNaRaiz.../ReadDir_da_raiz_falhou` e o Windows devem falhar), restaure.

- [ ] **Step 9: Gate e commit**

```bash
pwsh -File scripts/verify.ps1
git add internal/vault/walk.go internal/vault/walk_raiz_test.go internal/vault/walk_raiz_windows_test.go internal/watcher/watcher.go internal/watcher/varredura_raiz_test.go
git commit -m "fix(vault): a ReadDir failure on the scanned root is a root failure, not an unreadable entry"
```

(Acrescente `go.sum` ao `git add` só se ele mudou.)

#### Verificações
Além dos passos:
1. `walk.go` e `watcher.go` chamam a MESMA função (`vault.FalhaNaRaiz`); `grep -rn "d == nil" internal/vault internal/watcher` não devolve mais nenhuma decisão de raiz feita à mão.
2. `internal/watcher/watcher.go:63` (varredura inicial, `path == root`) NÃO foi tocado nesta Task — ele já estava certo; se você o migrou para `FalhaNaRaiz`, diga no relatório, e é aceitável desde que o comportamento seja idêntico.
3. O teste Windows (`walk_raiz_windows_test.go`) reprova ANTES do fix com a mensagem que nomeia a raiz — cole a saída do RED.
4. `go list -f '{{.Imports}}' ./internal/watcher` continua sem aresta nova (`vault` já era importado).

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
pwsh -File scripts/mutate.ps1 -Path internal/vault/walk.go `
  -Anchor 'return d == nil || caminho == raiz' `
  -Replacement 'return d == nil' `
  -Test TestFalhaNaRaiz -Package ./internal/vault/
```
E a segunda, no consumidor:
```bash
pwsh -File scripts/mutate.ps1 -Path internal/watcher/watcher.go `
  -Anchor 'vault.FalhaNaRaiz(dir, caminho, d)' `
  -Replacement 'd == nil' `
  -Test TestVarreDiretorioNovoNaoEngoleRaizIlegivel -Package ./internal/watcher/
```
(Este segundo só roda fora de Windows — `//go:build !windows`. Em Windows, faça a mutação à mão e rode `go test ./internal/vault -run TestWalkNaoEngoleRaizQueExisteMasNaoLe`, colando o diff e a saída.)

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

