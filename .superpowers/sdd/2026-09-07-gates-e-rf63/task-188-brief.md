### Task 188: `audit_reports.ps1` exige cabeçalho; dívidas fechadas em ESTADO.md

**Files:**
- Modify: `scripts/audit_reports.ps1:47-57` (param + `$SddRoot`), `:113-118` (`$Required`), `:146-150` (checagem)
- Create: `scripts/testdata/gates/sdd-falso/task-1-report.md` (relatório que nega ter evidência), `scripts/testdata/gates/sdd-falso/task-2-report.md` (relatório com cabeçalhos)
- Modify: `scripts/check_gates.ps1` (seção `audit_reports`)
- Modify: `docs/ESTADO.md:606-627` (duas dívidas → corrigidas), `docs/papeis/documentador.md:48`, `docs/papeis/testador.md` (onde cita `audit_reports`; achar com `grep -n audit_reports docs/papeis/testador.md`)

**Interfaces:**
- Consumes: `Caso` e a estrutura de `scripts/check_gates.ps1` da Task 187.
- Produces: `audit_reports.ps1 -SddRoot <dir>` — raiz alternativa para os relatórios e ledgers; padrão continua `.superpowers/sdd`.

- [ ] **Step 1: Fixtures.**

`scripts/testdata/gates/sdd-falso/task-1-report.md` — o relatório que nega ter evidência e, até hoje, passava. Precisa ter mais de 2000 bytes para não disparar `CURTO` (o que se testa aqui é `SECAO-AUSENTE`); preencher com prosa neutra em ASCII:

```
# Task 1 report — fixture: prose that names the sections without having them

## Status
DONE.

## What changed
Only a comment changed, so there is no RED/GREEN cycle and no mutation
proof to paste; the verification was reading the diff. This paragraph
exists to satisfy the old word-match: it contains the words red, green,
mutation and verification without a single section of any of them.

## Filler
(repita um paragrafo neutro de ~150 bytes, em ASCII, ate o arquivo passar de 2000 bytes — contar com `(Get-Item <arquivo>).Length` e colar o numero no relatorio)
```

`scripts/testdata/gates/sdd-falso/task-2-report.md` — o relatório com as quatro seções em cabeçalho, também acima de 2000 bytes:

```
# Task 2 report — fixture: the four sections as headings

## Status
DONE.

### RED — before the fix
(saida colada)

### GREEN — after the fix
(saida colada)

## Mutation proofs
(saida colada)

## Verification
(saida colada)

## Filler
(mesmo preenchimento ate passar de 2000 bytes)
```

- [ ] **Step 2: Casos em `check_gates.ps1` — antes de mudar o auditor.** Acrescentar, dentro do `try`, depois da seção do hook:

```powershell
    Write-Output ""
    Write-Output "=== audit_reports.ps1 ==="
    $Audit = Join-Path $PSScriptRoot 'audit_reports.ps1'
    $SddFalso = Join-Path $Fixtures 'sdd-falso'

    # Conta quantos SECAO-AUSENTE o auditor emite para um relatorio.
    function Secoes-Ausentes([string]$Task) {
        $saida = & pwsh -NoProfile -File $Audit -Task $Task -SddRoot $SddFalso 2>&1 | Out-String
        return ([regex]::Matches($saida, 'SECAO-AUSENTE')).Count.ToString()
    }

    # O bypass conhecido: prosa que NEGA ter RED/GREEN/mutacao satisfazia
    # tres das quatro palavras. Com cabecalho exigido, faltam as quatro.
    Caso -Nome 'prosa que nomeia as secoes sem te-las -> 4 SECAO-AUSENTE' `
        -Esperado '4' -Obtido (Secoes-Ausentes '1')

    Caso -Nome 'quatro secoes em cabecalho -> 0 SECAO-AUSENTE' `
        -Esperado '0' -Obtido (Secoes-Ausentes '2')
```

Mover a definição de `function Secoes-Ausentes` para fora do `try` (junto de `Decisao-Hook`) se o PowerShell reclamar de função dentro de bloco; o que importa é o contrato.

- [ ] **Step 3: RED.** `pwsh -File scripts/check_gates.ps1; echo EXIT=$?`. Esperado: `-SddRoot` não existe ainda → os dois casos novos reprovam (`EXIT=1`); os 9 do hook continuam `[OK]`. Colar.

- [ ] **Step 4: Mudar `scripts/audit_reports.ps1`.**

(a) `param`:
```powershell
param(
    # Audita so um numero de tarefa. Sem isto, audita todos os relatorios.
    [Parameter(Position = 0)]
    [string]$Task,
    # Raiz alternativa dos artefatos. Existe para check_gates.ps1 auditar
    # fixtures em vez do ledger real.
    [string]$SddRoot
)
```
e logo abaixo, no lugar de `$SddRoot = Join-Path $ProjectRoot '.superpowers\sdd'`:
```powershell
if (-not $SddRoot) { $SddRoot = Join-Path $ProjectRoot '.superpowers\sdd' }
```

(b) `$Required`: a seção tem de aparecer num **cabeçalho Markdown** — linha que começa com `#`. Trocar o bloco por:
```powershell
# A secao tem de estar num CABECALHO, nao em qualquer lugar do corpo. Ate
# 2026-09-07 bastava a palavra solta, e "nao ha ciclo RED/GREEN nem prova de
# mutacao" satisfazia tres das quatro — uma negacao explicita passava pelo
# mesmo portao que uma evidencia real (relatorio da Task 184, 2026-09-06).
# Os relatorios reais deste projeto ja usam "### RED — ...", "### GREEN — ...",
# "## Mutation proofs", "## Verification"/"## Gate outputs": o padrao casa o
# cabecalho, em qualquer nivel, em portugues ou ingles.
$Required = @(
    @{ Name = 'TDD/RED';     Pattern = '(?im)^#{1,6}\s.*(^|\W)red(\W|$)' },
    @{ Name = 'TDD/GREEN';   Pattern = '(?im)^#{1,6}\s.*(^|\W)green(\W|$)' },
    @{ Name = 'Mutacao';     Pattern = '(?im)^#{1,6}\s.*muta' },
    @{ Name = 'Verificacao'; Pattern = '(?im)^#{1,6}\s.*(verifica|verification|gate)' }
)
```
`(?m)` faz `^` casar início de linha em `$Body` (que já é `$Lines -join "`n"`). Nada mais muda no laço.

- [ ] **Step 5: GREEN.** `pwsh -File scripts/check_gates.ps1; echo EXIT=$?` → 11 `[OK]`, `EXIT=0`. Colar.

- [ ] **Step 6: Prova de mutação.** Reverter a mão o padrão de `Mutacao` para `'(?i)muta'` (o antigo), rodar `check_gates.ps1`, confirmar que o caso "prosa que nomeia as secoes" obtém `3` em vez de `4` e reprova, colar, restaurar, rodar, colar `EXIT=0`.

- [ ] **Step 7: Efeito nos relatórios reais — medir, não supor.** `pwsh -File scripts/audit_reports.ps1 2>&1 | grep -c 'SECAO-AUSENTE'` antes (com o script antigo — rodar ANTES do Step 4 e guardar) e depois. Colar os dois números no relatório e em `docs/ESTADO.md` (Step 8). Um aumento é esperado: relatórios que nunca tiveram as seções passam a ser sinalizados. Não "consertar" relatórios antigos — eles são histórico.

- [ ] **Step 8: Docs.**
  - `docs/ESTADO.md:606-615` (hook): trocar "**não corrigido** neste marco" por "**corrigido em 2026-09-07** (Task 187): o hook lê a mensagem de `-m`/`-F`; `scripts/check_gates.ps1` prova que o comentário de shell é recusado". Manter o mecanismo descrito.
  - `docs/ESTADO.md:616-627` (audit): trocar "**não corrigido**" por "**corrigido em 2026-09-07** (Task 188): seção só conta em cabeçalho Markdown; `check_gates.ps1` prova que a prosa que nega é sinalizada. Efeito medido nos relatórios reais: `<antes>` → `<depois>` `SECAO-AUSENTE`" com os números do Step 7.
  - `docs/papeis/documentador.md:48` e, **se** `grep -n audit_reports docs/papeis/testador.md` devolver algo, `testador.md` (em 2026-09-07 devolvia nada — então só `documentador.md`): uma frase — as quatro seções (RED, GREEN, mutação, verificação) precisam estar em **cabeçalho**; prosa não conta.
  - UTF-8 nos três.

- [ ] **Step 9: Gate completo, sem `-Skip`.** `pwsh -File scripts/verify.ps1; echo EXIT=$?` — colar as últimas 8 linhas. Esperado `EXIT=0`, 15 etapas.

- [ ] **Step 10: Commit** (mensagem em `.superpowers/sdd/2026-09-07-gates-e-rf63/commit-188.txt`):

```
fix(gates): audit_reports counts a section only when it is a heading

The four required sections of a task report were matched as bare words
anywhere in the body, so a sentence denying that RED, GREEN or a mutation
proof existed satisfied three of them. A section now counts only in a
Markdown heading, which is how every real report here already writes
them. check_gates.ps1 carries the denying report as a fixture and fails
if it passes again.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj
```

```bash
git add scripts/audit_reports.ps1 scripts/check_gates.ps1 scripts/testdata/gates/sdd-falso/task-1-report.md scripts/testdata/gates/sdd-falso/task-2-report.md docs/ESTADO.md docs/papeis/documentador.md docs/papeis/testador.md
git commit -F .superpowers/sdd/2026-09-07-gates-e-rf63/commit-188.txt
```

---

## Self-review

- **Cobertura:** item 1 → Task 186 (RF-63); item 2 → nenhuma task (decisão: manter; já registrado em `ESTADO.md:220` com a medição); item 3 → Task 186 (medição registrada, parqueado); item 4a → Task 187; item 4b → Task 188.
- **Placeholders:** o preenchimento das fixtures acima de 2000 bytes é deliberadamente "prosa neutra" — o conteúdo não importa, o tamanho sim, e a task cola o tamanho medido.
- **Consistência:** `Caso`, `Decisao-Hook`, `-Simular -Comando -EmStage`, `-SddRoot` têm o mesmo nome nas três tasks. `check_gates.ps1` é criado na 187 e estendido na 188; a 188 depende da 187.
- **Sem Go:** nenhuma task toca `.go`; `verify.ps1` continua rodando o resto.
