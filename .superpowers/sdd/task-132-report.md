# Task 132 — A2 + A3 + M7: `Replace` em duas fases

**Status:** DONE
**Commit:** `66fef5c` — `fix(index): rebuild Replace in two phases, closing A2, A3 and M7`

Os três se curam juntos porque são o mesmo desenho visto de ângulos diferentes.

---

## O mecanismo, reconferido em `internal/index/update.go`

```go
func (ix *Index) Replace(ctx, v, path) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()                        // <- lock do inicio ao fim

	chaves := ix.removeContributionsLocked(path)    // <- remove ANTES do I/O
	atomic.AddUint64(&ix.generation, 1)

	abs := v.Abs(path)
	info, err := os.Stat(abs)                   // <- I/O sob lock exclusivo
```

- **A2**: `os.Stat`, `IsCloudOnly` e a leitura inteira aconteciam com o lock
  exclusivo tomado. Todo `Get`/`List`/`Backlinks`/`TotalSize` esperava atrás
  dessa leitura a cada evento do watcher. Num OneDrive hidratando, é espera de
  **rede** disfarçada de disco, e contraria o que `index.go:17` declara.
- **A3**: as contribuições saíam **antes** de o arquivo ser lido. Falhando a
  leitura, a nota ficava fora dos metadados sem republish, enquanto a busca
  mantinha o documento velho — e `service.Search` descarta posting sem metadado,
  então **a nota sumia das respostas** até o próximo evento, reconciliação ou
  boot.

---

## O desenho novo

**Fase 1, sem lock:** `Stat`, classificação, `ReadAll`, parse. Tudo que toca
disco.
**Fase 2, sob lock:** só mutação em memória.

Nada é removido enquanto a leitura não tiver dado certo. **A janela do A3 deixa
de existir por construção, não por tratamento de erro.**

### A janela nova tem política explícita, não comentário

Com o I/O fora do lock, um `Remove` concorrente — que pega o mesmo lock — pode
ter apagado a nota entre a leitura e a publicação. Republicar seria ressuscitar
uma nota deletada, e ela ficaria no índice até o próximo evento.

A fase 2 re-`Stat`a antes de mutar e trata ausência como remoção. Um `Stat` é
barato: **o que custa caro num placeholder de nuvem é a LEITURA**, e essa agora
acontece fora do lock.

---

## Evidência de TDD

### RED

```
$ go test -run 'TestReplaceCom|TestReplacePublica' -v ./internal/index/

=== RUN   TestReplaceComErroDeLeituraNaoDeixaANotaForaDoIndice
    replace_duas_fases_windows_test.go:88: a nota saiu do indice por um erro
    TRANSITORIO de leitura: o arquivo continua no disco, e o indice agora
    discorda dele
--- FAIL
=== RUN   TestReplaceComNotaRemovidaEntreOEventoEALeitura
--- PASS
=== RUN   TestReplacePublicaAliasComoOBuild
--- PASS
```

O erro transitório é montado travando a leitura com acesso exclusivo
(`GENERIC_READ|GENERIC_WRITE`, `share=0`) — o arquivo **continua existindo**,
que é o que caracteriza "transitório".

### GREEN

```
--- PASS: TestReplaceComErroDeLeituraNaoDeixaANotaForaDoIndice (0.02s)
--- PASS: TestReplaceComNotaRemovidaEntreOEventoEALeitura (0.01s)
--- PASS: TestReplacePublicaAliasComoOBuild (0.01s)
```

E, porque isto mexe no coração do índice, os três pacotes que dependem dele com
detector de corrida:

```
ok  github.com/jonyd/gobsidian/internal/index     4.809s
ok  github.com/jonyd/gobsidian/internal/watcher  15.184s
ok  github.com/jonyd/gobsidian/internal/service  55.003s
```

---

## M7 veio junto

`Replace` refazia à mão seis derivações que `publishNoteLocked` já possuía:
`notes`, `lowerPath`, `byName`, `byAlias`, `tags` e citantes.

Ao unificar, apareceu que **`byAlias` tinha três contas**:

| onde | quando |
|---|---|
| `buildAliasMap` | passe separado no fim do `Build` e da carga do cache |
| `Replace` | inline |
| `publishNoteLocked` | **não publicava** |

`byAlias` entrou em `publishNoteLocked`, e `buildAliasMap` foi removida junto
dos seus dois chamadores. Uma nota indexada pelo watcher e outra indexada no
boot passavam por códigos diferentes para produzir a MESMA entrada — que é
exatamente o padrão que produziu o bug `[[STJ]]`, onde o boot escrevia minúsculo
e `Replace` escrevia cru.

**`TestReplacePublicaAliasComoOBuild` é uma guarda de paridade, não um RED.**
Ele passava antes e passa depois; o que ele faz é tornar a troca segura —
falharia se eu adotasse `publishNoteLocked` sem levar o alias junto. Compara as
duas construções pelo **efeito observável** (um link por alias resolvendo em
backlink), não por valor escrito à mão.

---

## Provas de mutação

**A primeira tentativa do A3 saiu `EXIT=1`, e a mutação estava errada, não a
regra.** Registrado porque é a distinção que o `mutate.ps1` existe para forçar:

```
-Anchor 'nota = construirNota(entry, data)'
-Replacement 'nota = construirNota(entry, data); _ = ix.removerEReprocessar(path)'
-> [!] O teste PASSOU com a regra mutada.  EXIT=1
```

A mutação punha a remoção no caminho de **sucesso**, enquanto o teste exercita o
de **falha** — o mutante nunca rodava. A mutação certa restaura a **ordem
antiga**:

```
-Anchor 'data, err := v.ReadAll(ctx, entry.Path)'
-Replacement '_ = ix.removerEReprocessar(path); data, err := v.ReadAll(ctx, entry.Path)'
-> [OK] O teste REPROVOU com a regra mutada.  EXIT=0
```

M7:

```
-Anchor 'ix.byAlias[key] = append(ix.byAlias[key], n.Path)' -Replacement '_ = key'
-> [OK] O teste REPROVOU com a regra mutada.  EXIT=0
```

---

## Verificações

1. **`construirNota` extraída** para que a fase 1 contenha todo o parse — a
   outra metade do trabalho que não precisa de exclusão.
2. **`removerEReprocessar` / `removerEReprocessarLocked`**: o par existe porque
   o caminho "sumiu antes do Stat" entra sem lock e o "sumiu entre as fases"
   entra com ele. Uma conta, dois pontos de entrada.
3. `pwsh -File scripts/verify.ps1`: **14 de 14 [OK]**.

---

## O que ficou de fora

**A contenção do A2 não foi medida, nem antes nem depois.** O redesenho tira o
I/O de dentro do lock, o que é correção estrutural — mas afirmar ganho exigiria
baseline, e nenhum foi tomado. Não há número neste relatório sobre A2, de
propósito.

**A Alternativa 3 (cópia-na-escrita)** segue rejeitada por falta de medição, com
o motivo em `docs/SUGESTOES.md`: trocaria contenção por alocação, e sem baseline
seria troca no escuro.
