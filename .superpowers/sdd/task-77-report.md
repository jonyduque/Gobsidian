# Relatório Task 77: fechamento do M6 e preparação da v1.0.0

**Data:** 2026-08-02
**Base:** `98b61a0`
**Commits:** `fbd74ed` (tabela completa de RNFs e as três medições novas),
`ac4c2cf` (ledger e este relatório), e a correção deste relatório logo depois —
os SHA-256 dos binários não podem estar no commit que os produz, então a
conferência dos números vive um commit à frente da tag.
**Tag:** `v1.0.0` criada **localmente**, apontando para `ac4c2cf`. **Não
publicada** — ver "O que ficou de fora" e a seção de publicação.

---

## Estado da release: PUBLICADA em 2026-08-03

Publicada depois, por decisão de quem pediu. A seção abaixo descreve o estado no
fechamento da tarefa, quando ela ainda não existia; o que a publicação mudou está
em `.superpowers/sdd/task-77-release-report.md`.

## Estado no fechamento da tarefa: PREPARADA, NÃO PUBLICADA

Isto é dito primeiro porque a armadilha que esta tarefa existe para não repetir é
exatamente afirmar estado não verificado. O README já declarou "v0.1 publicada"
sem tag e sem release.

Hoje, o que existe de verdade:

| Coisa | Estado | Como conferir |
|---|---|---|
| Tag `v1.0.0` local | **existe**, aponta para `ac4c2cf` | `git log -1 --oneline v1.0.0` |
| Tag `v1.0.0` no remoto | **não existe** | não foi feito `git push origin v1.0.0` |
| Release `v1.0.0` no GitHub | **não existe** | `gh release list` traz só `v0.1.0` |
| Binários dos três alvos | **existem em `dist/`**, com versão embutida | SHA-256 abaixo |
| `README.md` | **não foi tocado**; segue falando de v0.1 | é a única afirmação que continua verdadeira |

O `README.md` **não** foi alterado de propósito. Escrever "v1.0.0 publicada"
antes da release existir seria repetir a falha nomeada no brief.

---

## Evidência de TDD e prova de mutação

**Esta tarefa não tem prova de mutação, por construção**: o entregável é o portão
e o release, não uma regra nova. O brief diz isso explicitamente, e o que a
substitui é a saída real de cada comando, colada abaixo.

O único código novo aqui é `TestRNF03_RNF05_RNF06`, e ele teve ciclo:

### RED

A primeira execução reprovou em dois pontos, e os dois eram defeitos reais da
medição, não do produto:

```
RNF-03 note_read                     mediana 0s        p95 648.7µs
RNF-03 note_read: mediana = 0s, a medição não mediu nada
RNF-05 note_list (filtro de tag)     mediana 0s        p95 2.0271ms
RNF-05 note_list (filtro de tag): mediana = 0s, a medição não mediu nada
RNF-06 index.Replace (arquivo unico) mediana 18.5213ms p95 29.1249ms  teto 20ms
RNF-06 index.Replace (arquivo unico): p95 = 29.1249ms excede o teto de 20ms
--- FAIL: TestRNF03_RNF05_RNF06
```

### GREEN

Com a medição por lote e a separação entre alvo do RNF e linha de degradado do
PRD:

```
--- PASS: TestRNF03_RNF05_RNF06 (2.36s)
```

---

## Portão de release — saída real de cada comando

### `pwsh -File scripts/verify.ps1`

```
[...] 1. go build
[OK] go build
[...] 2. go test -race
[OK] go test -race
[...] 3. go test (tetos de latencia, sem -race)
[OK] go test (tetos de latencia, sem -race)
[...] 4. go vet (windows)
[OK] go vet (windows)
[...] 5. go vet (linux)
[OK] go vet (linux)
[...] 6. go vet (darwin)
[OK] go vet (darwin)
[...] 7. gofmt
[OK] gofmt
[...] 8. golangci-lint
[OK] golangci-lint
[...] 9. check_net (RNF-30)
[OK] check_net (RNF-30)
[...] 10. check_tool_params
[OK] check_tool_params

[OK] Bateria completa. Pode commitar.
VERIFY_EXIT=0
```

São **10** etapas, não as nove que o brief cita: `check_tool_params` entrou na
Task 69 e o teto do RNF-04 virou etapa própria depois da revisão da Task 72.

### `pwsh -File scripts/test_orphans.ps1 -Cycles 100`, nos três cenários

```
########## -Scenario stdin-eof ##########
    stdin-eof: 100x
[OK] Nenhum orfao em 100 ciclos
ORPHANS_stdin-eof_EXIT=0
########## -Scenario parent-death ##########
    parent-gone: 100x
[OK] Nenhum orfao em 100 ciclos
ORPHANS_parent-death_EXIT=0
########## -Scenario signal ##########
    signal: 100x
[OK] Nenhum orfao em 100 ciclos
ORPHANS_signal_EXIT=0
```

Cada cenário produziu **apenas** o motivo do mecanismo que ele nomeia. Antes da
Task 76 havia um cenário só, e `stdin-eof` vencia 100% de tudo.

### `pwsh -File scripts/check_tool_params.ps1`

```
[i] 12 structs de entrada, 68 parametros declarados.
[OK] todo parametro declarado e lido em algum lugar.
TOOLPARAMS_EXIT=0
```

### `pwsh -File scripts/audit_reports.ps1`

```
[!] 40 achado(s).
AUDIT_EXIT=1
```

**Sai 1, e isso não bloqueia.** Os 40 achados são de relatórios das Tasks 1 a 28
e 62 — seções ausentes em relatórios antigos, dois SHA de ledger cuja descrição
não bate com o range, e um `deadbee` que é **prosa** descrevendo uma sonda, não
um SHA citado. **Zero achados nas Tasks 69 a 77.** O script não julga conteúdo;
localiza a frase para alguém conferir, e foi conferida.

### `pwsh -File scripts/check_briefs.ps1`

```
[!] 65 achado(s).
BRIEFS_EXIT=1
```

Mesma leitura: todos são de briefs das Tasks 12 a 68. Filtrando por `task-69` a
`task-77`, **nenhum achado**.

---

## Binários

Compilação cruzada com `CGO_ENABLED=0`, `-trimpath` e a versão embutida por
`-ldflags`, a partir de `ac4c2cf` — o commit para onde a tag `v1.0.0` aponta:

```
version=v1.0.0 commit=ac4c2cf date=2026-08-02T20:55:59-03:00
```

| Arquivo | SHA-256 |
|---|---|
| `gobsidian_windows_amd64.exe` | `9735e344f2085c6b07239694656be0df5897b52ab512ba37be258ef9030e82b1` |
| `gobsidian_linux_amd64` | `3e4f4ff4ec242d776629cfeab449f2e11da535dc0ae27e8c951c6b209c985e5a` |
| `gobsidian_darwin_arm64` | `dc9211dac0ff94084afc666ea6d893b53ea8e66d6b6b36a522dfe4dc7d111f64` |

**O que foi verificado de cada um.** O de Windows **executa** nesta máquina:

```
gobsidian v1.0.0 (ac4c2cf) 2026-08-02T20:55:59-03:00
```

Os outros dois **não foram executados** — não há Linux nem macOS aqui. O que se
verificou deles é o formato:

```
gobsidian_linux_amd64:  ELF 64-bit LSB executable, x86-64, statically linked, stripped
gobsidian_darwin_arm64: Mach-O 64-bit arm64 executable, flags:<|DYLDLINK|PIE>
```

`gobsidian version` nesses dois alvos: **não verificado**. O CI roda `go test`
nos três sistemas, o que exercita o código, mas não este binário.

---

## Tabela de RNFs

`docs/OPERACAO.md` passou de 5 dos 22 RNFs para **os 22**, cada um com número
medido ou a palavra "não medido".

**Três RNFs não estão atingidos**, todos medidos:

| ID | Alvo | Medido | Diferença |
|---|---|---|---|
| **RNF-04** | p95 ≤ 100 ms | 181,25 ms para `limit: 200` a 5.000 notas | +81% |
| **RNF-06** | ≤ 20 ms | mediana 20,35 ms a 5.000 notas | +2% |
| **RNF-07** | RSS ≤ 60 MB | 67,08 MB cache quente; 112,96 MB a frio | +12% / +88% |

**Dois não foram medidos**, e estão escritos assim: RNF-08 (CPU em repouso) e
RNF-09 (linearidade até 20.000 notas).

### As três medições novas desta tarefa

```
  RNF-03 note_read                     mediana 206.466µs  p95 344.97µs   alvo 15ms   degradado 50ms    ATINGIDO
  RNF-05 note_list (filtro de tag)     mediana 249.24µs   p95 533.68µs   alvo 10ms   degradado 30ms    ATINGIDO
  RNF-06 index.Replace (arquivo unico) mediana 20.3475ms  p95 30.1365ms  alvo 20ms   degradado 100ms   NAO ATINGIDO
```

A primeira tentativa de medir RNF-03 e RNF-05 devolveu **mediana 0s**: as duas
operações custam centenas de microssegundos, abaixo do passo do relógio do
Windows. A guarda de mediana zerada acusou. A correção foi cronometrar um lote de
50 leituras (RNF-03) ou 20 listagens (RNF-05) e dividir — não afrouxar a
guarda.

---

## Conferência de todo SHA do ledger

`git cat-file -t` em cada um dos 19 SHAs registrados nas Tasks 69 a 77:

```
ac91156   commit      c6a0baa   commit      be2d2b6   commit
d15fddf   commit      0a8ba36   commit      3a04e3f   commit
daab822   commit      718995f   commit      12ad94f   commit
c0cfd9a   commit      74c4ecd   commit      23ecc2e   commit
2da8b90   commit      38a0cca   commit      74ecabb   commit
6cdc79e   commit      fbd74ed   commit
22836a6   commit      8679a70   commit
```

Nenhum aponta para o vazio. A Task 31 foi registrada em `14210ee`, que não
existe, e é por isso que esta conferência é passo obrigatório.

E a tag:

```
$ git log -1 --oneline v1.0.0
ac4c2cf docs(ledger): close M6 with tasks 69-77, and record that v1.0.0 is prepared but not published
```

A tag foi criada primeiro em `fbd74ed` e movida para `ac4c2cf` depois do commit
de ledger, para que quem baixar a tag receba também o relatório desta tarefa. Os
binários foram **recompilados** na mudança: o carimbo de commit dentro do binário
tem de bater com o commit da tag, senão o `gobsidian version` de um artefato de
release aponta para outro lugar.

---

## Como publicar

Não foi feito. A sequência, na ordem:

```bash
git push origin v1.0.0
gh release create v1.0.0 dist/gobsidian_windows_amd64.exe \
    dist/gobsidian_linux_amd64 dist/gobsidian_darwin_arm64 dist/SHA256SUMS.txt \
    --title "v1.0.0" --notes-file <notas>
gh release list          # confirmar que existe ANTES de escrever que existe
```

Só **depois** de `gh release list` mostrar `v1.0.0` é que o `README.md` pode
passar a dizê-lo. As notas precisam trazer os três RNFs não atingidos: quem
instala tem de saber que `limit: 200` a 5.000 notas custa 181 ms e que o RSS
passa de 60 MB nessa escala.

As tags de marco (`m2-watcher`, `m3-search`, `m4-writer`, `m5-refactor`) também
seguem só locais.

---

## O que ficou de fora

- **A publicação.** Decisão de quem pediu, tomada explicitamente: preparar tudo e
  parar antes de publicar.
- **RNF-08 e RNF-09.** Não medidos, escritos como não medidos. RNF-09 exige um
  cofre de 20.000 notas que não foi gerado; RNF-08 exige amostragem de CPU do
  processo em repouso, que nenhum harness deste projeto faz.
- **`gobsidian version` nos binários de Linux e macOS.** Não há como executá-los
  aqui. O formato foi conferido; a execução, não.
- **Os três RNFs não atingidos.** Fechá-los é trabalho novo, fora do M6: RNF-04
  pede atacar o custo por resultado da busca, RNF-07 pede reduzir a pegada do
  índice invertido, e RNF-06 está a 2% do alvo.
