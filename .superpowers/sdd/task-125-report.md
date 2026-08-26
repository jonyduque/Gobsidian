# Task 125 — `doctor` diagnostica o daemon

**Status:** DONE_WITH_CONCERNS (uma checagem mudou de lugar durante a execução; ver Verificações)
**Commit:** `499e744` — `fix(daemon): make silent failures diagnosable, and repair scan root handling`

---

## Evidência de TDD

### GREEN (testes novos, `internal/doctor`)

```
$ go test -race -run 'TestClasseDoCaminhoDoSocket|TestVizinhosParecidos|TestCheckRootExistsApontaAGrafia|TestUltimasLinhas' -v ./internal/doctor/

=== RUN   TestClasseDoCaminhoDoSocketDistingueOsEstados
=== RUN   TestClasseDoCaminhoDoSocketDistingueOsEstados/ausente
=== RUN   TestClasseDoCaminhoDoSocketDistingueOsEstados/arquivo_comum
=== RUN   TestClasseDoCaminhoDoSocketDistingueOsEstados/diretorio
=== RUN   TestClasseDoCaminhoDoSocketDistingueOsEstados/socket_real
--- PASS: TestClasseDoCaminhoDoSocketDistingueOsEstados (0.01s)
=== RUN   TestVizinhosParecidosAchaGrafiaComAcento
--- PASS: TestVizinhosParecidosAchaGrafiaComAcento (0.00s)
=== RUN   TestUltimasLinhasDevolveOFimDoArquivo
--- PASS: TestUltimasLinhasDevolveOFimDoArquivo (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/doctor	1.976s
```

---

## Prova de mutação

Regra: **diretório e arquivo comum no caminho do socket produzem classificações DIFERENTES.**

Ela importa porque o erro do dial não distingue os dois — medido, arquivo comum dá `10061` e diretório dá `10022`, e foi o `10022` que apareceu em campo sem reprodutor conhecido.

```
pwsh -File scripts/mutate.ps1 -Path internal/doctor/daemon.go `
  -Anchor 'return "DIRETORIO (nenhum daemon consegue usar este caminho)"' `
  -Replacement 'return fmt.Sprintf("arquivo comum de %d bytes (residuo; nenhum daemon escuta aqui)", fi.Size())' `
  -Test TestClasseDoCaminhoDoSocketDistingueOsEstados -Package ./internal/doctor/
```

Saída:

```
    --- FAIL: TestClasseDoCaminhoDoSocketDistingueOsEstados/diretorio (0.00s)
        daemon_test.go:64: classe de diretorio = "arquivo comum de 0 bytes (residuo; nenhum daemon escuta aqui)", queria conter "DIRETORIO"
    daemon_test.go:73: diretorio e arquivo comum produziram a MESMA classe; a distincao sumiu
FAIL
FAIL	github.com/jonyd/gobsidian/internal/doctor	1.408s
----------------------------------------------------------------------
[OK] internal/doctor/daemon.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

---

## Saída real do comando, nos dois estados

### Cofre com caminho errado (o defeito de campo)

```
$ ./bin/gobsidian.exe doctor --vault "C:/Users/jonyd/Obsidian/Jurisprudencia"

[!] raiz do cofre existe
     "C:\\Users\\jonyd\\Obsidian\\Jurisprudencia": GetFileAttributesEx C:\Users\jonyd\Obsidian\Jurisprudencia: The system cannot find the file specified.
     existe(m) ao lado, com grafia diferente: Jurisprudência
[!] Ha falhas bloqueantes acima
```

A linha `existe(m) ao lado, com grafia diferente` é a que teria encerrado, em um comando, um diagnóstico que custou quatro camadas de investigação manual e dois dias de servidor quebrado.

### Cofre saudável

```
$ ./bin/gobsidian.exe doctor --vault "C:/Users/jonyd/Obsidian/Oral"

[OK] caminho do socket do daemon
     C:\Users\jonyd\AppData\Local\gobsidian\run\c509b8ca23f51383.sock -- ausente
[OK] daemon respondendo
     nenhum daemon rodando (a ponte servira em processo)
[OK] log do daemon
     C:\...\c509b8ca23f51383.sock.log (2444 bytes, ultima escrita ha 2h44m0s)
      | time=2026-08-25T22:50:30.353-03:00 level=INFO msg="indice de busca pronto" origem=cache notas=78 duracao_ms=34
      | time=2026-08-26T05:38:03.216-03:00 level=INFO msg="daemon iniciado" vault=... read_only=true ociosidade_s=900
      | time=2026-08-26T05:38:03.282-03:00 level=INFO msg="servidor pronto" vault=... notes=78 index_ms=32 index_origin=cache
[OK] locks de daemon orfaos
     nenhum
[OK] Ambiente apto
```

---

## Verificações

1. **A checagem de grafia mudou de lugar durante a execução, e isso é um achado.** Ela começou como `checkGrafiaDoCofre`, uma verificação própria posterior às demais — e a primeira execução real mostrou que **ela nunca rodava**: `checkRootExists` é *halting*, aborta as seguintes, e o caso para o qual a checagem existia é exatamente o que dispara o halting. Foi dobrada para dentro de `checkRootExists` (`internal/doctor/checks.go`). O teste foi reapontado para a função que de fato roda.
   Lição registrada: **verificação posterior a um check halting não cobre o caso que causa o halting.**
2. **`socket_path` não abre o socket.** Usa `os.Lstat` — abrir é justamente a operação cujo resultado se está diagnosticando.
3. **`doctor > relatorio.txt` sai sem códigos de cor** — decisão por destino, via `internal/console`, preservada.
4. **RNF-30 verde.** O teste usa `net.Listen("unix", ...)` com a constante literal, que é o que `tools/netcheck` permite: `pwsh -File scripts/check_net.ps1` → `EXIT=0`, *"Nenhum pacote de internal/ ou cmd/ importa net/* ou abre socket que saia da maquina (verificado via netcheck vettool em windows, linux, darwin)"*.
5. **`daemon.PIDVivo`** foi exposto como invólucro fino sobre a implementação por plataforma, em vez de o `doctor` reimplementar a liveness — a armadilha do Windows (PID e creation time consultáveis depois da morte) já está paga num lugar só.
6. `pwsh -File scripts/verify.ps1`: **14 de 14 [OK]**.

---

## O que ficou de fora

**A checagem `daemon_vivo` não é testada por teste automatizado** — exercitá-la exigiria subir um daemon de verdade dentro do teste unitário, e a evidência que tenho dela é a saída manual acima, nos dois estados. As outras quatro têm teste.

`golangci-lint` acusou dois achados meus, ambos corrigidos antes do commit: `package-comments` (comentário de arquivo colado ao `package`) e `redefines-builtin-id` (variável chamada `real`).
