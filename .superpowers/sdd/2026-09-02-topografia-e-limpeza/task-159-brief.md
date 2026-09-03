### Task 159: `internal/vaulttest` — handle exclusivo, placeholder somente-nuvem e prazo comum

**Files:**
- Create: `internal/vaulttest/doc.go`
- Create: `internal/vaulttest/exclusivo_windows.go`
- Create: `internal/vaulttest/exclusivo_other.go`
- Create: `internal/vaulttest/somentenuvem_windows.go`
- Create: `internal/vaulttest/somentenuvem_other.go`
- Create: `internal/vaulttest/prazo.go`
- Create: `internal/vaulttest/exclusivo_windows_test.go`
- Modify: `internal/search/cloudonly_update_windows_test.go:44-66` (apagar `travarExclusivo`, usar `vaulttest.TravarExclusivo`)
- Modify: `internal/service/erro_engolido_windows_test.go:23` (apagar `travaExclusiva`; `:60-66` tirar a guarda)
- Modify: `internal/service/move_atomico_windows_test.go:70-85` (tirar a guarda `if origemExiste && destinoExiste && err == nil`)
- Modify: `internal/index/replace_duas_fases_windows_test.go:22` (apagar `travaLeitura`)
- Modify: `internal/index/build_descarte_windows_test.go:10` (apagar `lockFileForTest`)
- Modify: `internal/index/classify_cloudonly_windows_test.go:24` (apagar `marcarSomenteNuvem`)
- Modify: `cmd/gobsidian/boot_indice_busca_windows_test.go:28` (apagar `marcarOffline`)
- Modify: `internal/vault/walk_windows_test.go` (se a Task 147 deixou `travarDiretorioExclusivo` ali: migrar)
- Modify: `internal/daemon/daemon_test.go:23`, `internal/ipc/ipc_test.go:23`, `cmd/gobsidian/serve_test.go:18` (apagar `boundedWait` local, usar `vaulttest.Prazo`)
- Modify: `docs/ESTRUTURA.md` (entrada `internal/vaulttest/`), `docs/papeis/testador.md` (seção "handle exclusivo": apontar para o pacote)
- Modify: `CLAUDE.md` (grafo: `vaulttest → vault`, marcado "só em _test" — é pacote de teste, importado só por `_test.go`)

**Interfaces:**
- Consumes: `vault.LongPath(abs string) string` (`internal/vault/longpath_windows.go:27`).
- Produces (pacote `vaulttest`, importado apenas por arquivos `_test.go` de pacotes externos `*_test` ou `package main`):
  - `func TravarExclusivo(t testing.TB, abs string)` — Windows: `CreateFile(GENERIC_READ|GENERIC_WRITE, share=0)`, `t.Cleanup` fecha, e **prova** que `os.ReadFile(abs)` falha antes de devolver (o modelo de `search/cloudonly_update_windows_test.go:50`). Outros SO: `t.Skip("handle exclusivo e semantica do Windows")`.
  - `func TravarDiretorioExclusivo(t testing.TB, abs string)` — mesma coisa sobre diretório (`FILE_FLAG_BACKUP_SEMANTICS`), prova que `os.ReadDir(abs)` falha. Outros SO: Skip.
  - `func MarcarSomenteNuvem(t testing.TB, abs string)` — Windows: `FILE_ATTRIBUTE_OFFLINE`, cleanup restaura `NORMAL`, e prova `vault.IsCloudOnly` (ou o `Classify` equivalente que o pacote expõe — ver Verificações) devolve verdadeiro antes de devolver. Outros SO: Skip.
  - `const Prazo = 5 * time.Second` — o único `boundedWait`. A Task 165 depende dele.

**Por que um pacote de teste e não cópias:** cinco cópias de "handle exclusivo" e só uma prova que a trava trava (`docs/ARMADILHAS.md` já registra o mecanismo: "um handle exclusivo que pede só GENERIC_READ não barra `os.ReadFile`"). As quatro que não provam deixam `service/erro_engolido_windows_test.go:60-66` e `move_atomico_windows_test.go:70-85` guardarem toda asserção com `if … && err == nil` — se o share mode permitir a leitura, os testes passam sem afirmar nada. Um pacote com a prova dentro do helper faz a asserção incondicional.

**Ciclo de import:** `vaulttest` importa `vault`. Só arquivos `package vault_test` (externo) podem usá-lo dentro de `internal/vault/` — `walk_windows_test.go` é `vault_test`, `cloudonly_info_windows_test.go` é `package vault` e **fica como está**. `vaulttest` **não** importa `index`, `search`, `service`, `parser` — é isto que impede o ciclo nos testes externos desses pacotes. Não é `helpers.go`: o pacote tem um nome de domínio e cada arquivo uma responsabilidade.

- [ ] **Step 1: Criar o pacote com a prova dentro do helper**

`internal/vaulttest/doc.go`:

```go
// Package vaulttest oferece aos testes de outros pacotes as condicoes de
// ambiente que o produto promete respeitar e que so o sistema operacional
// pode criar: um arquivo ou diretorio que NAO pode ser aberto, um arquivo
// somente-nuvem, e o prazo unico de espera dos testes.
//
// Cada helper PROVA a condicao antes de devolver. Um handle exclusivo que nao
// barra a leitura tornaria vazia toda asserção de "nao abriu" — foi o que
// aconteceu quando cinco copias divergiram e so uma conferia.
//
// Importado apenas por arquivos _test.go. Importa vault e mais nada do
// dominio, para nunca fechar ciclo com quem o usa.
package vaulttest
```

`internal/vaulttest/exclusivo_windows.go`:

```go
//go:build windows

package vaulttest

import (
	"os"
	"testing"

	"github.com/jonyd/gobsidian/internal/vault"
	"golang.org/x/sys/windows"
)

// TravarExclusivo segura um handle exclusivo sobre o arquivo ate o fim do
// teste e CONFERE que ele barra os.ReadFile antes de devolver.
//
// Leitura E escrita: um handle exclusivo que pede so GENERIC_READ nao barra
// o os.ReadFile (medido; ver docs/ARMADILHAS.md).
func TravarExclusivo(t testing.TB, abs string) {
	t.Helper()
	h := abrirExclusivo(t, abs, 0)
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	if _, err := os.ReadFile(abs); err == nil {
		t.Fatalf("vaulttest: o handle exclusivo nao barrou a leitura de %s; a prova de 'nao abriu' seria vazia", abs)
	}
}

// TravarDiretorioExclusivo e o equivalente para diretorio: com share mode 0 o
// os.ReadDir falha, que e a condicao que vault.Walk e o watcher precisam
// distinguir de "diretorio vazio".
func TravarDiretorioExclusivo(t testing.TB, abs string) {
	t.Helper()
	h := abrirExclusivo(t, abs, windows.FILE_FLAG_BACKUP_SEMANTICS)
	t.Cleanup(func() { _ = windows.CloseHandle(h) })
	if _, err := os.ReadDir(abs); err == nil {
		t.Fatalf("vaulttest: o handle exclusivo nao barrou a listagem de %s", abs)
	}
}

func abrirExclusivo(t testing.TB, abs string, flags uint32) windows.Handle {
	t.Helper()
	p, err := windows.UTF16PtrFromString(vault.LongPath(abs))
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL|flags, 0)
	if err != nil {
		t.Skipf("vaulttest: nao foi possivel abrir %s em modo exclusivo: %v", abs, err)
	}
	return h
}
```

`internal/vaulttest/exclusivo_other.go`:

```go
//go:build !windows

package vaulttest

import "testing"

// TravarExclusivo nao existe fora do Windows: POSIX nao tem share mode. O
// teste que depende dele pula, e o nome do pulo diz por que.
func TravarExclusivo(t testing.TB, _ string) {
	t.Helper()
	t.Skip("handle exclusivo e semantica do Windows")
}

func TravarDiretorioExclusivo(t testing.TB, _ string) {
	t.Helper()
	t.Skip("handle exclusivo e semantica do Windows")
}
```

`internal/vaulttest/somentenuvem_windows.go` — transcrever `marcarSomenteNuvem` de `internal/index/classify_cloudonly_windows_test.go:24-44` (incluindo o comentário sobre `RECALL_ON_DATA_ACCESS` não ser gravável) como `func MarcarSomenteNuvem(t testing.TB, abs string)`, e acrescentar antes do `return` a prova:

```go
	if !vault.IsCloudOnly(abs) {
		t.Fatalf("vaulttest: FILE_ATTRIBUTE_OFFLINE nao fez vault.IsCloudOnly(%s) responder verdadeiro", abs)
	}
```

(Se `vault.IsCloudOnly` tiver assinatura diferente — conferir com `gopls` em `internal/vault/cloudonly*.go` — usar a que existe. **Não** criar função nova em `vault` para isso.)

`internal/vaulttest/somentenuvem_other.go`: `MarcarSomenteNuvem` que faz `t.Skip("FILE_ATTRIBUTE_OFFLINE e atributo NTFS")`.

`internal/vaulttest/prazo.go`:

```go
package vaulttest

import "time"

// Prazo e o unico limite de espera dos testes que aguardam algo assincrono
// (socket, handshake, desligamento). Um defeito real nao pode travar
// "go test -race ./..." ate o timeout de 10 minutos; e tres pacotes com tres
// valores (2 s, 3 s, 5 s) eram tres respostas para a mesma pergunta.
const Prazo = 5 * time.Second
```

- [ ] **Step 2: Teste do próprio pacote (Windows)**

`internal/vaulttest/exclusivo_windows_test.go` (`package vaulttest_test`):

```go
//go:build windows

package vaulttest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jonyd/gobsidian/internal/vaulttest"
)

// A prova dentro do helper e o que o pacote vende. Este teste confere que ela
// existe: depois de TravarExclusivo, a leitura falha; depois do Cleanup, volta
// a funcionar (o handle foi fechado e nao vazou para o proximo teste).
func TestTravarExclusivoBarraLeituraEDevolveNoCleanup(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Run("travado", func(t *testing.T) {
		vaulttest.TravarExclusivo(t, abs)
		if _, err := os.ReadFile(abs); err == nil {
			t.Fatal("leitura passou com handle exclusivo aberto")
		}
	})
	if _, err := os.ReadFile(abs); err != nil {
		t.Fatalf("depois do Cleanup a leitura devia voltar: %v", err)
	}
}

func TestTravarDiretorioExclusivoBarraListagem(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Run("travado", func(t *testing.T) {
		vaulttest.TravarDiretorioExclusivo(t, dir)
		if _, err := os.ReadDir(dir); err == nil {
			t.Fatal("ReadDir passou com handle exclusivo aberto")
		}
	})
	if _, err := os.ReadDir(dir); err != nil {
		t.Fatalf("depois do Cleanup a listagem devia voltar: %v", err)
	}
}
```

Run: `go test ./internal/vaulttest/ -run 'TestTravar' -v` — Expected: PASS nos dois.

- [ ] **Step 3: Migrar os seis helpers e tirar as guardas**

Para cada arquivo listado em **Files**, apagar o helper local e substituir a chamada por `vaulttest.X`. Nas duas guardas de `service`:

`internal/service/move_atomico_windows_test.go:70-85` — o bloco hoje é `if origemExiste && destinoExiste && err == nil { … asserções … }`. Fica:

```go
	if !origemExiste || !destinoExiste {
		t.Fatalf("o move nao pode ter apagado a origem nem criado o destino com o handle exclusivo aberto: origem=%v destino=%v", origemExiste, destinoExiste)
	}
	if err == nil {
		t.Fatal("MoveNote devia falhar com o destino travado")
	}
	// … asserções que estavam dentro do if, agora incondicionais …
```

`internal/service/erro_engolido_windows_test.go:60-66` — mesma transformação: o `if` que envolve as asserções vira `t.Fatalf` na negação. Ler o teste antes: as asserções que ele guarda são o contrato do teste; nenhuma pode ser perdida.

Run: `go test -race ./internal/... ./cmd/... 2>&1 | tail -30` — Expected: `ok` em todos; os testes Windows que antes podiam passar em silêncio agora afirmam.

- [ ] **Step 4: Prova de que a guarda removida vale**

Em `internal/service/move_atomico_windows_test.go`, trocar temporariamente a chamada `vaulttest.TravarExclusivo(t, destino)` por nada (comentar a linha). Rodar `go test ./internal/service/ -run TestMoveAtomico -v`. Expected: **FAIL** — antes desta tarefa, sem trava o teste passava em silêncio; agora `t.Fatal("MoveNote devia falhar com o destino travado")` nomeia. Restaurar. Colar a saída no relatório.

- [ ] **Step 5: `boundedWait` → `vaulttest.Prazo`**

Em `internal/daemon/daemon_test.go:23`, `internal/ipc/ipc_test.go:23`, `cmd/gobsidian/serve_test.go:18`: apagar a const local e o comentário; `gopls rename` não serve (a const some), então substituir cada uso por `vaulttest.Prazo` e acrescentar o import. `cmd/gobsidian/serve_test.go` usava 2 s: o prazo sobe para 5 s — só afeta o pior caso (falha), não o caso feliz.

Run: `go test -race ./internal/daemon/ ./internal/ipc/ ./cmd/... 2>&1 | tail -5` — Expected: `ok`.

- [ ] **Step 6: Documentação e grafo**

`CLAUDE.md`, bloco "Quatro arestas existem só em teste": acrescentar `vaulttest → vault` na lista de teste e uma linha dizendo que `vaulttest` é importado só por `_test.go`. `docs/ESTRUTURA.md`: entrada `internal/vaulttest/` com uma linha. `docs/papeis/testador.md`: onde fala de handle exclusivo, apontar para `vaulttest.TravarExclusivo` e apagar a receita inline se houver.

Run: `pwsh -File scripts/verify.ps1` — Expected: verde, 14 etapas.

- [ ] **Step 7: Commit**

```bash
git add internal/vaulttest/ internal/search/cloudonly_update_windows_test.go internal/service/erro_engolido_windows_test.go internal/service/move_atomico_windows_test.go internal/index/replace_duas_fases_windows_test.go internal/index/build_descarte_windows_test.go internal/index/classify_cloudonly_windows_test.go cmd/gobsidian/boot_indice_busca_windows_test.go internal/vault/walk_windows_test.go internal/daemon/daemon_test.go internal/ipc/ipc_test.go cmd/gobsidian/serve_test.go docs/ESTRUTURA.md docs/papeis/testador.md CLAUDE.md
git commit -m "test(vaulttest): one exclusive-handle helper that proves it locks, and one wait budget"
```

#### Verificações

- `go list -deps ./internal/vaulttest/ | grep gobsidian` mostra **só** `internal/vault` e `internal/vaulttest` (e `internal/text` se `vault` o importar — conferir; nada mais).
- `grep -rn "CreateFile(" --include=*_test.go internal/ cmd/` devolve zero fora de `internal/vaulttest/`.
- `grep -rn "boundedWait" --include=*_test.go .` devolve zero.
- `grep -rn "FILE_ATTRIBUTE_OFFLINE" --include=*_test.go internal/ cmd/` devolve zero fora de `internal/vaulttest/`.
- Step 4 colado no relatório com o FAIL.
- `verify.ps1` verde; `GOOS=linux go vet ./...` verde (os `_other.go` compilam).

#### Regras de execução

- Nunca `git checkout/restore/stash/clean/reset`. Commit por caminho explícito.
- Sem `helpers.go`/`utils.go`/`common.go`. Um arquivo por responsabilidade.
- `vaulttest` não ganha import de `index`, `search`, `service`, `parser`, `mcpsrv` — se um teste precisar de algo desses, o helper não pertence aqui.
- Código de plataforma atrás de build tag em arquivo separado.
- Não apagar asserção nenhuma dos testes migrados; só a guarda `if` que as tornava condicionais.

#### Comando de mutação

Esta tarefa não tem prova de mutação por `mutate.ps1`: a regra provada é de teste, não de produto, e a prova é o Step 4 (remover a trava, ver o FAIL nomeado, restaurar).

#### Contrato de relatório

`.superpowers/sdd/2026-09-02-topografia-e-limpeza/task-159-report.md`: status, SHA do commit, saída de `go list -deps`, saída do Step 4 (FAIL) e do `verify.ps1` (última linha), lista dos seis helpers apagados com `arquivo:linha` original.

