# Task 124 — observabilidade do daemon: quem morre diz por quê

**Status:** DONE
**Commit:** `499e744` — `fix(daemon): make silent failures diagnosable, and repair scan root handling`

---

## Evidência de TDD

### RED

```
$ go test -run 'TestDaemonComCofreInexistenteRegistraCausa|TestServeEmProcessoComCofreInexistenteRegistraCausa' ./cmd/gobsidian/

--- FAIL: TestDaemonComCofreInexistenteRegistraCausa (0.03s)
    daemon_log_test.go:49: nenhum log de ERROR antes da saida.
        log foi:
        time=2026-08-26T08:11:21.838-03:00 level=INFO msg="daemon iniciado" vault=C:\...\cofre-que-nao-existe socket=C:\Users\jonyd\AppData\Local\gobsidian\run\8b07cb2eda25261e.sock read_only=false ociosidade_s=1
--- FAIL: TestServeEmProcessoComCofreInexistenteRegistraCausa (0.02s)
    daemon_log_test.go:88: nenhum log de ERROR antes da saida.
        log foi:
    daemon_log_test.go:91: o log nao nomeia o caminho recusado "C:\\...\\outro-cofre-ausente".
        log foi:
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	4.549s
```

O RED reproduziu a evidência de campo **literalmente**: uma linha `INFO "daemon iniciado"` e nada mais — exatamente o conteúdo dos dois `.sock.log` da máquina do dono em 24/08.

### GREEN

```
$ go test -run 'TestDaemonComCofreInexistenteRegistraCausa|TestServeEmProcessoComCofreInexistenteRegistraCausa' -v ./cmd/gobsidian/

=== RUN   TestDaemonComCofreInexistenteRegistraCausa
--- PASS: TestDaemonComCofreInexistenteRegistraCausa (0.03s)
=== RUN   TestServeEmProcessoComCofreInexistenteRegistraCausa
--- PASS: TestServeEmProcessoComCofreInexistenteRegistraCausa (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	3.895s
```

---

## Prova de mutação

Regra: **o ramo de saída por cofre inválido loga em nível ERROR antes de sair.**

```
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/daemon.go `
  -Anchor 'log.Error("daemon nao pode montar o servico",' `
  -Replacement 'log.Info("daemon nao pode montar o servico",' `
  -Test TestDaemonComCofreInexistenteRegistraCausa -Package ./cmd/gobsidian/
```

Saída:

```
--- FAIL: TestDaemonComCofreInexistenteRegistraCausa (0.03s)
    daemon_log_test.go:49: nenhum log de ERROR antes da saida.
        log foi:
        time=... level=INFO msg="daemon iniciado" vault=...
        time=... level=INFO msg="daemon nao pode montar o servico" vault=... err="raiz do cofre inacessivel \"...\": GetFileAttributesEx ...: The system cannot find the file specified."
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	4.101s
----------------------------------------------------------------------
[OK] cmd/gobsidian/daemon.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

`EXIT=0` = o teste reprovou sob mutação, que é o resultado desejado (o código de saída do `mutate.ps1` é invertido de propósito). A saída mutada também mostra, colada, **a mensagem que existia e nunca chegava a ninguém**: `raiz do cofre inacessivel ... The system cannot find the file specified.`

---

## O que entrou

**(a)** `runDaemon` (`cmd/gobsidian/daemon.go`) loga a causa antes de cada `return err` — no ramo de `ipc.Listen` e no de `construirServico`. O segundo é o que matou os dois daemons de 24/08: é onde `vault.New` recusa o cofre.

**(b)** `serveEmProcesso` (`cmd/gobsidian/serve.go`) recebe o mesmo tratamento. Vale mesmo tendo ali um stderr com leitor: o fallback em processo é obrigatório quando o daemon não sobe, e se os dois caminhos de boot morrerem calados um cofre mal configurado não produz mensagem acionável em lugar nenhum.

**(c)** `errnoDe` em `cmd/gobsidian/ponte.go`, consumida nos três pontos de queda. Medido em 2026-08-26 com `net.Dial("unix", ...)` no Windows:

| Estado do caminho | errno |
|---|---|
| arquivo comum · socket órfão · caminho inexistente | `10061` ECONNREFUSED |
| **diretório** | **`10022`** EINVAL |

O comentário da função registra explicitamente que o número **não** deve ser usado para decidir se há daemon vivo — errnos diferentes descrevem o mesmo estado e o mesmo errno descreve estados diferentes.

---

## Verificações

1. **Todos os `return` de erro do `runDaemon` logam** — conferido nos dois ramos, antes e depois da linha `"daemon iniciado"`.
2. **Nenhuma validação nova foi acrescentada.** `vault.New` (`internal/vault/vault.go:90-95`) já recusava caminho ausente e não-diretório. Esta tarefa faz o erro existente chegar ao log. *(Esta verificação derrubou a Task 127, que planejava reimplementar a validação — ver o plano.)*
3. **Nenhum log novo em stdout.** stdout continua pertencendo ao JSON-RPC.
4. `pwsh -File scripts/verify.ps1`: **14 de 14 [OK]**, `[OK] Bateria completa. Pode commitar.`

---

## O que ficou de fora

Falhas **anteriores** ao logger existir — `config.Load` e `novoLoggerDoDaemon` — continuam subindo para o cobra, cujo stderr não tem leitor num processo detachado. É a forma que casa com o sintoma "log ausente" e a Task 126 passou a nomeá-la explicitamente na mensagem de prazo estourado, mas **não foi corrigida na origem**. Exige decidir onde um daemon escreve quando ainda não sabe onde escrever.

## `git status --porcelain`

Nenhum arquivo de terceiros tocado: `test-vault/`, `.claude/skills/troglodita*/`, `Resume-Claude.ps1` e os `task-N-base.txt` ficaram fora do commit de propósito.
