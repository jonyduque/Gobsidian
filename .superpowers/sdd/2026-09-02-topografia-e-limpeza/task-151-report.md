# Task 151 — relatório

## Status

`DONE`

## Commit

```
9fb5500 fix(service): trash moves the note instead of copying it, and the copy fallback refuses cloud placeholders
```

Arquivos no commit (4): `internal/service/write.go`,
`internal/service/delete_test.go`,
`internal/service/erro_engolido_windows_test.go`, `docs/TOOLS.md`.

Nota sobre o commit: a primeira tentativa usou a sintaxe de here-string do
PowerShell (`@'…'@`) na ferramenta Bash, que é Git Bash — o `@` virou parte do
assunto (`@ fix(service): …`). Corrigido com `git commit --amend -F <arquivo>`
antes de qualquer outra coisa; SHA final `9fb5500`. Nenhum comando proibido foi
usado (`--amend` não está na lista de proibidos; `checkout`/`restore`/`stash`/
`clean`/`reset` não foram tocados).

---

## Desvios do brief

Dois, ambos deliberados e mínimos:

**1. O teste portátil usa `createDeleteService`, não `newTestService`/`writeFile`.**
Ruling 1 do despacho. `internal/service/delete_test.go` é `package service_test`
e os helpers nomeados no Step 1 do brief moram em `package service`
(`read_test.go`), inalcançáveis dali. As asserções são as do brief, palavra por
palavra.

**2. `destinoNaLixeira` devolve `vault.CanonicalPath`, não `string`.**
O brief escrevia `absTrash, _, err = vault.Resolve(...)` — descartando o
`CanonicalPath` que `Resolve` já calculou — e o chamador fazia
`vault.CanonicalPath(trashRel)`, uma segunda conversão do mesmo texto. Isso é
exatamente o padrão que "uma conta por regra" (`CLAUDE.md`) proíbe: a chave da
lixeira é a chave que a trava de `moverCorpo` usa, e duas contas para a mesma
chave é como elas divergem. A assinatura ficou:

```go
func (s *Service) destinoNaLixeira(canonical vault.CanonicalPath) (trashRel vault.CanonicalPath, absTrash string, err error)
```

e o chamador usa `string(trashRel)` para o campo `TrashPath` do resultado.
Comportamento idêntico — `Canonicalize` preserva a grafia e só normaliza barras,
e `trashRel` já era um caminho com barras — mas com uma origem só para o valor.
A lógica de colisão de nome não mudou.

---

## Evidência de TDD

### RED — teste portátil (Step 2)

```
$ go test ./internal/service -run TestDeleteNoteToTrashMoveSemCopiar -timeout 60s -v
=== RUN   TestDeleteNoteToTrashMoveSemCopiar
    delete_test.go:172: mtime da copia na lixeira = 2026-09-03 19:55:01.1197168 -0300 -03, quer 2020-01-02 03:04:05 +0000 UTC: a lixeira COPIOU em vez de mover
--- FAIL: TestDeleteNoteToTrashMoveSemCopiar (0.06s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.433s
FAIL
```

A verificação 1 do brief está satisfeita: o teste REPROVOU antes do fix, com a
mensagem que o brief previu. Nenhuma asserção foi trocada.

### RED — teste Windows (Step 4)

```
$ go test ./internal/service -run TestDeleteNoteToTrashNaoBaixaPlaceholder -timeout 60s -v
=== RUN   TestDeleteNoteToTrashNaoBaixaPlaceholder
    erro_engolido_windows_test.go:110: codigo = INTERNAL, quer CLOUD_ONLY_FILE: lendo nota "nuvem.md": open C:\Users\jonyd\AppData\Local\Temp\TestDeleteNoteToTrashNaoBaixaPlaceholder4029424390\001\nuvem.md: The process cannot access the file because it is being used by another process.
--- FAIL: TestDeleteNoteToTrashNaoBaixaPlaceholder (0.02s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.422s
FAIL
```

`INTERNAL`, não `CLOUD_ONLY_FILE` — o código antigo leu (tentou ler) o
placeholder.

### GREEN — Step 7, antes do ajuste do teste do B5

```
$ go test ./internal/service -run 'Delete|Trash|Move' -race -timeout 60s -v
=== RUN   TestCaminhoUnicodeSobreviveAoMove
--- PASS: TestCaminhoUnicodeSobreviveAoMove (0.02s)
=== RUN   TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing
--- PASS: TestMoveNote_BrokenAnchorsReportedOnlyWhenMissing (0.03s)
=== RUN   TestDeleteNote_BrokenAnchorsReportedOnDeletion
--- PASS: TestDeleteNote_BrokenAnchorsReportedOnDeletion (0.02s)
=== RUN   TestDeleteNote_ReportBrokenLinksBeforeDeletion
--- PASS: TestDeleteNote_ReportBrokenLinksBeforeDeletion (0.02s)
=== RUN   TestDeleteNote_ToTrashFalseDefiniteDelete
--- PASS: TestDeleteNote_ToTrashFalseDefiniteDelete (0.02s)
=== RUN   TestDeleteNote_TrashNameCollision
--- PASS: TestDeleteNote_TrashNameCollision (0.04s)
=== RUN   TestDeleteNoteToTrashMoveSemCopiar
--- PASS: TestDeleteNoteToTrashMoveSemCopiar (0.01s)
=== RUN   TestDeleteNote_DryRunDoesNotDelete
--- PASS: TestDeleteNote_DryRunDoesNotDelete (0.02s)
=== RUN   TestDeleteToTrashNaoMenteQuandoORemoveFalha
    erro_engolido_windows_test.go:68: o erro nao explica que a copia na lixeira existe: a nota foi copiada para ".trash/origem.md" mas a origem "origem.md" nao pode ser removida (remove C:\Users\jonyd\AppData\Local\Temp\TestDeleteToTrashNaoMenteQuandoORemoveFalha3459908827\001\origem.md: The process cannot access the file because it is being used by another process.); a nota existe nos dois caminhos ate a origem ser liberada
--- FAIL: TestDeleteToTrashNaoMenteQuandoORemoveFalha (0.02s)
=== RUN   TestDeleteNoteToTrashNaoBaixaPlaceholder
--- PASS: TestDeleteNoteToTrashNaoBaixaPlaceholder (0.01s)
=== RUN   TestMoveDryRunNaoApresentaDiffVazioComoResultado
--- PASS: TestMoveDryRunNaoApresentaDiffVazioComoResultado (0.01s)
=== RUN   TestMoveNaoReportaSucessoComNotaDuplicada
    move_atomico_windows_test.go:81: estado duplicado, mas o erro foi reportado: a nota foi copiada para "destino.md" mas a origem "origem.md" nao pode ser removida (…)
--- PASS: TestMoveNaoReportaSucessoComNotaDuplicada (0.01s)
=== RUN   TestMoveNaoReescreveCitantesAntesDeMoverOCorpo
--- PASS: TestMoveNaoReescreveCitantesAntesDeMoverOCorpo (0.01s)
=== RUN   TestNoteMovePartialFailureReportsWhatWasApplied
--- PASS: TestNoteMovePartialFailureReportsWhatWasApplied (0.14s)
=== RUN   TestMoveNote_DryRunLeavesMtimeIntact
--- PASS: TestMoveNote_DryRunLeavesMtimeIntact (0.03s)
=== RUN   TestMoveNote_UpdateLinksFalse
--- PASS: TestMoveNote_UpdateLinksFalse (0.01s)
=== RUN   TestMoveNote_CreateFoldersFalseMissingDir
--- PASS: TestMoveNote_CreateFoldersFalseMissingDir (0.01s)
=== RUN   TestMoveNote_OutsideVaultAndAlreadyExists
--- PASS: TestMoveNote_OutsideVaultAndAlreadyExists (0.02s)
=== RUN   TestMoveNote_PreservesAliasAndAnchor
--- PASS: TestMoveNote_PreservesAliasAndAnchor (0.04s)
=== RUN   TestMoveNote_HappyPathActuallyMovesTheFile
--- PASS: TestMoveNote_HappyPathActuallyMovesTheFile (0.03s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	2.115s
FAIL
```

A única falha é a que o Step 7 do brief previu, com a mensagem prevista:
`TestDeleteToTrashNaoMenteQuandoORemoveFalha` procurava a palavra "lixeira" e o
erro agora vem de `moverCorpo`, que nomeia o destino pelo caminho
(`.trash/origem.md`). Asserção trocada por `.trash` — mesma garantia, a palavra
que a conta única usa —, com comentário no teste explicando a data e o motivo. A
troca está registrada no corpo do commit.

### GREEN — depois do ajuste

```
$ go test ./internal/service -run 'Delete|Trash|Move' -race -timeout 60s
ok  	github.com/jonyd/gobsidian/internal/service	2.981s
```

---

## Prova de mutação

### Regra 1 — o guarda de placeholder no fallback de `moverCorpo`

Comando (âncora copiada do arquivo):

```
pwsh -File scripts/mutate.ps1 -Path internal/service/write.go `
  -Anchor 'if n, ok := s.index.Get(de); ok && n.CloudOnly {' `
  -Replacement 'if false {' `
  -Test TestDeleteNoteToTrashNaoBaixaPlaceholder -Package ./internal/service/
```

Saída:

```
[...] Mutando internal/service/write.go
      - if n, ok := s.index.Get(de); ok && n.CloudOnly {
      + if false {

[...] go test -race -run TestDeleteNoteToTrashNaoBaixaPlaceholder ./internal/service/
----------------------------------------------------------------------
--- FAIL: TestDeleteNoteToTrashNaoBaixaPlaceholder (0.01s)
    erro_engolido_windows_test.go:113: codigo = INTERNAL, quer CLOUD_ONLY_FILE: lendo nota de origem "nuvem.md": open C:\Users\jonyd\AppData\Local\Temp\TestDeleteNoteToTrashNaoBaixaPlaceholder62660994\001\nuvem.md: The process cannot access the file because it is being used by another process.
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.582s
FAIL
----------------------------------------------------------------------
[OK] internal/service/write.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

Teste que reprovou: `TestDeleteNoteToTrashNaoBaixaPlaceholder`,
`internal/service/erro_engolido_windows_test.go:113`. `EXIT=0` — a regra está
verificada. Restauro confirmado pelo próprio script (SHA-256 byte a byte).

### Regra 2 — a lixeira move, não copia (mutação manual)

Mutação aplicada — a chamada a `moverCorpo` no ramo `ToTrash` trocada por uma
cópia:

```diff
-		if err := s.moverCorpo(ctx, canonical, trashRel, absTrash); err != nil {
-			return DeleteNoteResult{}, err
-		}
+		raw, errLeitura := os.ReadFile(absPath)
+		if errLeitura != nil {
+			return DeleteNoteResult{}, Errorf(CodeInternal, "MUTACAO lendo: %v", errLeitura)
+		}
+		if err := writer.WriteAtomic(ctx, absTrash, raw); err != nil {
+			return DeleteNoteResult{}, err
+		}
+		if err := os.Remove(absPath); err != nil {
+			return DeleteNoteResult{}, err
+		}
```

Saída:

```
$ go test ./internal/service -run TestDeleteNoteToTrashMoveSemCopiar -timeout 60s -v
=== RUN   TestDeleteNoteToTrashMoveSemCopiar
    delete_test.go:172: mtime da copia na lixeira = 2026-09-03 19:58:09.9040159 -0300 -03, quer 2020-01-02 03:04:05 +0000 UTC: a lixeira COPIOU em vez de mover
--- FAIL: TestDeleteNoteToTrashMoveSemCopiar (0.04s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/service	1.367s
FAIL
```

Teste que reprovou: `TestDeleteNoteToTrashMoveSemCopiar`,
`internal/service/delete_test.go:172`.

Restauro (edição reversa, nunca `git checkout`), confirmado por busca e por
teste:

```
$ grep -n "MUTACAO" -r internal/
sem mutacao residual

$ go test ./internal/service -run TestDeleteNoteToTrashMoveSemCopiar -timeout 60s
ok  	github.com/jonyd/gobsidian/internal/service	1.359s
```

`git diff --stat internal/service/write.go` depois do restauro mostrava
`46 insertions(+), 47 deletions(-)` — as mudanças da própria Task 151, sem
resíduo da mutação; o commit `9fb5500` carrega exatamente essas linhas.

---

## As verificações do brief

**1. `TestDeleteNoteToTrashMoveSemCopiar` reprova antes do fix.**
Confirmado — saída colada em "RED — teste portátil" acima. O mtime fixado em
2020-01-02 virou o mtime do momento da execução, porque `WriteAtomic` criou um
arquivo novo. Nenhuma asserção foi trocada.

**2. Sem deadlock: o caminho da lixeira não passa pelas duas travas.**
Confirmado. O ramo `ToTrash` retorna antes de `s.locker.Lock(canonical)`, que
agora fica logo acima do `os.Remove` da exclusão definitiva. Todo `go test`
desta tarefa rodou com `-timeout 60s`, inclusive `-race`, e nenhum travou —
o mais longo foi 2,981 s.

**3. O guarda está no fallback de cópia, e o teste o exercita com o rename
forçado a falhar.**
Confirmado. `travaExclusiva` abre `GENERIC_READ|GENERIC_WRITE` com `share=0`
antes da chamada, o `os.Rename` é recusado e o fallback é o caminho que roda —
prova disso é a mensagem do RED, `lendo nota de origem "nuvem.md"`, que só
existe dentro do fallback de `moverCorpo`.

**4. `TestDeleteToTrashNaoMenteQuandoORemoveFalha` continua passando com a
asserção ajustada para `.trash`.**
Confirmado — está no `ok` do GREEN pós-ajuste. A asserção de duplicação (o que o
teste realmente cobre, o B5) não mudou; só a palavra procurada na mensagem.

**5. `docs/TOOLS.md` em `note_delete` descreve o move, o fallback e o erro
`CLOUD_ONLY_FILE`.**
Feito. Acrescentados dois parágrafos na seção `note_delete`: um dizendo que
`to_trash` **move** (`os.Rename`) pela mesma conta de `note_move`, que no mesmo
volume é atômico, e descrevendo o fallback copia-e-remove com o erro do remove
conferido (`FILE_LOCKED`, nomeando o caminho em `.trash/`); e um bloco
**Erros.** documentando `CLOUD_ONLY_FILE` quando o rename é recusado sobre um
placeholder. O bloco de schema JSON não foi tocado — `check_tool_params`
compara-o com o código. Encoding validado:

```
$ python -c "open('C:/Users/jonyd/Projetos/Gobsidian/docs/TOOLS.md',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"
[OK] UTF-8 valido
```

---

## `verify.ps1`

Rodado completo, sem `-SkipCross`/`-SkipNet`, antes do commit. **13 etapas**, e
não 14 — o `CLAUDE.md` diz 14 e o script imprimiu 13; não mudei nem um nem
outro, só registro a divergência. Última linha:

```
[OK] Bateria completa. Pode commitar.
```

Etapas, todas `[OK]`: `go build`; `go test -race`; `go test` (tetos de latência,
sem `-race`); `go vet` windows/linux/darwin; `gofmt`; `golangci-lint`;
`golangci-lint` (linux); `check_net` (RNF-30); `check_tool_params`;
`check_doc_refs`; `check_readme_anchors`.

---

## O que ficou de fora

- **A divergência "14 etapas" do `CLAUDE.md` contra as 13 que `verify.ps1`
  imprime.** Registrada acima, não corrigida: é outra categoria de commit
  (`docs:`), e este commit é de uma categoria só.
- **`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` já aparecia
  modificado no `git status` do despacho, por outra sessão.** Essa modificação
  não é minha. O que **é** meu é a entrada da Task 151 que acrescentei ao fim
  do arquivo (ver "Ledger" abaixo) — acréscimo puro, nada reescrito. Não
  commitei o ledger: as Tasks 146–150 também o deixaram fora do commit de
  código.
- **`MoveNote` continua fazendo sua própria resolução de destino**, enquanto a
  lixeira agora tem `destinoNaLixeira`. Não são a mesma conta (a lixeira tem
  regra de colisão com timestamp que o move não tem), então não há duplicação a
  unificar aqui; só anoto que os dois caminhos convergiram em `moverCorpo` e
  divergem antes dele.

---

## Uma falha intermitente que apareceu DEPOIS do commit, e não é desta tarefa

Rodando `go test ./internal/service` inteiro logo após o commit, com a máquina
sob carga (eu tinha três `go test -v` concorrentes e há outros agentes no mesmo
working tree), o pacote reprovou:

```
--- FAIL: TestRNF04SnippetConcurrencyLimit200 (4.33s)
    search_test.go:647:   limit: 200 concorrente   mediana 17.6504ms    p95 33.1737ms    teto 22ms
    search_test.go:653:   limit: 200 concorrente   rodada 1/3 estourou (33.1737ms > 22ms); repetindo
    search_test.go:647:   limit: 200 concorrente   mediana 20.0658ms    p95 39.3824ms    teto 22ms
    search_test.go:653:   limit: 200 concorrente   rodada 2/3 estourou (39.3824ms > 22ms); repetindo
    search_test.go:647:   limit: 200 concorrente   mediana 19.3054ms    p95 47.0622ms    teto 22ms
    search_test.go:658: p95 de limit: 200 = 47.0622ms excede o teto de 22ms em 3 rodadas seguidas
```

É o teto de latência da RNF-04 em `search_test.go`, sobre `vault_search` —
nenhuma relação com `note_delete`, com a lixeira, com `moverCorpo` ou com
qualquer linha do commit `9fb5500`. Sozinho, numa máquina em repouso, passa com
folga:

```
$ go test ./internal/service -run TestRNF04SnippetConcurrencyLimit200 -timeout 120s -v
    search_test.go:647:   limit: 200 concorrente   mediana 16.2376ms    p95 19.8856ms    teto 22ms
--- PASS: TestRNF04SnippetConcurrencyLimit200 (2.03s)
ok  	github.com/jonyd/gobsidian/internal/service	3.579s
```

E a etapa 3 do `verify.ps1` — exatamente a que roda os tetos de latência sem
`-race` — estava verde no momento do commit. Registro em vez de "corrigir":
não medi nada que sustente uma mudança de teto, e mexer nisso seria outra
categoria de commit. Se alguém vir esse teste vermelho no CI, o dado acima é o
ponto de partida: 19,9 ms com a máquina livre, até 47 ms com ela disputada.

Observação relacionada, do mesmo período: `docs/TOOLS.md` voltou a aparecer
modificado no working tree **depois** do meu commit — é a seção **Retorno** de
`note_move` (o `diffs` do dry-run), edição de outro agente no mesmo tree. Não é
minha e não a toquei; a seção `note_delete` que este commit acrescentou está
intacta em `9fb5500`.

---

## Ledger

O plano `2026-09-02-topografia-e-limpeza` não tem ledger próprio — seu
`progress.md` é um ponteiro para o ledger único do projeto,
`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`. A entrada da Task 151
foi **acrescentada ao fim** desse arquivo, antes deste relatório ser fechado:

```
Task 151: implemented, review PENDING (commit 9fb5500, implementer impl-151).
Trash is now a rename through moverCorpo, and moverCorpo's copy fallback
refuses cloud placeholders (CLOUD_ONLY_FILE). Both new tests proven RED first;
mutation proof EXIT=0 on the guard, manual mutation (copy in place of the
moverCorpo call) reds TestDeleteNoteToTrashMoveSemCopiar. verify.ps1 green,
13 steps, no -Skip flags. Deviation from the brief: destinoNaLixeira returns
vault.CanonicalPath from Resolve instead of a second vault.CanonicalPath(string)
cast in the caller ("uma conta por regra"); rationale in task-151-report.md.
Brief-predicted assertion change: TestDeleteToTrashNaoMenteQuandoORemoveFalha
now matches ".trash" instead of "lixeira". Report:
.superpowers/sdd/2026-09-02-topografia-e-limpeza/task-151-report.md
```

Diz `review PENDING`, não `complete`: a revisão chega depois deste relatório e
quem a registra é o orquestrador. Encoding validado (`[OK] UTF-8 valido`).

`scripts/audit_reports.ps1 151` roda com 13 achados, **todos no ledger e todos
anteriores a esta tarefa** — SHAs das Tasks 4 e 6, relatórios ausentes das Tasks
94–103, e um `[SHA-FANTASMA] deadbee`. Nenhum é sobre a Task 151 nem sobre este
relatório; a seção `=== Relatorios (1) ===` saiu sem achado. Não os corrigi:
são de outra categoria e de outro marco.

---

## `git status --porcelain`

Colado depois do commit:

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
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/review-1a22508..4a06a24.diff
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/review-40521d8..3dfcf8e.diff
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/review-4a06a24..8cfec55.diff
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
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-151-base.txt
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-151-brief.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-151-report.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-152-brief.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-153-brief.md
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-155-brief.md
?? Resume-Claude.ps1
?? "test-vault/test vault/.obsidian/community-plugins (conflito 01M08RSCY410E88M0516VB4K9R 2026-08-17 22h52).json"
?? "test-vault/test vault/.obsidian/core-plugins (conflito 01M08RSCY410E88M0516VB4K9R 2026-08-17 22h52).json"
?? "test-vault/test vault/.obsidian/plugins/gosync/"
?? "test-vault/test vault/A\303\247\303\243o.md"
?? "test-vault/test vault/Pasted image 20260814221454.png"
?? "test-vault/test vault/Sem t\303\255tulo.md"
?? "test-vault/test vault/main.js"
?? "test-vault/test vault/manifest.json"
```

Esse recorte é do instante logo após o commit. Minutos depois `docs/TOOLS.md`
voltou a aparecer como ` M` — edição de outro agente na seção `note_move`, não
minha; ver a seção anterior.

Nenhum arquivo do usuário foi tocado: `test-vault/`, `.claude/skills/` e
`Resume-Claude.ps1` aparecem exatamente como no `git status` do despacho, e o
`git add` foi por caminho explícito (quatro caminhos), nunca `git add -A`. O
único acréscimo à lista é `task-151-report.md`, este arquivo.
