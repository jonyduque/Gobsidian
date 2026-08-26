# Task 130 — A7: o atalho do `Apply` espelha a condição da `Reconcile`

**Status:** DONE
**Commit:** `539d25c` — `fix(watcher,service): close A7, A1, B5, B17 and B6`

Uma linha de produção. Primeira da ordem de execução de propósito: sozinha
corta o modo de falha mais provável num cofre OneDrive.

---

## O mecanismo, reconferido no código

A assimetria é literal. `internal/watcher/overflow.go:58` já exigia a condição
certa:

```go
if n, ok := idx.Get(e.Path); ok && (inv == nil || inv.HasDoc(string(e.Path))) {
```

e `internal/watcher/apply.go:85` não a espelhava:

```go
if n, ok := idx.Get(path); ok {
```

**Duas cópias de uma regra, e a errada estava no caminho mais usado.**

Um único `searchInv.Update` falho — que só produz `log.Warn`, logo abaixo no
mesmo laço — deixa os metadados em dia e a posting ausente. A partir daí, todo
evento com mtime e tamanho iguais cai no `continue`, e o índice de busca nunca
se recompõe. O OneDrive re-emite evento de arquivo intocado como rotina, então
o gatilho não é raro.

---

## Evidência de TDD

### RED

```
$ go test -run 'TestAtalhoDoApply' -v ./internal/watcher/

=== RUN   TestAtalhoDoApplyConsultaOIndiceDeBusca
    atalho_busca_test.go:86: o indice de busca NAO se recompos: o atalho de
    mtime/tamanho pulou o evento sem consultar HasDoc (skipped=1, processed=1)
--- FAIL: TestAtalhoDoApplyConsultaOIndiceDeBusca (5.02s)
=== RUN   TestAtalhoDoApplyAindaPulaQuandoTudoEstaEmDia
--- PASS: TestAtalhoDoApplyAindaPulaQuandoTudoEstaEmDia (0.03s)
```

`skipped=1` é o mecanismo exato: o evento chegou e foi pulado.

### GREEN

```
--- PASS: TestAtalhoDoApplyConsultaOIndiceDeBusca (0.02s)
--- PASS: TestAtalhoDoApplyAindaPulaQuandoTudoEstaEmDia (0.02s)
ok  	github.com/jonyd/gobsidian/internal/watcher	2.014s
```

---

## Prova de mutação

A mutação restaura a condição antiga **literal**, e não desliga um condicional:

```
pwsh -File scripts/mutate.ps1 -Path internal/watcher/apply.go `
  -Anchor 'if n, ok := idx.Get(path); ok && (searchInv == nil || searchInv.HasDoc(string(path))) {' `
  -Replacement 'if n, ok := idx.Get(path); ok {' `
  -Test TestAtalhoDoApplyConsultaOIndiceDeBusca -Package ./internal/watcher/
```

```
FAIL
----------------------------------------------------------------------
[OK] internal/watcher/apply.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

---

## Verificações

1. **Contrapeso obrigatório.** `TestAtalhoDoApplyAindaPulaQuandoTudoEstaEmDia`
   existe porque a correção poderia ter sido "nunca pular", e o atalho — que
   existe para não reindexar a cada evento espúrio do OneDrive — deixaria de
   funcionar em silêncio, trocando um defeito por um custo. O teste afirma que
   `skipped` continua subindo quando tudo está em dia.
2. **`HasDoc` não lê disco**: o custo do atalho continua sendo memória.
3. **Guarda de cenário no RED**: o teste confirma que a nota chegou ao índice de
   busca ANTES de remover a posting. Sem isso, a asserção final poderia passar
   num estado em que nada nunca esteve indexado.
4. `pwsh -File scripts/verify.ps1`: **14 de 14 [OK]**.

---

## O que ficou de fora

**A condição não foi extraída para uma função compartilhada** entre `Apply` e
`Reconcile`. Seria mais fiel a "uma conta por regra", mas os dois laços têm
formas diferentes — um itera evento, o outro entrada de varredura — e forçar uma
assinatura comum acrescentaria mais do que remove. Decisão registrada aqui para
não parecer esquecimento.
