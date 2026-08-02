# Relatório Task 76: as duas lacunas de teste que atravessaram todos os marcos

**Data:** 2026-08-02
**Base:** `341b124`
**Commits:** `343cef6` (cenários, teste de symlink, CI), `ce5dddc` (timeout do
harness por profundidade de cadeia).

---

## Evidência de TDD

O entregável é um cenário de teste, então o ciclo é sobre ele: primeiro provar
que o cenário reprova quando o mecanismo não está lá (RED), depois que passa
quando está (GREEN).

### RED

Cada cenário foi rodado com o mecanismo que ele nomeia REMOVIDO do
`internal/lifecycle`. As duas saídas estão em "Prova por remoção", abaixo: o
`parent-death` deixou 3 de 3 órfãos sem `parent-watch`, e o `signal` deixou 3 de
3 ciclos sem `reason=` sem `watchSignals`.

O gate de `reason=` também reprovou de verdade durante o desenvolvimento, antes
de o cenário de sinal ficar correto:

```
[!] Ciclo 3: reason=stdin-eof, esperado 'signal' no cenario 'signal'
[!] FALHA: 1 de 1 ciclo(s) encerraram por um motivo diferente de 'signal'
```

### GREEN

Com os mecanismos restaurados, 100 ciclos de cada cenário, local e no CI, com o
motivo certo e zero órfãos. Números em "Distribuição de `reason=`".

---

## Lacuna 1 — vigília do pai e sinais nunca verificados ponta a ponta

O harness tinha **um** cenário. O host segurava a ponta de escrita do pipe de
stdin, então matá-lo produzia EOF, e o EOF sempre vencia a corrida: 100/100 nas
duas rodadas anteriores. Os outros dois mecanismos eram código não exercitado.

Agora são três cenários, um por mecanismo, e **cada um desconecta os outros
dois**:

| Cenário | O que morre | stdin do servidor | Único mecanismo possível |
|---|---|---|---|
| `stdin-eof` | o host | pipe cuja ponta de escrita o host segura | EOF |
| `parent-death` | o host intermediário | pipe cuja ponta de escrita o **keeper** segura, e o keeper sobrevive | vigília do pai |
| `signal` | **nada** | `CONIN$`, que nunca chega a EOF | `CTRL_BREAK` |

**Como o `parent-death` mantém o stdin aberto.** A cadeia é keeper → host →
servidor. O keeper lança o host com `RedirectStandardInput`, ficando com a ponta
de escrita; o host lança o servidor **sem** redirecionar, e no Windows o filho
herda o handle de stdin do pai. Matar o host fecha só a cópia dele. O servidor
tem a própria, e o keeper continua segurando a escrita — não há EOF.

**Como o `signal` desliga os dois.** O host fica vivo, então a vigília do pai
não dispara. O stdin do servidor é `CONIN$`, o buffer de entrada do console, que
não fecha. O servidor é criado com `CREATE_NEW_PROCESS_GROUP` via `CreateProcessW`
— `ProcessStartInfo` não expõe `dwCreationFlags` — para que
`GenerateConsoleCtrlEvent` alcance o grupo dele sozinho e **não** mate o host,
que devolveria a corrida aos outros mecanismos.

### O gate de `reason=`

O harness agora reprova quando o `reason=` registrado não é o do mecanismo que o
cenário nomeia. Sem isso, o `parent-death` caindo em `stdin-eof` pareceria verde
— e é assim que a lacuna sobreviveu: o harness contava "encerrou", nunca
"encerrou por qual mecanismo".

Ele funcionou durante o próprio desenvolvimento. A primeira tentativa do cenário
de sinal deu:

```
[!] Ciclo 3: reason=stdin-eof, esperado 'signal' no cenario 'signal'
[!] FALHA: 1 de 1 ciclo(s) encerraram por um motivo diferente de 'signal'
```

O servidor herdava o stdin do harness, que roda sem entrada interativa, e via EOF
antes de qualquer sinal. Sem o gate, esse ciclo teria sido contado como sucesso.

### Prova por remoção — vigília do pai

```
[OK] vigilia do pai removida
[...] 3 ciclos de encerramento abrupto (host com pipe real em stdin)
[!] Ciclo 1: PID 26244 sobreviveu
[!] Ciclo 2: PID 30156 sobreviveu
[!] Ciclo 3: PID 34020 sobreviveu
[!] FALHA: 3 de 3 ciclo(s) encerraram sem registrar 'reason=' - nenhum mecanismo de encerramento disparou, e zero orfaos nao prova nada nesse caso
[!] FALHA: 3 orfao(s) em 3 ciclos
```

**E, com a MESMA remoção, o cenário antigo continuou verde:**

```
[...] 3 ciclos de encerramento abrupto (host com pipe real em stdin)
[i] motivos observados nos logs de debug:
    stdin-eof: 3x
[OK] Nenhum orfao em 3 ciclos
```

Este par é a medida exata da lacuna: a vigília do pai podia ser **removida
inteira** e o gate de órfãos passava.

### Prova por remoção — sinais

```
[OK] vigia de sinais removido
[...] 3 ciclos de encerramento abrupto (host com pipe real em stdin)
[!] FALHA: 3 de 3 ciclo(s) encerraram sem registrar 'reason=' - nenhum mecanismo de encerramento disparou, e zero orfaos nao prova nada nesse caso
```

Repare que **não houve órfão**: o processo morreu mesmo assim, pelo handler
padrão do Windows para `CTRL_BREAK`. Sem o gate de `reason=`, esta rodada seria
verde com o tratamento de sinal do lifecycle removido — que é literalmente o
defeito achado na revisão do M0 ("servidor morrendo sozinho dá rodada verde sem
mecanismo nenhum ter disparado"). O gate o bloqueia.

### Distribuição de `reason=` em 100 ciclos por cenário

Máquina local (maquina de referencia, 12 núcleos, Windows 11):

```
stdin-eof     -> stdin-eof: 100x     [OK] Nenhum orfao em 100 ciclos   EXIT=0
parent-death  -> parent-gone: 100x   [OK] Nenhum orfao em 100 ciclos   EXIT=0
signal        -> signal: 100x        [OK] Nenhum orfao em 100 ciclos   EXIT=0
```

E no CI (`windows-latest`, run `30768734765`, job `orphans` verde em 49m8s), com
os três cenários no `ci.yml`:

```
    stdin-eof: 100x
[OK] Nenhum orfao em 100 ciclos
    parent-gone: 100x
[OK] Nenhum orfao em 100 ciclos
    signal: 100x
[OK] Nenhum orfao em 100 ciclos
```

O cenário `signal` era a dúvida — ele abre `CONIN$`, e um runner sem console
faria `CreateFileW` falhar. O runner tem console, e ele roda. **O custo é real e fica registrado:** o job
`orphans` passou de 15m11s (run 30761531221, um cenário) para 49m8s (run
30768734765, três cenários) — três rodadas de 100 ciclos com 8 s de assentamento
cada.

Nenhum cenário produziu um motivo que não fosse o seu. Antes desta tarefa,
`stdin-eof` vencia em 100% de tudo.

### Um achado do CI

Na primeira rodada com os cenários (`30765025161`), o `parent-death` mediu
`parent-gone: 91x`, zero órfãos, e reprovou pelos **9 ciclos que não mediram
nada**: `PID do servidor nao apareceu em 5000ms`. O mecanismo estava certo; o
harness é que estava apertado. `parent-death` e `signal` põem três processos
entre o harness e o servidor em vez de dois, cada nível é um `pwsh` a mais para
subir, e um runner de 2 vCPU não cabe nisso em 5 s. O
padrão passou a ser 5 s para `stdin-eof` e 20 s para as cadeias mais fundas
(`ce5dddc`); um `-PidTimeoutMs` explícito continua vencendo.

---

## Lacuna 2 — RNF-32, links simbólicos

`vault.Walk` não seguir link simbólico valia "por construção" —
`filepath.WalkDir` não segue — e tinha só teste indireto na Task 7. Propriedade
que vale por construção vale até alguém trocar a construção.

`TestWalkNaoSegueSymlink` põe uma nota **fora** da raiz, um link para essa pasta
**dentro** dela, e reprova se a varredura entregar a nota de fora.

**Rodou com privilégio, e passou** (esta máquina cria link simbólico):

```
=== RUN   TestWalkNaoSegueSymlink
--- PASS: TestWalkNaoSegueSymlink (0.01s)
ok  	github.com/jonyd/gobsidian/internal/vault	0.974s
```

**Prova de que a asserção não é vazia.** Mutação que faz a varredura começar um
nível acima, de modo que a nota de fora aparece:

```
[...] Mutando internal/vault/symlink_test.go
      - v, err := vault.New(root)
      + v, err := vault.New(base)

    symlink_test.go:92: Walk atravessou o link simbolico e entregou "fora/vazada.md" — RNF-32 violado: a varredura saiu da raiz do cofre
    symlink_test.go:97: Walk nao entregou dentro.md; entradas vistas: [cofre/dentro.md fora/vazada.md] — o cenario nao exercitou nada e a ausencia de vazada.md nao significa nada
FAIL
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
MUTATE_EXIT=0
```

As **duas** asserções dispararam: a de vazamento e a guarda de cenário.

**Prova do caminho de skip.** Forçando `ERROR_PRIVILEGE_NOT_HELD` (1314), que é o
que o Windows devolve sem elevação e que `os.IsPermission` **não** reconhece:

```
    symlink_test.go:68: RNF-32 NAO VERIFICADO nesta maquina: criacao de link simbolico recusada por PERMISSAO (A required privilege is not held by the client.). No Windows exige privilegio elevado ou Modo de Desenvolvedor. Rode elevado para exercitar este teste.
--- SKIP: TestWalkNaoSegueSymlink (0.01s)
```

A mensagem diz que pulou **por permissão** e que o RNF não foi verificado. Skip
permanente sem essa frase é cobertura fantasma, e este projeto já teve três
testes assim. Qualquer erro que **não** seja de privilégio é `t.Fatalf`, não
skip.

---

## Bateria

```
[OK] Bateria completa. Pode commitar.
VERIFY_EXIT=0
```

---

## O que ficou de fora

- **Nada.** Os três cenários e o teste de RNF-32 foram entregues.
- **RNF-32 sem privilégio não foi observado num processo realmente sem
  privilégio**, só com o erro injetado. Rebaixar o privilégio do processo de
  teste não é coisa que dê para fazer de dentro dele; o que está provado é que o
  ramo existe, é alcançável e traz a mensagem certa.
- **Nada em Unix.** Os três cenários são de Windows: `orphan_host.ps1`,
  `taskkill`, `CreateProcessW`, `CONIN$`. A vigília do pai em Unix compara o
  ppid capturado no startup e tem teste unitário
  (`parent_identity_unix_test.go`), mas não harness ponta a ponta. Fica
  registrado como lacuna nova, menor que a que esta tarefa fechou.
