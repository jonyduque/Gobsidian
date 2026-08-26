# Task 128 — C1 + C3: uma guarda só, e o boot passando por ela

**Status:** DONE
**Commit:** `e345578` — `fix(vault,search): close the three critical file-access findings`

Os dois críticos se consertam juntos porque o remédio é o mesmo: fazer
`Inverted.Update` ser o **único** caminho de indexação a partir de arquivo, com
as duas guardas dentro dela.

---

## C3 — anexo era lido inteiro

### RED

```
$ go test -run 'TestUpdateNaoLe|TestUpdateLeNota' ./internal/search/

--- FAIL: TestUpdateNaoLeAnexo (0.01s)
    anexo_nao_lido_test.go:59: o anexo foi LIDO: o termo-sonda "xyzzyplugh" entrou no indice (1 postings)
    anexo_nao_lido_test.go:63: DocLength do anexo = 4, queria 0: bytes de anexo entraram no divisor do BM25
--- FAIL: TestUpdateNaoLeArquivoExcluido (0.01s)
    anexo_nao_lido_test.go:91: arquivo excluido foi LIDO: "xyzzyplugh" entrou no indice
FAIL
FAIL	github.com/jonyd/gobsidian/internal/search	0.878s
```

**`DocLength do anexo = 4`** é o número que muda a natureza do achado. A
auditoria classificou C3 como "Bug/Falha-de-contrato/Performance"; o divisor da
normalização por tamanho do BM25 recebendo bytes de anexo é **ranking errado**,
não lentidão.

O anexo do teste tem **texto ASCII legível**, não bytes binários, de propósito:
um `.png` de bytes aleatórios passaria a asserção por acidente, porque não
produziria token nenhum de qualquer forma — o teste estaria medindo o conteúdo
em vez da guarda.

### GREEN

```
--- PASS: TestUpdateNaoLeAnexo (0.00s)
--- PASS: TestUpdateNaoLeArquivoExcluido (0.00s)
--- PASS: TestUpdateLeNota (0.01s)
ok  	github.com/jonyd/gobsidian/internal/search	0.844s
```

### Prova de mutação

```
pwsh -File scripts/mutate.ps1 -Path internal/search/inverted.go `
  -Anchor 'if vault.Classify(path) != vault.ClassNote {' `
  -Replacement 'if false {' `
  -Test TestUpdateNaoLeAnexo -Package ./internal/search/
```

```
FAIL
----------------------------------------------------------------------
[OK] internal/search/inverted.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

---

## C1 — o boot contornava a guarda

### RED, contra o boot DE PRODUÇÃO

```
$ go test -run TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem ./cmd/gobsidian/

--- FAIL: TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem (0.04s)
    boot_indice_busca_windows_test.go:94: o boot ABRIU arquivo que nao devia: "sesquipedaliano" entrou no indice via [na-nuvem.md]
FAIL
FAIL	github.com/jonyd/gobsidian/cmd/gobsidian	2.645s
```

O teste chama `buildInvertedIndex` diretamente. **Isto é o ponto da tarefa.** O
que "provava" a regra antes era um dublê — `construirComoOBoot`, em
`internal/search/cloudonly_update_windows_test.go:132` — que chamava
`Inverted.Update` num laço e afirmava em comentário ser *"exatamente como
buildInvertedIndex faz"*. Não era: produção fazia `v.ReadAll` + `inv.Add`.

O comentário do dublê passou a ser verdadeiro só depois desta correção.

### GREEN

```
--- PASS: TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem (0.06s)
ok  	github.com/jonyd/gobsidian/cmd/gobsidian	4.146s
```

### Prova de mutação

A mutação **restaura o corpo antigo literal**, em vez de desligar um
condicional — é a forma mais forte disponível aqui, porque prova que o teste
pega o defeito original e não uma aproximação dele:

```
pwsh -File scripts/mutate.ps1 -Path cmd/gobsidian/serve.go `
  -Anchor 'if err := inv.Update(ctx, v, p); err != nil {' `
  -Replacement 'if err := func() error { d, e := v.ReadAll(ctx, p); if e != nil { return e }; b, _ := vault.StripBOM(d); inv.Add(string(p), search.Analyze(string(b))); return nil }(); err != nil {' `
  -Test TestBuildInvertedIndexNaoAbrePlaceholderDeNuvem -Package ./cmd/gobsidian/
```

```
FAIL
----------------------------------------------------------------------
[OK] cmd/gobsidian/serve.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

---

## Verificações

1. **Contrapeso obrigatório em ambos.** `TestUpdateLeNota` e a asserção 3 do
   teste de boot existem porque uma guarda larga demais — "não leia nada" —
   passaria em todos os testes de recusa e **desligaria a busca inteira em
   silêncio**. Nota tem de continuar sendo lida; `DocLength("hidratada.md")`
   tem de ser diferente de zero.
2. **Cobertura entra, conteúdo não.** Os dois casos guardados chamam
   `ix.Add(path, nil)`, não `return` seco. Fora de `docLengths` a entrada não
   conta em `DocCount`, o cabeçalho do cache declara menos entradas do que o
   índice de metadados enxerga, e **todo** boot conclui "cache parcial" e
   regrava o cache inteiro. É a armadilha irmã da nota sem token nenhum, e as
   duas asserções de `HasDoc` a cobrem.
3. **`TestUpdateNaoLeArquivoExcluido`** existe para que a guarda não pudesse ter
   sido escrita como "se não for nota, mas for anexo", deixando ruído e
   diretório excluído passando.
4. **O comentário falso de `inverted.go` foi corrigido**, e agora descreve os
   chamadores reais.
5. `pwsh -File scripts/verify.ps1`: **14 de 14 [OK]**.

---

## O que ficou de fora

- **O atalho de mtime/tamanho da reconciliação (`overflow.go:58`) não foi
  tocado.** Ele consulta `idx.Get`, que resolve só notas, então todo anexo
  continua falhando no atalho. A diferença é que agora falhar no atalho custa
  uma classificação e um `Add(path, nil)`, não uma leitura — o dano principal
  saiu, o trabalho redundante ficou. É o complemento 1.4 do plano, ainda aberto.
- **O dublê `construirComoOBoot` continua existindo.** O comentário dele deixou
  de mentir, mas quem prova o boot agora é o teste novo, em `cmd/gobsidian`.
