# Task 150 — relatório

## Status
DONE

## Commit
`8cfec55 fix(service): note_patch mode goes through ValidarEnum and the error lists the modes that exist`

## Evidência de TDD

**RED** — teste novo adicionado a `internal/service/limites_enums_test.go`, rodado antes de tocar `write.go`:

```
$ go test ./internal/service/ -run TestPatchNoteModeInvalidoListaOsModosQueExistem -v
=== RUN   TestPatchNoteModeInvalidoListaOsModosQueExistem
    limites_enums_test.go:170: a mensagem nao lista o modo real "replace_section": mode = "append_to_heading" invalido; aceitos: replace_heading_and_section, append_to_heading, replace_block, append_to_note
    limites_enums_test.go:175: a mensagem lista o modo fantasma "append_to_heading" como aceito: mode = "append_to_heading" invalido; aceitos: replace_heading_and_section, append_to_heading, replace_block, append_to_note
    limites_enums_test.go:175: a mensagem lista o modo fantasma "append_to_note" como aceito: mode = "append_to_heading" invalido; aceitos: replace_heading_and_section, append_to_heading, replace_block, append_to_note
--- FAIL: TestPatchNoteModeInvalidoListaOsModosQueExistem (0.01s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	0.889s
FAIL
```

Confirma exatamente o que o brief previu: falha por `replace_section` ausente e pelos dois modos fantasma listados como aceitos.

**GREEN** — após trocar o bloco de `mode`/`default` em `write.go` por `ValidarEnum`:

```
$ go test ./internal/service/ -run 'TestPatchNoteModeInvalidoListaOsModosQueExistem|TestPatchModeInvalidoNaoEErroInterno' -v
=== RUN   TestPatchModeInvalidoNaoEErroInterno
--- PASS: TestPatchModeInvalidoNaoEErroInterno (0.02s)
=== RUN   TestPatchNoteModeInvalidoListaOsModosQueExistem
--- PASS: TestPatchNoteModeInvalidoListaOsModosQueExistem (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/service	0.905s
```

Pacote inteiro (`internal/service`), sem `-run`, também verde:

```
$ go test ./internal/service/...
ok  	github.com/jonyd/gobsidian/internal/service	22.999s
```

## Prova de mutação

Regra reivindicada: a lista de modos aceitos vive **só** na chamada a `ValidarEnum`
(`write.go:304-305`); se um modo real sai da lista, o teste novo reprova.

Comando (âncora copiada do arquivo, não de memória):

```bash
pwsh -File scripts/mutate.ps1 -Path internal/service/write.go `
  -Anchor '"replace_section", "replace_heading_and_section", "replace_block")' `
  -Replacement '"replace_heading_and_section", "replace_block")' `
  -Test TestPatchNoteModeInvalidoListaOsModosQueExistem -Package ./internal/service/
```

Saída:

```
Carregado em 702ms
[...] Mutando internal/service/write.go
      - "replace_section", "replace_heading_and_section", "replace_block")
      + "replace_heading_and_section", "replace_block")

[...] go test -race -run TestPatchNoteModeInvalidoListaOsModosQueExistem ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestPatchNoteModeInvalidoListaOsModosQueExistem (0.02s)
    limites_enums_test.go:170: a mensagem nao lista o modo real "replace_section": mode = "append_to_heading" invalido; aceitos: replace_heading_and_section, replace_block
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.566s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

`TestPatchNoteModeInvalidoListaOsModosQueExistem` (linha 170 de
`limites_enums_test.go`) reprovou sob a mutação, pelo motivo esperado
("nao lista o modo real"). Restauro confirmado pela própria saída do
`mutate.ps1` (SHA-256 confere) e por `git diff --stat`:

```
$ git diff --stat internal/service/write.go
 internal/service/write.go | 23 +++++++++++------------
 1 file changed, 11 insertions(+), 12 deletions(-)
```

(as 23 linhas são a mudança real que fica, não a mutação — o `mutate.ps1` já
havia restaurado antes deste diff ser tirado.)

O comando de mutação sugerido pelo despacho (adicionar `"xyz"` à lista aceita)
foi descartado por instrução do orquestrador: não faz o teste novo falhar,
porque a mensagem continua citando os três modos reais — não prova a regra.
Usei em vez disso o comando acima, que remove `replace_section` da lista.

## Achado extra durante o gate: lint

O texto do teste do brief (`strings.Contains(msg[strings.Index(msg,
"aceitos"):], fantasma)` dentro do mesmo `if` do `strings.Contains(msg,
"aceitos")`) reprovou `golangci-lint` (gocritic `offBy1`) nas etapas 6 e 7 do
gate, em ambas as plataformas — `Index()` pode devolver -1 e o linter não liga
o guard do `&&` à fatia. Corrigi extraindo `idxAceitos := strings.Index(msg,
"aceitos")` uma vez, com guarda explícita `idxAceitos != -1` antes de fatiar.
Mesma asserção, mesmo comportamento de teste — só o formato mudou para o
linter aceitar. `go vet` não pegou isso (confirma a regra do papel do
implementador: `go vet` não é `errcheck`/`gocritic`).

## Verificações do brief

1. **A lista de modos aparece UMA vez em `write.go`** — confirmado por
   inspeção: só a chamada a `ValidarEnum` (linhas 304-305) cita os três
   nomes; o `default:` do `switch` (agora inalcançável) não repete a lista,
   só devolve `CodeInternal` com o `mode` que chegou até ali, e o comentário
   diz que é guarda morta.
2. **Schema (`tools_write.go`) e `docs/TOOLS.md` citam os mesmos três modos,
   na mesma ordem** — `tools_write.go:35` e `TOOLS.md:403` citam
   `replace_section, replace_heading_and_section, replace_block` nessa
   ordem. `check_tool_params.ps1`:

   ```
   [OK] todo parametro declarado e lido em algum lugar.
   ```

3. **Mensagem de erro para `mode: "xyz"`** (capturada com um teste
   descartável, rodado e removido, não escrita de memória):

   ```
   mode = "xyz" invalido; aceitos: replace_section, replace_heading_and_section, replace_block
   ```

## `verify.ps1`

Rodado em foreground, sem `-SkipCross`/`-SkipNet`, depois da correção do
lint:

```
[...] 13. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
```

13 etapas, todas `[OK]`.

Uma rodada anterior com `-SkipCross -SkipNet` (antes da correção do lint)
reprovou nas etapas 6 e 7 (`golangci-lint` local e linux) pelo achado acima;
não é evidência de gate verde e não substitui a rodada completa citada.

## O que ficou de fora

Nada do escopo do brief. A correção do lint na formulação exata do teste do
brief (seção "Achado extra" acima) não estava no brief, mas foi necessária
para o gate ficar verde — não é uma categoria de commit diferente, é o mesmo
teste, só reescrito para passar no linter.

## `git status --porcelain`

Antes do `git add`/commit (arquivos do usuário — `test-vault/`,
`.claude/skills/`, `Resume-Claude.ps1`,
`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` — não tocados por
esta tarefa):

```
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
 M docs/TOOLS.md
 M internal/mcpsrv/tools_write.go
 M internal/service/limites_enums_test.go
 M internal/service/write.go
 M "test-vault/test vault/.obsidian/community-plugins.json"
 D "test-vault/test vault/.obsidian/plugins/parity-dumper/main.js"
 D "test-vault/test vault/.obsidian/plugins/parity-dumper/manifest.json"
 M "test-vault/test vault/.obsidian/workspace.json"
?? .claude/skills/troglodita-commit/
?? .claude/skills/troglodita-help/
?? .claude/skills/troglodita-review/
?? .claude/skills/troglodita/
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/review-1a22508..4a06a24.diff
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/review-40521d8..3dfcf8e.diff
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/review-a74b766..1a22508.diff
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-147-base.txt
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-147-brief.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-147-report.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-148-base.txt
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-148-brief.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-148-report.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-149-base.txt
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-149-brief.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-149-report.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-150-base.txt
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-150-brief.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-150-report.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-151-brief.md
?? Resume-Claude.ps1
?? "test-vault/test vault/.obsidian/community-plugins (conflito 01M08RSCY410E88M0516VB4K9R 2026-08-17 22h52).json"
?? "test-vault/test vault/.obsidian/core-plugins (conflito 01M08RSCY410E88M0516VB4K9R 2026-08-17 22h52).json"
?? "test-vault/test vault/.obsidian/plugins/gosync/"
?? "test-vault/test vault/Ação.md"
?? "test-vault/test vault/Pasted image 20260814221454.png"
?? "test-vault/test vault/Sem título.md"
?? "test-vault/test vault/main.js"
?? "test-vault/test vault/manifest.json"
```

Apenas `internal/service/write.go`, `internal/service/limites_enums_test.go`,
`internal/mcpsrv/tools_write.go` e `docs/TOOLS.md` foram adicionados ao
commit.
