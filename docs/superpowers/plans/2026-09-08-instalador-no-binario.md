# O instalador é o próprio binário — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `gobsidian` instala, configura e atualiza a si mesmo nas três plataformas, substituindo por completo `install.ps1` (729 linhas) e `installer/install.js` (1 009), que hoje são a mesma lógica escrita duas vezes sem um único teste.

**Architecture:** Três pacotes novos. `internal/hosts` é folha e cuida dos 9 hosts MCP: detectar e fundir JSON. `internal/selfupdate` é o **único** pacote com `net/http`, sob a segunda exceção nomeada à RNF-30 (D-13). `internal/instalar` orquestra: trava global, presença de processos, limpeza, PATH, manifesto e troca do binário.

A peça que dispensa código de plataforma é a **presença por trava de kernel**. Não existe enumeração de processos portável em Go, e o projeto proíbe `if runtime.GOOS ==` em lógica compartilhada. A saída reusa o que já existe: `internal/daemon/trava.go` trava uma **faixa de bytes distante dos dados** — escolhida assim de propósito, para que `doctor` consiga ler o PID de dentro do arquivo travado. Logo, um processo que segura a trava do próprio arquivo de presença durante toda a vida é, ao mesmo tempo, **detectável** (a trava está tomada) e **legível** (o conteúdo diz quem ele é). Processo morto solta a trava sozinho, pelo kernel: não há PID obsoleto a interpretar, que é exatamente a lição registrada em `trava.go`.

**Tech Stack:** Go 1.25.0, cobra, `golang.org/x/sys` (já é dependência e `trava_windows.go` já a usa diretamente), `net/http` só em `internal/selfupdate`. Nenhuma dependência nova — **nunca** `go mod tidy`.

**Spec:** [`docs/superpowers/specs/2026-09-08-instalador-e-encerramento-design.md`](../specs/2026-09-08-instalador-e-encerramento-design.md), commit `4390a1b`. Este plano implementa D-01 a D-11 e D-13. As seções 5 e 6 (logs e anteparo) são o plano 1, já concluído.

## Global Constraints

Valem **todas** as do plano 1 (`2026-09-08-encerramento-e-logs.md`), sem exceção. As que este plano acrescenta:

- **A RNF-30 continua valendo em toda parte menos num lugar.** `net/http` só em `internal/selfupdate`, com host em **constante literal**. URL vinda de variável é recusada, do mesmo jeito que rede vinda de variável já é. `tools/netcheck` ganha a regra e `scripts/check_gates.ps1` ganha três casos: o que recusa, o que aceita, o inverso.
- **Nenhum daemon encerra outro daemon** (D-08). Quem encerra é o instalador, com aval explícito do usuário, listando PID e cofre.
- **Sem elevação** (D-04). Nunca UAC. Destino padrão dentro do perfil do usuário.
- **A limpeza remove só lixo comprovadamente órfão** (D-05). Cache de cofre existente nunca é tocado — reconstruir custou 3021 ms no cofre de referência do dono.
- **Nenhum tipo do SDK MCP cruza para fora de `internal/mcpsrv`.** O instalador escreve configuração de host como JSON genérico; ele não sabe o que é uma tool.
- O grafo de dependências ganha exatamente três nós e nenhuma aresta que crie ciclo. Aresta nova precisa de justificativa escrita.

---

## File Structure

| Arquivo | Responsabilidade | Task |
|---|---|---|
| `internal/instalar/presenca.go` (novo) | processo registra a si mesmo por trava de kernel; lista quem está vivo | 195 |
| `internal/instalar/presenca_test.go` (novo) | vivo aparece, morto não, conteúdo legível sob trava | 195 |
| `internal/daemon/trava.go` | `TentarTravar`/`TravaDeArquivo` exportados para `instalar` reusar a primitiva | 195 |
| `internal/instalar/trava_global.go` (novo) | trava de instalação; `serve` e `daemon` saem se ela estiver tomada | 196 |
| `cmd/gobsidian/serve.go`, `cmd/gobsidian/daemon.go` | consultam a trava global ao subir | 196 |
| `internal/instalar/limpeza.go` (novo) | lixo órfão do diretório de runtime e caches de cofres que sumiram | 197 |
| `internal/doctor/` | `--fix` chama a MESMA limpeza | 197 |
| `internal/hosts/hosts.go` (novo) | os 9 hosts: onde mora o config de cada um | 198 |
| `internal/hosts/merge.go` (novo) | funde a entrada `gobsidian` preservando o resto do JSON | 198 |
| `internal/hosts/testdata/` (novo) | golden files por host | 198 |
| `internal/selfupdate/selfupdate.go` (novo) | release, SHA-256, download; **único** com `net/http` | 199 |
| `tools/netcheck/netcheck.go` | a regra da exceção | 199 |
| `scripts/check_gates.ps1`, `docs/PRD.md` §6.4 | os três casos e a redação normativa | 199 |
| `internal/instalar/manifesto.go` (novo) | o que foi instalado: caminho, versão, hash, PATH, hosts | 200 |
| `internal/instalar/path_windows.go`, `path_other.go` (novos) | entrada de PATH por plataforma, atrás de build tag | 200 |
| `internal/instalar/instalar.go` (novo) | a sequência de D-04/D-05/D-06 e a troca por rename | 200 |
| `cmd/gobsidian/install.go`, `update.go`, `path.go`, `vaults.go` (novos) | os subcomandos | 201 |
| `cmd/gobsidian/main.go` | registro dos subcomandos e o comportamento sem argumentos (D-11) | 201 |
| `bootstrap/install.sh`, `install.ps1`, `install.nu` (novos) | baixar e rodar, e nada mais | 202 |
| `install.ps1`, `installer/` | **apagados** | 202 |
| `README.md` | a forma nova de instalar | 202 |
| `.github/workflows/ci.yml`, `release.yml` | job do instalador; release exige CI verde; `verify.ps1` no CI | 203 |

---

### Task 195: presença — quem está rodando, sem enumerar processos

**Files:**
- Create: `internal/instalar/presenca.go`, `internal/instalar/presenca_test.go`
- Modify: `internal/daemon/trava.go` (exportar a primitiva)

**Interfaces:**
- Produces:
  - `instalar.Presenca` — `{PID int; Cofre string; Papel string; Versao string; Arquivo string}`
  - `instalar.Registrar(runtimeDir, cofre, papel, versao string) (liberar func(), err error)`
  - `instalar.Vivos(runtimeDir string) ([]Presenca, error)`
  - `daemon.TravaDeArquivo` e `daemon.TentarTravar(path string) (*TravaDeArquivo, bool, error)` — os mesmos `travaDeArquivo`/`tentarTravar` de hoje, exportados.

**Por que assim, e não enumerando processos:** Go não tem listagem de processos portável, e a alternativa seria código de plataforma em três variantes só para responder "quem está vivo". `trava.go` já resolve o problema difícil — posse decidida pelo kernel, sem PID obsoleto — e já trava uma faixa **em `1<<62`**, longe do conteúdo, precisamente para que outro processo consiga ler o arquivo. Presença é essa mesma primitiva usada para uma segunda pergunta.

- [ ] **Step 1: Exportar a primitiva em `internal/daemon/trava.go`.** Renomear `travaDeArquivo` → `TravaDeArquivo` e `tentarTravar` → `TentarTravar`, mantendo `TravaEmUso` como está. Atualizar os chamadores internos (`lock.go`, `trava.go`) e os testes do pacote. **Nenhuma mudança de comportamento**; se algum teste falhar, é sinal de que houve.

Run: `go test ./internal/daemon/ -v`

- [ ] **Step 2: Escrever o teste que falha**, em `internal/instalar/presenca_test.go`:

```go
package instalar

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPresencaVivoApareceEMortoNao e a razao de existir do pacote: em
// 2026-09-08 havia DOIS processos servindo o cofre Estudo e gravando o mesmo
// inverted_cache.gob, e isso so ficou visivel comparando milissegundos entre
// linhas de log duplicadas. Nenhum comando respondia "quem esta servindo este
// cofre agora?".
func TestPresencaVivoApareceEMortoNao(t *testing.T) {
	dir := t.TempDir()

	liberar, err := Registrar(dir, `C:\Cofre`, "serve", "v9.9.9")
	if err != nil {
		t.Fatalf("Registrar() error = %v", err)
	}

	vivos, err := Vivos(dir)
	if err != nil {
		t.Fatalf("Vivos() error = %v", err)
	}
	if len(vivos) != 1 {
		t.Fatalf("Vivos() = %d presencas, esperado 1", len(vivos))
	}
	if vivos[0].PID != os.Getpid() {
		t.Errorf("PID = %d, esperado %d", vivos[0].PID, os.Getpid())
	}
	if vivos[0].Cofre != `C:\Cofre` || vivos[0].Papel != "serve" || vivos[0].Versao != "v9.9.9" {
		t.Errorf("conteudo ilegivel sob a trava: %+v", vivos[0])
	}

	// Soltar a trava e o que o KERNEL faz quando o processo morre. Depois
	// disso, a presenca nao conta mais -- e nao ha PID obsoleto a interpretar,
	// que e a licao registrada em internal/daemon/trava.go.
	liberar()

	vivos, err = Vivos(dir)
	if err != nil {
		t.Fatalf("Vivos() error = %v", err)
	}
	if len(vivos) != 0 {
		t.Fatalf("Vivos() = %d presencas depois de liberar, esperado 0", len(vivos))
	}
}

// TestPresencaArquivoOrfaoNaoContaEPodeSerLimpo: o arquivo SOBREVIVE ao
// processo, de proposito (o mesmo desenho de trava.go, que nunca remove o
// lock). O que decide e a trava, nunca a existencia.
func TestPresencaArquivoOrfaoNaoContaEPodeSerLimpo(t *testing.T) {
	dir := t.TempDir()
	orfao := filepath.Join(dir, "abc.serve.4242"+SufixoDePresenca)
	if err := os.WriteFile(orfao, []byte(`{"pid":4242,"cofre":"X","papel":"serve"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	vivos, err := Vivos(dir)
	if err != nil {
		t.Fatalf("Vivos() error = %v", err)
	}
	if len(vivos) != 0 {
		t.Fatalf("um arquivo de presenca sem trava contou como processo vivo: %+v", vivos)
	}
}
```

- [ ] **Step 3: Rodar e ver falhar.** Expected: `undefined: Registrar`.

- [ ] **Step 4: Implementar `presenca.go`.** O arquivo é `<runtimeDir>/<chave>.<papel>.<pid>.presenca`, o conteúdo é JSON, e a trava é segurada até `liberar()`. `Vivos` percorre o diretório, pergunta `daemon.TravaEmUso` para cada `.presenca` e lê o conteúdo dos tomados.

- [ ] **Step 5: Rodar e ver passar.**

- [ ] **Step 6: Prova de mutação COMPILANDO.** Fazer `Vivos` contar por existência de arquivo em vez de por trava tomada; `TestPresencaArquivoOrfaoNaoContaEPodeSerLimpo` tem de falhar nomeando "contou como processo vivo". Colar a saída.

- [ ] **Step 7: Gate e commit.**

---

### Task 196: trava global de instalação

**Files:**
- Create: `internal/instalar/trava_global.go`, `internal/instalar/trava_global_test.go`
- Modify: `cmd/gobsidian/serve.go`, `cmd/gobsidian/daemon.go`

**Interfaces:**
- Produces: `instalar.TomarTravaGlobal(runtimeDir string) (liberar func(), err error)` e `instalar.InstalacaoEmCurso(runtimeDir string) (bool, error)`.

**Por que ela é o mecanismo principal, e não rede de segurança:** com D-06/D-07 o instalador encerra todos os processos. Um host MCP respawna o servidor em segundos — medido em 2026-09-07, o host reconectou 15 s depois da desconexão. Sem a trava, o respawn cai no meio da troca do binário e duas versões voltam a conviver, que é o defeito inteiro que este trabalho fecha.

- [ ] **Step 1: Teste que falha:** com a trava tomada, `InstalacaoEmCurso` devolve `true`; solta, devolve `false`.

- [ ] **Step 2: Rodar e ver falhar.**

- [ ] **Step 3: Implementar** sobre `daemon.TentarTravar`, arquivo `<runtimeDir>/instalacao.lock`, nunca removido.

- [ ] **Step 4: Ligar em `serve` e `daemon`.** Ao subir, antes de qualquer montagem: se `InstalacaoEmCurso`, logar `WARN` com mensagem explícita e **sair com código 0** — instalação em curso não é falha do servidor, e um código de erro faria o host tentar de novo em laço.

- [ ] **Step 5: Teste de ponta:** `runDaemon` com a trava tomada não monta índice nenhum.

- [ ] **Step 6: Prova de mutação.**

- [ ] **Step 7: Gate e commit.**

---

### Task 197: limpeza de lixo comprovadamente órfão

**Files:**
- Create: `internal/instalar/limpeza.go`, `internal/instalar/limpeza_test.go`
- Modify: `internal/doctor/` (`--fix` chama a mesma função)

**Interfaces:**
- Produces: `instalar.Limpar(runtimeDir, cacheRaiz string, aplicar bool) (Relatorio, error)`, com `Relatorio{Locks, Sockets, Presencas, Caches []string; Bytes int64}`. Com `aplicar=false` ela **só relata** — é o que `doctor` sem `--fix` mostra.

**A regra é mecânica, nunca heurística** (D-05):

| Alvo | Condição |
|---|---|
| `.lock`, `.presenca` | trava **não** tomada (`daemon.TravaEmUso`) **e** cofre não existe mais |
| `.sock` | sem ouvinte (`ipc.AlguemEscuta`) **e** cofre não existe mais |
| `.log` | acima de 5 MB → rotacionado, **nunca** apagado |
| diretório de cache | cofre não existe mais |

"Cofre não existe mais" é decidível sem adivinhação: `search.CacheHeader.VaultPath` já guarda o caminho e já é gravado (`internal/search/persist.go:44`, `persist_codec.go:187`). Medido em 2026-09-08 no diretório do dono: **960 `.lock`** e **11 `.log`** (um com 727 261 bytes). Os **4 `.sock`** presentes **não** são lixo — as chaves são `Estudo`, `Jurisprudência`, `Oral` e `Revisão`, e os quatro cofres existem; socket sem daemon rodando é estado normal. A primeira redação deste plano dizia que três eram de cofres inexistentes, sem ter verificado.

- [ ] **Step 1: Teste que falha**, com um diretório de runtime falso contendo os quatro casos **e** os quatro inversos (lock tomado, socket com ouvinte, log pequeno, cache de cofre existente). Sem os inversos, uma limpeza que apagasse tudo passaria.

- [ ] **Step 2 a 5:** RED, implementar, GREEN, ligar em `doctor --fix`.

- [ ] **Step 6: Prova de mutação:** remover a condição "cofre não existe mais" e provar que o caso inverso falha.

- [ ] **Step 7: Gate e commit.**

---

### Task 198: os 9 hosts MCP

**Files:**
- Create: `internal/hosts/hosts.go`, `internal/hosts/merge.go`, `internal/hosts/hosts_test.go`, `internal/hosts/testdata/`

**Interfaces:**
- Produces: `hosts.Host{Chave, Nome string; Caminho func(raiz string) string}`, `hosts.Detectar(raiz string) []Host`, `hosts.Fundir(caminho string, entrada Entrada) error`, `hosts.Entrada{Command string; Args []string}`.

Os nove, extraídos de `install.ps1:503-640` — **ler o arquivo antes de escrever, e não confiar nesta lista de memória**: `claude-desktop`, `claude-code`, `gemini-cli`, `antigravity`, `antigravity-ide`, `codex`, `vscode`, `cursor`, `windsurf`.

`Fundir` **preserva o resto do JSON**: o arquivo é do usuário e tem outros servidores MCP dentro. `install.ps1:462` já faz isso; a diferença é que aqui haverá teste.

- [ ] **Step 1: Ler `install.ps1:455-690` inteiro** e transcrever os caminhos e o formato de cada host para `testdata/`, um golden por host: entrada antes, entrada depois.

- [ ] **Step 2 a 7:** RED com os goldens, implementar, GREEN, mutação (fundir que sobrescreve o arquivo inteiro tem de falhar), gate, commit.

---

### Task 199: `selfupdate`, e a segunda exceção da RNF-30

**Files:**
- Create: `internal/selfupdate/selfupdate.go`, `internal/selfupdate/selfupdate_test.go`
- Modify: `tools/netcheck/netcheck.go`, `scripts/check_gates.ps1`, `docs/PRD.md` §6.4

**Interfaces:**
- Produces: `selfupdate.Transporte` (interface de uma função só, para o teste não precisar de rede), `selfupdate.UltimaVersao(ctx, t Transporte) (Release, error)`, `selfupdate.Baixar(ctx, t Transporte, r Release, destino string) error` — que **confere o SHA-256 e aborta na divergência** (D-03).

**A ordem importa:** a regra do `netcheck` entra **antes** do pacote, para que a primeira versão do pacote já seja validada por ela. Gate que nasce depois do código que ele deveria ter recusado não prova nada.

- [ ] **Step 1: Escrever os três casos em `scripts/check_gates.ps1`** — `net/http` em `internal/selfupdate` é **aceito**; em qualquer outro pacote é **recusado**; host vindo de variável em `internal/selfupdate` é **recusado**.

- [ ] **Step 2: Rodar `check_gates.ps1` e ver os casos falharem** (a regra ainda não existe).

- [ ] **Step 3: Implementar a regra em `tools/netcheck`.**

- [ ] **Step 4: `check_gates.ps1` verde.**

- [ ] **Step 5: Redação normativa em `docs/PRD.md` §6.4**, citando a decisão do dono de 2026-09-08 e o precedente de 2026-08-05.

- [ ] **Step 6: Só então** escrever `internal/selfupdate`, com teste usando transporte falso em memória — **nenhum `net` em teste**.

- [ ] **Step 7: Prova de mutação, gate e commit.**

---

### Task 200: manifesto, PATH e a sequência de instalação

**Files:**
- Create: `internal/instalar/manifesto.go`, `internal/instalar/path_windows.go`, `internal/instalar/path_other.go`, `internal/instalar/instalar.go` e seus testes

**Interfaces:**
- Produces: `instalar.Manifesto{Binario, Versao, Hash string; PathAdicionado bool; Hosts []string; Em time.Time}`, `instalar.LerManifesto/GravarManifesto`, `instalar.AdicionarAoPath/RemoverDoPath`, `instalar.Instalar(ctx, Opcoes) (Resultado, error)`.

A troca do binário usa **rename**, e não é otimização: `gobsidian update` **é** o executável que precisa substituir. Medido em 2026-09-08 nesta máquina — sobrescrever um `.exe` em execução falha com `Device or resource busy`; renomeá-lo funciona; o processo antigo segue vivo lendo do arquivo renomeado; e apagar o renomeado funciona mesmo com ele rodando.

- [ ] **Step 1 a 7:** o ciclo de sempre, com o teste da sequência usando interfaces pequenas para as operações perigosas (encerrar processo, mover arquivo), de modo que a sequência seja testável **sem tocar na máquina**.

---

### Task 201: os subcomandos

**Files:**
- Create: `cmd/gobsidian/install.go`, `update.go`, `path.go`, `vaults.go` e testes
- Modify: `cmd/gobsidian/main.go`

`gobsidian` **sem argumentos** (D-11): se não instalado **e** o terminal for interativo, autoinstala; caso contrário, ajuda. O teste de interatividade existe para que um host MCP que invoque o binário sem argumento **nunca** dispare uma instalação.

- [ ] **Step 1 a 7:** ciclo de sempre. O teste de D-11 cobre os quatro cruzamentos: instalado × não instalado, interativo × não interativo.

---

### Task 202: bootstraps, e a remoção do que eles substituem

**Files:**
- Create: `bootstrap/install.sh`, `bootstrap/install.ps1`, `bootstrap/install.nu`
- Delete: `install.ps1`, `installer/`
- Modify: `README.md`

Cada bootstrap faz **uma coisa**: baixa o executável para um diretório temporário e o roda. Nada de detectar cofre, nada de mexer em PATH, nada de fundir JSON — isso tudo passou para o binário, onde é testável.

- [ ] **Step 1: Escrever os três.** ASCII puro na saída.
- [ ] **Step 2: Rodar cada um numa máquina limpa o suficiente** (diretório temporário) e colar a saída.
- [ ] **Step 3: Apagar `install.ps1` e `installer/`** — `git rm`, nunca `git clean`.
- [ ] **Step 4: README.** A seção de instalação passa a descrever o binário.
- [ ] **Step 5: `check_doc_refs` e `check_readme_anchors` verdes** — os dois vão acusar as referências mortas, e é para isso que existem.
- [ ] **Step 6: Gate e commit.**

---

### Task 203: CI

**Files:**
- Modify: `.github/workflows/ci.yml`, `.github/workflows/release.yml`

- [ ] **Step 1: Job do instalador**, nas três plataformas: instala em diretório temporário, confere PATH e manifesto.
- [ ] **Step 2: `release.yml` passa a exigir `ci.yml` verde** — hoje uma tag libera com CI vermelho.
- [ ] **Step 3: `verify.ps1` roda no CI.** Hoje o gate documentado e o CI são duas contas da mesma regra, e `mutate.ps1`, `check_gates.ps1` e `audit_reports.ps1` nunca rodam lá.
- [ ] **Step 4: `go-version` alinhado** com a toolchain do dono (1.26.5 em 2026-09-08).
- [ ] **Step 5: Smoke test dos três bootstraps.**
- [ ] **Step 6: Commit.**
