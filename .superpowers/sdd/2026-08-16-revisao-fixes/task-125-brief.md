# Task 125 — `doctor` diagnostica o daemon

**Tier: modelo barato.** A lista de checagens está fechada abaixo.

#### Onde encaixa
Depois da 124, que cria as mensagens que este comando vai ler.

#### Por que existe
`doctor` é o comando que alguém roda quando **já está confuso**, e hoje ele não
diz uma palavra sobre a metade do sistema que quebra: nem socket, nem lock, nem
log, nem daemon vivo. `internal/doctor/checks.go` tem `checkCacheDir` e nenhum
equivalente para o runtime.

O diagnóstico que originou esta tarefa custou quatro camadas de investigação
manual — processos e parentesco, `%LOCALAPPDATA%\gobsidian\run\`, os logs do
host em `%LOCALAPPDATA%\Claude\Logs\mcp-server-*.log`, e uma reprodução com
stderr capturado. **Todas as quatro respostas estavam disponíveis para um
programa.**

#### O que vincula esta tarefa
- **Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`.
  Console PowerShell em CP-850 renderiza o resto como lixo, e este é justamente
  o comando de quem já está perdido.
- **`doctor` e `version` imprimem em stdout de propósito** — são CLI, não
  servidores. A distinção merece comentário onde aparecer.
- **Cor decidida pelo destino, não por `os.Stdout` global** (`internal/console`),
  senão `doctor > relatorio.txt` sai sujo.
- **`doctor` sai 1 nos checks halting** — corrigido antes, não regredir.

#### As checagens que entram
Uma por linha, cada uma com nome estável:

1. **`socket_path`** — o caminho derivado de `ipc.SocketPath(cfg.VaultPath)`, e
   **o que existe lá**, distinguindo: ausente · socket (reparse point) · arquivo
   comum · **diretório** · outro. A distinção não é acadêmica: medido em
   2026-08-26, `net.Dial("unix", …)` devolve `10061` (refused) para arquivo
   comum, socket órfão **e** caminho inexistente, mas `10022`
   (`An invalid argument was supplied`) para **diretório**. Só o `10022` estava
   nos logs do dono, e nenhum dos reprodutores conhecidos o explica — o check
   existe para que a próxima ocorrência se explique sozinha.
2. **`daemon_vivo`** — tenta `DialAndHandshake` com prazo curto. `[OK]` com o
   PID quando responde; `[i]` quando não há daemon (não é erro); `[!]` quando o
   arquivo existe e o dial falha, **imprimindo o errno numérico**.
3. **`daemon_log`** — existência, tamanho, idade e as **três últimas linhas** de
   `<socket>.log`. É onde a Task 124 passou a escrever a causa.
4. **`locks_orfaos`** — `*.sock.lock` cujo PID não corresponde a processo vivo.
   Medido: cinco deles na máquina do dono, de 15 a 19/08, todos com PID morto.
5. **`grafia_do_cofre`** — confere que `cfg.VaultPath` existe **e** compara com
   a grafia real do disco. `C:\Users\jonyd\Obsidian\Jurisprudencia` não existe;
   `Jurisprudência` existe; nada dizia isso a ninguém.

Nenhuma delas é halting salvo a 5 quando o cofre não existe — aí já é falha de
cofre, que o `doctor` hoje trata.

#### O que prova esta tarefa
Teste por checagem, com o estado montado em `t.TempDir()`: socket ausente;
arquivo comum no lugar do socket; **diretório** no lugar do socket; lock com PID
morto; cofre com grafia divergente. Cada um afirma o **marcador e o texto**, não
só o status.

Prova de mutação: apagar a distinção diretório × arquivo comum na checagem 1
(devolver o mesmo texto para os dois) ⇒ o teste do diretório reprova.

#### Verificações
Além dos passos:
1. `doctor > relatorio.txt` sai **sem** códigos de cor — a decisão é pelo
   destino, não por `os.Stdout` global.
2. Marcadores ASCII puros. Nenhum caractere fora de ASCII na saída.
3. `ExitCode` continua 1 nos checks halting e 0 no resto: daemon ausente é
   `[i]`, não falha.
4. A checagem `socket_path` não pode **abrir** o socket para classificá-lo —
   `os.Lstat` e atributos bastam, e abrir muda o que se está diagnosticando.
5. `gobsidian doctor --vault <cofre real>` roda sem daemon vivo e com daemon
   vivo, e diz coisas diferentes nos dois.

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde, contagem de passos colada.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`.
- Nunca `go mod tidy`.
- `doctor` imprime em **stdout de propósito** — é CLI, não servidor. A distinção
  merece comentário onde aparecer.
- Escopo não encolhe em silêncio: se uma das cinco checagens não couber,
  entregue as outras quatro e **diga qual ficou de fora e por quê**.

#### Comando de mutação
A âncora está no código que você vai escrever; copie a linha literal depois de
escrevê-la. O alvo é a distinção diretório × arquivo comum:

```bash
pwsh -File scripts/mutate.ps1 -Path internal/doctor/checks.go `
  -Anchor '<a linha que devolve o texto do caso DIRETORIO>' `
  -Replacement '<o mesmo texto do caso ARQUIVO COMUM>' `
  -Test TestCheckSocketPathDistingueDiretorio -Package ./internal/doctor/
```

**Files:** `internal/doctor/checks.go`, `internal/doctor/doctor.go`, os
`_test.go` correspondentes, e `docs/OPERACAO.md` (a seção de diagnóstico).

#### Contrato de relatório
A saída **literal** de `gobsidian doctor --vault <fixture>` em dois estados —
um saudável e um com socket quebrado — coladas lado a lado. Mais a prova de
mutação e o `verify.ps1`.

---

