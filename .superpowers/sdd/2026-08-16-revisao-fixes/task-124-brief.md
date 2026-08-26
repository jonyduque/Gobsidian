# Task 124 — observabilidade do daemon: quem morre diz por quê

**Tier: modelo barato.** Sem decisão de projeto; o desenho está aqui inteiro.

#### Onde encaixa
Primeira da Fase 4, **antes** da 125 e da 126. Vem primeiro de propósito: a 126
mexe na lógica de partida, e mexer nela enquanto as falhas são mudas repete a
investigação que originou este lote.

#### A evidência que originou esta tarefa (2026-08-26, cofres reais do dono)
Dois daemons na máquina do dono morreram deixando **uma linha** no log e nada
mais — `%LOCALAPPDATA%\gobsidian\run\<VaultKey>.sock.log`:

```
time=2026-08-24T12:56:46.418-03:00 level=INFO msg="daemon iniciado" vault="C:\\Users\\jonyd\\Obsidian\\Revis<U+FFFD>o" socket=...7a43b2b161338f9a.sock read_only=false ociosidade_s=900
(fim do arquivo)
```

O outro, `4568ecbd07c39faa.sock.log`, repetiu a mesma linha duas vezes (24/08
12:56 e 19:15) para `C:\Users\jonyd\Obsidian\Jurisprudencia` — caminho **sem
acento, que não existe no disco**; só `Jurisprudência` existe. O daemon subiu,
falhou ao indexar um caminho inexistente e saiu sem registrar nada.

Do lado do host o sintoma não ajuda em nada — `mcp-server-gobsidian-jurisprudencia.log`:

```
[gobsidian-jurisprudencia] [info] Server transport closed unexpectedly, this is
likely due to the process exiting early.
[gobsidian-jurisprudencia] [error] Couldn't start for Cowork and Code sessions.
```

Um cofre inexistente produziu, ao longo de dois dias, **zero** mensagens
acionáveis em três lugares diferentes.

#### O que vincula esta tarefa
- **stdout pertence ao JSON-RPC.** Todo log vai para stderr via `log/slog`. No
  daemon o stderr é redirecionado para `<socket>.log` pelo spawner; é lá que a
  mensagem tem de cair.
- **Não afirme estado que você não verificou.** O relatório desta tarefa cola o
  conteúdo real do arquivo de log produzido pelo teste, não a descrição dele.
- **Mensagem host-facing não leva caminho absoluto** (B9); absoluto só no `slog`.

#### O que entra

**(a) Todo caminho de saída do daemon loga a causa antes de sair.**
Em `internal/daemon/daemon.go` e `cmd/gobsidian/daemon.go`: falha de
`index.Build`, de `ipc.Listen`, de leitura de config, de montagem do serviço —
cada uma emite `log.Error` com a causa e o campo que a identifica, **antes** do
`return`/`os.Exit`, e o processo sai com código diferente de zero.

**(b) Cofre inválido falha alto, nomeando o caminho.**
Caminho inexistente, inacessível, ou que não é diretório: erro na entrada, com o
caminho na mensagem de log. Vale para o daemon **e** para `serveEmProcesso`, que
hoje morre igualmente calado.

**(c) O errno numérico entra no log de dial.**
`cmd/gobsidian/ponte.go`, nos três pontos de queda: além da mensagem do Go,
registrar o número, via `errors.As` para `syscall.Errno`. A prosa do Windows não
distingue casos que se comportam de forma diferente — foi preciso reverter a
mensagem para descobrir que `An invalid argument was supplied` é `10022` e
`actively refused` é `10061`, e essa distinção decide a Task 126.

#### O que prova esta tarefa
RED que falha **hoje**, em `internal/daemon`:

```go
func TestDaemonComCofreInexistenteRegistraCausa(t *testing.T) {
	dir := t.TempDir()
	inexistente := filepath.Join(dir, "cofre-que-nao-existe")
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	err := runDaemon(context.Background(), config.Config{VaultPath: inexistente}, log)
	if err == nil {
		t.Fatal("runDaemon devolveu nil para cofre inexistente")
	}
	saida := buf.String()
	if !strings.Contains(saida, "level=ERROR") {
		t.Errorf("nenhum log de ERROR antes da saida; log foi:\n%s", saida)
	}
	if !strings.Contains(saida, inexistente) {
		t.Errorf("o log nao nomeia o caminho do cofre; log foi:\n%s", saida)
	}
}
```

Provas de mutação exigidas, com a saída de `scripts/mutate.ps1` colada:
1. Remover o `log.Error` do ramo de saída por cofre inválido ⇒ o teste acima
   reprova pelo nome e pela linha.
2. Trocar o `return err` por `return nil` nesse mesmo ramo ⇒ reprova.

#### Verificações
Além dos passos:
1. `runDaemon` (`cmd/gobsidian/daemon.go:131`) hoje devolve erro em
   `ipc.Listen` (`:133-135`) **sem logar**, e a montagem do serviço mais adiante
   devolve o erro de `vault.New` do mesmo jeito. Confira que **todos** os
   `return` de erro anteriores e posteriores ao `log.Info("daemon iniciado")`
   (`:139`) passaram a logar.
2. `vault.New` **já valida** existência e tipo do caminho
   (`internal/vault/vault.go:90-95`). Esta tarefa **não** acrescenta validação —
   ela faz o erro que já existe chegar ao log. Se você se pegar escrevendo
   `os.Stat` novo, parou no lugar errado.
3. Rode o binário contra um caminho inexistente e **leia o arquivo**
   `<socket>.log` que ele produz. Uma linha só significa que a tarefa não está
   pronta.
4. Nenhum log novo em stdout — stdout pertence ao JSON-RPC.

#### Regras de execução
- Gate: `pwsh -File scripts/verify.ps1` verde, com a contagem de passos colada.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean` nem `git reset`.
- Nunca `go mod tidy`.
- `golangci-lint version` conferido antes de confiar num zero — o CI fixa
  `v2.12.2`.
- Se um teste falhar por motivo que este brief não explica, **pare e reporte**
  `BLOCKED`; não ajuste a expectativa para o código passar.

#### Comando de mutação
Copie a âncora **do arquivo**, não de memória — âncora digitada sai `EXIT=2`:

```bash
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/daemon.go `
  -Anchor 'return fmt.Errorf("abrindo socket do daemon: %w", err)' `
  -Replacement 'return nil' `
  -Test TestDaemonComCofreInexistenteRegistraCausa -Package ./cmd/gobsidian/
```

`0` = o teste reprovou sob mutação (é o que se quer). `1` = a regra está escrita
e não verificada. `2` = âncora ambígua ou build quebrado.

**Files:** `cmd/gobsidian/daemon.go`, `cmd/gobsidian/ponte.go`,
`cmd/gobsidian/servico.go`, `internal/daemon/daemon.go`, mais o `_test.go` novo.

#### Contrato de relatório
Status; commit; RED com a saída falhando e GREEN com a saída passando; as duas
provas de mutação com `EXIT=0` e o texto do `mutate.ps1`; `verify.ps1` com a
contagem de passos; e **o conteúdo literal do arquivo de log** que o teste
produziu, para que se veja a mensagem que passou a existir.

---

