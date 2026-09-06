# Task 153 — Relatório

## Status
DONE

## Commit
`5360a3b` fix(doctor): list the listen lock too, deriving both lock suffixes and the log path from daemon [sem-doc]

## Evidência de TDD

### `internal/daemon` — `TestEhArquivoDeTravaCobreAsDuasTravas`

RED (antes de `EhArquivoDeTrava`/`sufixoTrava`/`sufixoTravaDeEscuta` existirem):

```
$ go test ./internal/daemon/ -run TestEhArquivoDeTravaCobreAsDuasTravas -v
# github.com/jonyd/gobsidian/internal/daemon [github.com/jonyd/gobsidian/internal/daemon.test]
internal\daemon\lock_sufixos_test.go:16:10: undefined: sufixoTrava
internal\daemon\lock_sufixos_test.go:17:10: undefined: sufixoTravaDeEscuta
internal\daemon\lock_sufixos_test.go:23:13: undefined: EhArquivoDeTrava
internal\daemon\lock_sufixos_test.go:32:6: undefined: EhArquivoDeTrava
FAIL	github.com/jonyd/gobsidian/internal/daemon [build failed]
FAIL
```

GREEN (depois de `lock.go` ganhar as constantes e a função):

```
$ go test ./internal/daemon/ -run TestEhArquivoDeTravaCobreAsDuasTravas -v
=== RUN   TestEhArquivoDeTravaCobreAsDuasTravas
--- PASS: TestEhArquivoDeTravaCobreAsDuasTravas (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/daemon	1.780s
```

### `internal/doctor` — `TestCheckLocksDeDaemonEnxergaListenLock` e `TestCheckDaemonLogUsaACaminhoDoLogDoDaemon`

RED (antes de `doctor/daemon.go` usar `daemon.EhArquivoDeTrava`/`daemon.CaminhoDoLog`) — rodado com `daemon.EhArquivoDeTrava` e `daemon.CaminhoDoLog` já existindo em `daemon` (do passo anterior), mas com `doctor/daemon.go` ainda no filtro antigo `.sock.lock`:

```
$ go test ./internal/doctor/ -run 'TestCheckLocksDeDaemonEnxergaListenLock|TestCheckDaemonLogUsaACaminhoDoLogDoDaemon' -v
=== RUN   TestCheckLocksDeDaemonEnxergaListenLock
    daemon_test.go:131: com a trava de escuta tomada, doctor disse: "nenhuma trava em uso"
--- FAIL: TestCheckLocksDeDaemonEnxergaListenLock (0.46s)
=== RUN   TestCheckDaemonLogUsaACaminhoDoLogDoDaemon
--- PASS: TestCheckDaemonLogUsaACaminhoDoLogDoDaemon (0.02s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/doctor	2.021s
FAIL
```

`TestCheckDaemonLogUsaACaminhoDoLogDoDaemon` já passava antes do fix, porque a
conta antiga (`sock + ".log"`) e a nova (`daemon.CaminhoDoLog`) calculam o
MESMO valor hoje — o teste prova que a troca de conta não mudou o
comportamento, não reproduz um defeito visível. O defeito de fato (trava de
escuta invisível) está só no primeiro teste, e esse é o que estava RED.

GREEN (depois de `checkLocksDeDaemon` usar `daemon.EhArquivoDeTrava` e
`checkDaemonLog` usar `daemon.CaminhoDoLog`):

```
$ go test ./internal/daemon ./internal/doctor -run 'EhArquivoDeTrava|ListenLock|CaminhoDoLog' -v
=== RUN   TestEhArquivoDeTravaCobreAsDuasTravas
--- PASS: TestEhArquivoDeTravaCobreAsDuasTravas (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/daemon	1.832s
=== RUN   TestCheckLocksDeDaemonEnxergaListenLock
--- PASS: TestCheckLocksDeDaemonEnxergaListenLock (1.54s)
=== RUN   TestCheckDaemonLogUsaACaminhoDoLogDoDaemon
--- PASS: TestCheckDaemonLogUsaACaminhoDoLogDoDaemon (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/doctor	3.264s
```

## Prova de mutação

Regra: `checkLocksDeDaemon` reconhece a trava de escuta via
`daemon.EhArquivoDeTrava`, não mais por `strings.HasSuffix(e.Name(),
".sock.lock")`.

Comando (variante do brief, porque `strings` não é mais importado em
`doctor/daemon.go` depois do fix — a substituição literal `".sock.lock"` do
brief causaria `EXIT=2` de build; usei a variante alternativa que o próprio
brief já previa para esse caso):

```
$ pwsh -File scripts/mutate.ps1 -Path internal/doctor/daemon.go `
    -Anchor '!daemon.EhArquivoDeTrava(e.Name())' `
    -Replacement '!(len(e.Name()) > 10 && e.Name()[len(e.Name())-10:] == ".sock.lock")' `
    -Test TestCheckLocksDeDaemonEnxergaListenLock -Package ./internal/doctor/
```

Saída:

```
Carregado em 719ms
[...] Mutando internal/doctor/daemon.go
      - !daemon.EhArquivoDeTrava(e.Name())
      + !(len(e.Name()) > 10 && e.Name()[len(e.Name())-10:] == ".sock.lock")

[...] go test -race -run TestCheckLocksDeDaemonEnxergaListenLock ./internal/doctor/
----------------------------------------------------------------------
--- FAIL: TestCheckLocksDeDaemonEnxergaListenLock (0.48s)
    daemon_test.go:131: com a trava de escuta tomada, doctor disse: "nenhuma trava em uso"
FAIL
FAIL	github.com/jonyd/gobsidian/internal/doctor	2.360s
FAIL
----------------------------------------------------------------------
[OK] internal/doctor/daemon.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

Teste que reprova: `TestCheckLocksDeDaemonEnxergaListenLock`, linha
`daemon_test.go:131` (a asserção `!strings.Contains(r.Detail, "listen.lock")`).

Restauro: `mutate.ps1` confirmou SHA-256 idêntico antes/depois. Confirmação
independente:

```
$ git diff --stat internal/doctor/daemon.go
 internal/doctor/daemon.go | 5 ++---
 1 file changed, 2 insertions(+), 3 deletions(-)
```

(Esse diff é o fix real contra HEAD anterior ao commit, não resíduo da
mutação — a mutação em si não deixou marca, confirmada pelo hash do
`mutate.ps1`.)

## As verificações do brief

1. **Nada ficou em `%LOCALAPPDATA%\gobsidian\run\` com o nome do cofre de
   teste.** O diretório de runtime é indexado por `config.VaultKey` (hash),
   não por nome — não há como a lista mostrar um nome de cofre de teste
   literal. `ls` do diretório mostra centenas de arquivos `.sock.lock` /
   `.sock.listen.lock` de hashes variados: isso é esperado e documentado em
   `ARMADILHAS.md` ("Não remova o arquivo de trava" — os `.lock` persistem
   por design, o kernel libera a trava, não a ausência do arquivo). O que
   este teste especificamente precisa limpar é o `.sock.log` que ele
   escreveu — `t.Cleanup(func() { _ = os.Remove(esperado) })` — e o `ls`
   pós-execução não mostra nenhum `.sock.log` com timestamp da janela de
   execução dos testes (20:58–20:59); os únicos `.sock.log` na listagem são
   de execuções anteriores (Sep 2 e Sep 3 04:00), não desta rodada.
2. `grep -rn '".lock"\|".listen.lock"\|".log"' internal/daemon internal/doctor`:
   ```
   internal/daemon/lock.go:115:	sufixoTrava         = ".lock"
   internal/daemon/lock.go:116:	sufixoTravaDeEscuta = ".listen.lock"
   internal/daemon/lock.go:122:	return strings.HasSuffix(nome, sufixoTrava) // ".listen.lock" tambem termina em ".lock"
   internal/daemon/lock_sufixos_test.go:19:		sock + ".log":              false,
   internal/daemon/log.go:25:	return sock + ".log", nil
   ```
   As únicas ocorrências de produção são as constantes em `lock.go` e a
   constante em `log.go` (`CaminhoDoLog`). A linha em
   `lock_sufixos_test.go:19` é dado de teste (o caso negativo `sock + ".log":
   false` do próprio brief), não uma segunda conta.
3. `go list -f '{{.Imports}}' ./internal/doctor` ainda inclui
   `github.com/jonyd/gobsidian/internal/ipc` — `checkSocketPath` e
   `checkDaemonVivo` continuam usando `ipc.SocketPath`/`ipc.DialAndHandshake`.
   A aresta `doctor → ipc` **não** sumiu; por isso `CLAUDE.md` não foi
   tocado, conforme a decisão do orquestrador.

## `verify.ps1`

```
[...] 13. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```
13 etapas, todas `[OK]`, rodado antes do commit.

## O que ficou de fora

- `CLAUDE.md` não foi editado: a aresta `doctor → ipc` continua existindo
  (verificação 3 acima), então o grafo do documento não mudou.
- Nenhuma atualização de `docs/ARMADILHAS.md` — o commit foi marcado
  `[sem-doc]` porque a regra "uma conta por regra" que este fix reforça já
  está documentada ali (a consolidação de `daemon.CaminhoDoLog` em
  2026-08-26); nada de novo em contrato, arquitetura ou requisito mudou, e o
  brief não listava um arquivo de `docs/` para editar. Isso disparou o hook
  de pré-commit ("2 arquivo(s) .go de produção em stage e NENHUM arquivo de
  documentação"), resolvido com o marcador `[sem-doc]` e a justificativa no
  corpo do commit.
- O brief pedia o comando de mutação literal com `-Replacement
  '!strings.HasSuffix(e.Name(), ".sock.lock")'` primeiro; ele teria dado
  `EXIT=2` de build porque `strings` deixou de ser importado em
  `doctor/daemon.go` depois do fix (só sobrou em uso indireto por outras
  funções do arquivo, então o import continua presente — mas o próprio brief
  já previa essa ambiguidade e ofereceu a variante alternativa, que é a que
  usei diretamente para não gastar uma rodada em EXIT=2).

## `git status --porcelain`

Antes do commit (arquivos do usuário, nenhum tocado):
```
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
 M "test-vault/test vault/.obsidian/community-plugins.json"
 D "test-vault/test vault/.obsidian/plugins/parity-dumper/main.js"
 D "test-vault/test vault/.obsidian/plugins/parity-dumper/manifest.json"
 M "test-vault/test vault/.obsidian/workspace.json"
?? .claude/skills/troglodita-commit/
?? .claude/skills/troglodita-help/
?? .claude/skills/troglodita-review/
?? .claude/skills/troglodita/
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/... (diffs e briefs de outras tasks)
?? Resume-Claude.ps1
?? "test-vault/test vault/..." (vários)
```

Depois do commit de Task 153: os quatro arquivos do brief (`lock.go`,
`lock_sufixos_test.go`, `doctor/daemon.go`, `doctor/daemon_test.go`) saíram
do status; todo o resto acima permanece idêntico — nenhum arquivo do usuário
tocado.
