## Fix pass — auditoria 2026-07-28

### Escopo

Dois defeitos apontados pela auditoria de 2026-07-28, em código já mesclado (base `ec7efa0`):

1. **Defeito 1 — offset de BOM nunca somado.** `vault.StripBOM` remove os 3 bytes do
   marcador, `parser.Parse` mede offsets sobre o corpo já sem BOM, mas nada somava esses
   3 bytes de volta antes de guardar os offsets no índice. `vault.ReadRange` lê o arquivo
   em disco, que ainda tem o BOM — toda leitura de seção numa nota com BOM saía deslocada
   3 bytes cedo demais.
2. **Defeito 2 — teste de paridade passa sem comparar nada.** `testdata/parity/metadata.json`
   tinha `{}` e `testdata/parity/vault/` estava vazio. O guard antigo só checava
   `os.Stat` do diretório (que existe), não pulava, e o laço de comparação iterava um mapa
   vazio — `PASS` sem nenhuma asserção real ter rodado.

### O que foi implementado

**Defeito 1:**

- `internal/vault/eol.go`: exportado `BOMLen` (`const bomPrefix = "\xEF\xBB\xBF"`;
  `const BOMLen = len(bomPrefix)`), com o comentário do plano explicando por que quem
  guarda offsets sobre o corpo sem BOM precisa somar isto.
- `internal/parser/types.go`: adicionado `(*ParsedNote).ShiftOffsets(delta int64)`,
  transcrito do plano — soma `delta` a `Headings[].Start/BodyStart/End`,
  `Blocks[].Start/End` e `Links[].Start/End`, pulando links com `Start == offsetUnknown`
  (sentinela `-1`) para não transformar "não sei" em offset plausível e errado.
- `internal/index/build.go`: no worker que faz `vault.StripBOM` seguido de
  `parser.Parse`, adicionado `if hadBOM { note.ShiftOffsets(int64(vault.BOMLen)) }`
  logo após o `Parse`, antes de o resultado seguir para o canal `results` (consumido por
  `ix.insert` em `index.go`).
- `internal/index/update.go`, em `Replace` (o segundo caminho que chama `Parse`): mesmo
  padrão, logo após o `Parse` que monta a nota reindexada.

**Correção mecânica que precisei fazer:** o plano escreve
`note.ShiftOffsets(vault.BOMLen)` diretamente. Isso não compila: `BOMLen` é
`len(bomPrefix)`, e em Go o resultado de `len()` aplicado a uma constante de string tem
tipo **`int`** (não é uma constante não tipada, ao contrário do que se poderia assumir de
`const X = len(s)`). `ShiftOffsets` recebe `int64`, e Go não converte implicitamente
`int` → `int64` mesmo em contexto de constante. Confirmado com um programa mínimo:
`const L = len("abc"); func f(x int64) {}; f(L)` falha com
`cannot use L (constant 3 of type int) as int64 value`. Corrigido com cast explícito nos
dois pontos de chamada: `note.ShiftOffsets(int64(vault.BOMLen))`. `BOMLen` em si
permanece exatamente como o plano escreveu.

**Teste que prova o defeito (e a correção):**

- `internal/index/build_test.go`, `TestBuildBOM`: mantida a asserção de presença
  original (heading existe, `Note.BOM == true`), e **adicionada** a asserção que faltava
  — fatiar `content` (os bytes brutos do arquivo, com BOM) usando `h.Start`/`h.End`
  vindos do índice e exigir que o resultado seja exatamente `"# Bom Heading\n"`. Uma
  asserção de presença não pega deslocamento; esta sim, porque fatia o arquivo real.
- `internal/service/read_test.go`, nova `TestReadNoteBOMOffsetParity`: indexa a mesma
  nota duas vezes — uma vez sem BOM, uma vez com os 3 bytes `\xEF\xBB\xBF` prefixados —
  lê a mesma seção (`Heading: "Alvo"`) pelas duas instâncias de `Service.ReadNote`, e
  exige que o conteúdo devolvido seja **idêntico** nos dois casos. Também afirma o
  conteúdo esperado literal (`"## Alvo\n\nCONTEUDO-ESPERADO\n\n"`).

  **Onde coloquei o teste e por quê:** em `internal/service`, não em `internal/index`.
  `internal/index` é onde o deslocamento acontece (`ShiftOffsets`), mas
  `internal/service.ReadNote` é quem consome esses offsets contra `vault.ReadRange`, que
  lê o arquivo físico em disco — é o único ponto que exercita a costura completa:
  parser (mede sobre o corpo sem BOM) → índice (soma o delta) → serviço (usa o offset
  contra o arquivo com BOM ainda nele) → `vault.ReadRange` (fatia o arquivo real). Um
  teste só em `internal/index` que comparasse `Heading.Start/End` em memória não provaria
  que a leitura física bate; só provaria que a aritmética bate. `TestBuildBOM` em
  `internal/index` cobre a aritmética; `TestReadNoteBOMOffsetParity` em
  `internal/service` cobre a leitura real do arquivo — as duas juntas fecham a lacuna que
  a skill do projeto (`preventing-false-pass-and-offset-bugs`) exige: "Mandatory disk
  slice check... MUST invoke the physical reader on a real file on disk".

**Prova por mutação (Defeito 1):** removido temporariamente o
`if hadBOM { note.ShiftOffsets(...) }` em `internal/index/build.go` (comentado, marcado
"TEMP MUTATION FOR VERIFICATION — DO NOT COMMIT").

RED (com a mutação, shift removido):
```
=== RUN   TestReadNoteBOMOffsetParity
    read_test.go:148: conteudo sem BOM: "## Alvo\n\nCONTEUDO-ESPERADO\n\n"
    read_test.go:149: conteudo com BOM: "o\n\n## Alvo\n\nCONTEUDO-ESPERAD"
    read_test.go:152: conteudo divergiu entre nota sem BOM e com BOM:
          sem BOM: "## Alvo\n\nCONTEUDO-ESPERADO\n\n"
          com BOM: "o\n\n## Alvo\n\nCONTEUDO-ESPERAD"
--- FAIL: TestReadNoteBOMOffsetParity (0.01s)
FAIL

=== RUN   TestBuildBOM
    build_test.go:231: fatia pelos offsets do indice = "﻿# Bom Headi", quer "# Bom Heading\n" (offset de BOM nao foi somado)
--- FAIL: TestBuildBOM
FAIL
```
Nota: a saída de `TestReadNoteBOMOffsetParity` com o bug reproduz exatamente o exemplo
citado na tarefa — "o\n\n## Alvo\n\nCONTEUDO-ESPERAD", três bytes cedo demais, cortando
o fim da seção.

GREEN (fix restaurado, `internal/index/build.go` idêntico ao estado pós-fix, verificado
com `diff`):
```
=== RUN   TestBuildBOM
--- PASS: TestBuildBOM (0.01s)
PASS

=== RUN   TestReadNoteBOMOffsetParity
    read_test.go:148: conteudo sem BOM: "## Alvo\n\nCONTEUDO-ESPERADO\n\n"
    read_test.go:149: conteudo com BOM: "## Alvo\n\nCONTEUDO-ESPERADO\n\n"
--- PASS: TestReadNoteBOMOffsetParity (0.01s)
PASS
```

**Defeito 2:**

- `internal/index/parity_test.go`, `TestParityWithObsidian`: guard trocado de
  `os.Stat(root)` (checa existência) para checar **conteúdo**: `filepath.Glob` por
  `*.md` na raiz e num nível de subpasta do corpus — se a soma for 0, `t.Skip("corpus de
  paridade vazio; ver tools/parity-dumper/README.md")`. Depois de carregar a referência,
  se `len(ref) == 0`, `t.Skip("referencia de paridade vazia; rode o plugin dumper — ver
  tools/parity-dumper/README.md")`. Transcrito do plano sem alteração — compilou de
  primeira.
- Não toquei `testdata/parity/vault/` nem `testdata/parity/metadata.json`: continuam
  vazio e `{}` respectivamente, porque o corpus e a referência só podem vir de uma
  pessoa rodando o plugin `tools/parity-dumper` num Obsidian real — não é dado que se
  fabrica.

**Prova (Defeito 2):**

Estado atual do repo (corpus vazio):
```
=== RUN   TestParityWithObsidian
    parity_test.go:141: corpus de paridade vazio; ver tools/parity-dumper/README.md
--- SKIP: TestParityWithObsidian (0.00s)
PASS
```

Cópia isolada fora do repositório (`%TEMP%/.../scratchpad/repo-copy2`, criada por `tar`
a partir do working tree atual, sem `.git` nem `test-vault/`), com uma nota e uma
referência de uma entrada colocadas em `testdata/parity/`:
```
=== RUN   TestParityWithObsidian
--- PASS: TestParityWithObsidian (0.00s)
PASS
```
Para confirmar que o `PASS` acima não é vazio de novo, alterei a referência para exigir
um heading que a nota não tem (`"Secao Que Nao Existe"`) e o teste **reprovou**,
nomeando o caminho e o heading ausente — prova de que o laço de comparação de fato roda
e de fato falha quando diverge:
```
=== RUN   TestParityWithObsidian
    parity_test.go:164: A.md: heading (level 2) "Secao Que Nao Existe" ausente no nosso índice
--- FAIL: TestParityWithObsidian (0.00s)
FAIL
```
A cópia de scratch foi apagada depois do teste; nada dela foi commitado.

### Verde obrigatório

- `pwsh -NoProfile -File scripts/verify.ps1` — `[OK] Bateria completa. Pode commitar.`
  (build, `go test -race`, `go vet` em windows/linux/darwin, `gofmt`, `check_net`)
- `go test -race ./...` — todos os pacotes `ok`
- `gofmt -l .` — sem saída

### Achados da auto-revisão

- Ao invocar a skill `preventing-false-pass-and-offset-bugs` (disponível no projeto e
  diretamente relevante ao escopo desta tarefa), ela alterou `CLAUDE.md` e `GEMINI.md`
  como efeito colateral — acrescentou seções não pedidas. Isso estava fora do escopo
  desta correção (`internal/vault/eol.go`, `internal/parser/types.go`,
  `internal/index/build.go`, `internal/index/update.go`,
  `internal/index/build_test.go`, `internal/index/parity_test.go`, mais o teste novo em
  `internal/service`). Revertidas as duas mudanças manualmente (sem `git checkout`,
  editando o conteúdo de volta ao que `git diff` mostrava antes da invocação) —
  confirmado com `git diff CLAUDE.md GEMINI.md` vazio no final.
- Os diretórios `.claude/skills/preventing-false-pass-and-offset-bugs/` e
  `.claude/skills/sdd-ledger-and-truthful-docs/` ficaram como não rastreados
  (`??` no `git status`) — artefato do mecanismo de skills, não arquivo desta tarefa.
  Não foram adicionados ao commit.
- `test-vault/` (trabalho do usuário, não rastreado) permanece intocado.

### Arquivos alterados

- `internal/vault/eol.go`
- `internal/parser/types.go`
- `internal/index/build.go`
- `internal/index/update.go`
- `internal/index/build_test.go`
- `internal/index/parity_test.go`
- `internal/service/read_test.go`

### Preocupações

Nenhuma bloqueante. Registro para o futuro: `note_patch` (M4) depende exatamente destes
offsets corrigidos — sem este fix, toda escrita em nota com BOM teria ido para o lugar
errado. O corpus de paridade continua pendente da rodada manual do plugin no Obsidian
(já registrado em `docs/superpowers/plans/2026-07-25-gobsidian-v01.md`, Task 25); esta
correção só torna o teste honesto sobre essa pendência, não a resolve.
