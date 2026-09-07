### Task 187: `pre_commit_docs.ps1` lê a mensagem; `check_gates.ps1` prova que o bypass é recusado

**Files:**
- Modify: `scripts/pre_commit_docs.ps1`
- Create: `scripts/check_gates.ps1`
- Create: `scripts/testdata/gates/msg-com-sem-doc.txt`, `scripts/testdata/gates/msg-sem-escotilha.txt`
- Modify: `scripts/verify.ps1:253` (após `check_readme_anchors`)
- Modify: `CLAUDE.md:207` e a regra da escotilha; `docs/OPERACAO.md` (seção que lista as etapas de `verify.ps1`, localizar com `grep -n 'check_readme_anchors' docs/OPERACAO.md`); `docs/papeis/documentador.md:143`

**Interfaces:**
- Produces: `scripts/pre_commit_docs.ps1` com parâmetros `-Simular`, `-Comando <string>`, `-EmStage <string[]>`. Em modo `-Simular`, `-Comando` é a linha de comando que o hook receberia em `tool_input.command` e `-EmStage` substitui `git diff --cached --name-only`. O parâmetro antigo `-Mensagem` **some** (era a linha de comando com nome errado; ninguém o chama — confirmar com `git grep -n 'pre_commit_docs.ps1 -' -- '*.md' '*.ps1' '*.json'`).
- Produces: função interna `Extrair-Mensagem([string]$comando)` → string com a concatenação de todos os `-m`/`--message=` (entre aspas duplas, simples ou sem aspas) e do conteúdo do arquivo de `-F <arquivo>`/`--file=<arquivo>` quando existir; string vazia quando nenhum dos dois aparece ou o arquivo não existe.
- Produces: `scripts/check_gates.ps1`, que a Task 188 estende com a seção `audit_reports`. Contrato: `Caso -Nome <string> -Esperado <string> -Obtido <string>` imprime `[OK] <nome>` ou `[!] <nome>: esperado <e>, obtido <o>` e acumula; no fim imprime `[OK] check_gates: N casos` e sai 0, ou `[!] check_gates: K de N casos reprovados` e sai 1.

- [ ] **Step 1: Escrever `scripts/check_gates.ps1` com os casos do hook — antes de mudar o hook.**

```powershell
#Requires -Version 7.0
<#
.SYNOPSIS
    Prova que os gates recusam o bypass que ja passou por eles.

.DESCRIPTION
    Dois gates deste projeto casavam a PALAVRA e nao a evidencia, e foram
    contornados sem intencao de contornar (docs/ESTADO.md, dividas de
    2026-09-06):

      - pre_commit_docs.ps1 procurava [sem-doc] na LINHA DE COMANDO do git
        commit, e um comentario de shell (`# [sem-doc]`) o satisfazia com a
        mensagem dizendo outra coisa.
      - audit_reports.ps1 procurava as secoes obrigatorias do relatorio por
        palavra solta no corpo, e "nao ha ciclo RED/GREEN nem prova de
        mutacao" satisfazia tres delas.

    Cada caso aqui e um bypass conhecido com a decisao que o gate DEVE tomar.
    Roda como etapa do verify.ps1: gate que volta a aceitar o bypass reprova
    o gate inteiro.

.NOTES
    Saida em ASCII puro. Sai 1 se qualquer caso reprovar.
#>
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$Fixtures = Join-Path $PSScriptRoot 'testdata\gates'
$Hook = Join-Path $PSScriptRoot 'pre_commit_docs.ps1'

$script:Total = 0
$script:Reprovados = 0

function Caso {
    param([string]$Nome, [string]$Esperado, [string]$Obtido)
    $script:Total++
    if ($Esperado -eq $Obtido) {
        Write-Output "[OK] $Nome"
    }
    else {
        $script:Reprovados++
        Write-Output "[!] ${Nome}: esperado '$Esperado', obtido '$Obtido'"
    }
}

# Decisao do hook em modo simulado. O hook imprime o JSON de hook em stdout;
# so a decisao interessa aqui.
function Decisao-Hook {
    param([string]$Comando, [string[]]$EmStage)
    $json = & pwsh -NoProfile -File $Hook -Simular -Comando $Comando -EmStage $EmStage
    return ($json | ConvertFrom-Json).hookSpecificOutput.permissionDecision
}

Push-Location $ProjectRoot
try {
    Write-Output "=== pre_commit_docs.ps1 ==="
    $go = @('internal/index/resolve.go')
    $goEDoc = @('internal/index/resolve.go', 'docs/TOOLS.md')
    $msgSem = 'scripts/testdata/gates/msg-sem-escotilha.txt'
    $msgCom = 'scripts/testdata/gates/msg-com-sem-doc.txt'

    # O bypass real de 2026-09-06: a escotilha num comentario de shell, a
    # mensagem (via -F) sem ela.
    Caso -Nome 'escotilha em comentario de shell, mensagem sem ela -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook "git commit -F $msgSem # [sem-doc]" $go)

    Caso -Nome 'escotilha no arquivo de -F -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook "git commit -F $msgCom" $go)

    Caso -Nome 'escotilha em -m entre aspas duplas -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit -m "fix: x [sem-doc]"' $go)

    Caso -Nome 'escotilha em -m entre aspas simples -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook "git commit -m 'fix: x [sem-doc]'" $go)

    Caso -Nome 'escotilha fora das aspas de -m -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit -m "fix: x" # [sem-doc]' $go)

    Caso -Nome '-m sem escotilha, .go sem doc -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit -m "fix: x"' $go)

    Caso -Nome '-m sem escotilha, .go com doc -> allow' `
        -Esperado 'allow' -Obtido (Decisao-Hook 'git commit -m "fix: x"' $goEDoc)

    Caso -Nome '-F arquivo inexistente, escotilha no comando -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit -F nao-existe.txt [sem-doc]' $go)

    Caso -Nome 'sem -m nem -F (editor), .go sem doc -> deny' `
        -Esperado 'deny' -Obtido (Decisao-Hook 'git commit' $go)
}
finally {
    Pop-Location
}

Write-Output ""
if ($script:Reprovados -gt 0) {
    Write-Output "[!] check_gates: $script:Reprovados de $script:Total casos reprovados"
    exit 1
}
Write-Output "[OK] check_gates: $script:Total casos"
exit 0
```

Fixtures:

`scripts/testdata/gates/msg-sem-escotilha.txt`:
```
fix(index): a message that does not use the hatch

Body without the tag.
```

`scripts/testdata/gates/msg-com-sem-doc.txt`:
```
test(index): tighten an assertion [sem-doc]

The contract did not change; only the test did.
```

- [ ] **Step 2: Rodar e ver falhar (RED).** `pwsh -File scripts/check_gates.ps1; echo EXIT=$?`. Esperado: o hook atual não conhece `-Comando`/`-EmStage`, então `pwsh` reprova com erro de parâmetro, `ConvertFrom-Json` falha ou a decisão vem vazia — todos os 9 casos `[!]`, `EXIT=1`. Colar a saída no relatório como RED. Se por acaso algum caso passar, o caso está errado; corrigir o caso, não o hook.

- [ ] **Step 3: Mudar `scripts/pre_commit_docs.ps1`.**

(a) Bloco `param`:
```powershell
param(
    # Para teste: pula a leitura de stdin. -Comando e a linha que o hook
    # receberia em tool_input.command; -EmStage substitui `git diff --cached`.
    [switch]$Simular,
    [string]$Comando = "",
    [string[]]$EmStage = @()
)
```

(b) Depois de `function Emitir`, a extração da mensagem:
```powershell
# A escotilha vale na MENSAGEM, nunca na linha de comando. Ate 2026-09-07 o
# hook procurava [sem-doc] no texto do comando inteiro, e um comentario de
# shell (`git commit -F msg.txt # [sem-doc]`) o satisfazia com a mensagem
# dizendo outra coisa — foi assim que os commits de 2026-09-06 passaram, por
# instrucao do brief, sem que ninguem quisesse burlar nada. Gate que le a
# escotilha fora do lugar onde ela fica visivel no historico nao gateia.
#
# Le -m/--message= (aspas duplas, simples ou sem aspas, repetidos) e o
# arquivo de -F/--file=. Sem nenhum dos dois — commit que abriria editor — a
# mensagem e desconhecida e a escotilha nao vale.
function Extrair-Mensagem([string]$linha) {
    $partes = [System.Collections.Generic.List[string]]::new()
    $reM = '(?:-m|--message)(?:=|\s+)(?:"([^"]*)"|''([^'']*)''|(\S+))'
    foreach ($m in [regex]::Matches($linha, $reM)) {
        $texto = @($m.Groups[1].Value, $m.Groups[2].Value, $m.Groups[3].Value) | Where-Object { $_ } | Select-Object -First 1
        if ($texto) { $partes.Add($texto) }
    }
    $reF = '(?:-F|--file)(?:=|\s+)(?:"([^"]*)"|''([^'']*)''|(\S+))'
    foreach ($m in [regex]::Matches($linha, $reF)) {
        $arquivo = @($m.Groups[1].Value, $m.Groups[2].Value, $m.Groups[3].Value) | Where-Object { $_ } | Select-Object -First 1
        if ($arquivo -and (Test-Path -LiteralPath $arquivo -PathType Leaf)) {
            $partes.Add((Get-Content -LiteralPath $arquivo -Raw -Encoding utf8))
        }
    }
    return ($partes -join "`n")
}
```

(c) No `try`: substituir `$comando = $Mensagem` por `$comando = $Comando`; substituir o teste `if ($comando -match '\[sem-doc\]')` por:
```powershell
    $mensagem = Extrair-Mensagem $comando
    if ($mensagem -match '\[sem-doc\]') {
        Emitir "allow" "escotilha [sem-doc] na mensagem do commit"
    }
```

(d) Stage: substituir `$emStage = @(git diff --cached --name-only 2>$null | Where-Object { $_ })` por:
```powershell
    $emStage = if ($Simular) { @($EmStage) } else { @(git diff --cached --name-only 2>$null | Where-Object { $_ }) }
```

(e) No texto de `deny`, trocar a linha `"teste que nao muda contrato), inclua [sem-doc] na mensagem do commit."` por `"teste que nao muda contrato), inclua [sem-doc] na MENSAGEM do commit (-m ou"` seguida de `"arquivo de -F). Na linha de comando fora da mensagem ele nao vale."`.

(f) No `.DESCRIPTION`, parágrafo ESCOTILHA: acrescentar ao fim: `A escotilha vale na mensagem (-m ou arquivo de -F). Ate 2026-09-07 valia em qualquer lugar da linha de comando, inclusive num comentario de shell, e por isso nao valia nada.`

- [ ] **Step 4: Rodar e ver passar (GREEN).** `pwsh -File scripts/check_gates.ps1; echo EXIT=$?`. Esperado: 9 `[OK]`, `[OK] check_gates: 9 casos`, `EXIT=0`. Colar a saída.

- [ ] **Step 5: Prova de mutação, no passado, com saída colada.** Reverter temporariamente (a mão, editando — nunca `git checkout`) a linha `$mensagem = Extrair-Mensagem $comando` para `$mensagem = $comando`, rodar `check_gates.ps1`, confirmar que os casos "escotilha em comentario de shell" e "escotilha fora das aspas" e "-F inexistente" reprovam (`EXIT=1`), colar a saída, restaurar a linha, rodar de novo, colar `EXIT=0`.

- [ ] **Step 6: Wire em `verify.ps1`.** Depois de `Invoke-Step "check_readme_anchors" {...}` (linha 253), acrescentar:

```powershell
# Os gates que casam a palavra em vez da evidencia foram contornados sem
# intencao (ESTADO.md, 2026-09-06). check_gates prova, a cada rodada, que o
# bypass conhecido continua recusado.
Invoke-Step "check_gates" { & (Join-Path $PSScriptRoot "check_gates.ps1") }
```

- [ ] **Step 7: Docs.**
  - `CLAUDE.md:207`: `# o gate: 14 etapas, para no primeiro erro` → `# o gate: 15 etapas, para no primeiro erro`. Na frase que lista o que ele cobre ("cobre build, `go test -race`, … `check_doc_refs` e `check_readme_anchors`"), acrescentar `e check_gates` ao fim da lista.
  - `docs/OPERACAO.md`: onde as etapas do `verify.ps1` estão listadas (achar com `grep -n 'check_readme_anchors' docs/OPERACAO.md`), acrescentar `check_gates` com uma linha: "prova que `pre_commit_docs.ps1` e `audit_reports.ps1` recusam os bypasses conhecidos". Se `OPERACAO.md` não listar etapas, registrar "não lista" no relatório e não inventar seção.
  - `docs/papeis/documentador.md:143`: onde descreve o hook, acrescentar que `[sem-doc]` vale na mensagem (`-m`/`-F`), não na linha de comando.
  - Validar UTF-8 nos três.

- [ ] **Step 8: Gate.** `pwsh -File scripts/verify.ps1 -SkipCross -SkipNet; echo EXIT=$?` — esperado `EXIT=0` com 15 etapas listadas; colar as últimas 6 linhas.

- [ ] **Step 9: Commit** (mensagem em `.superpowers/sdd/2026-09-07-gates-e-rf63/commit-187.txt`):

```
fix(gates): pre_commit_docs reads the hatch from the commit message, not the command line

The hook looked for [sem-doc] anywhere in the git commit command, so a
shell comment satisfied it while the message said something else. It now
extracts the message from -m/--message= and from the file behind
-F/--file=, and the hatch counts only there.

check_gates.ps1 runs the known bypasses against the hook and fails when
one is accepted again; verify.ps1 runs it as its fifteenth step.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01P5wkw6PAdBFzF3uB1w1jNj
```

```bash
git add scripts/pre_commit_docs.ps1 scripts/check_gates.ps1 scripts/testdata/gates/msg-sem-escotilha.txt scripts/testdata/gates/msg-com-sem-doc.txt scripts/verify.ps1 CLAUDE.md docs/OPERACAO.md docs/papeis/documentador.md
git commit -F .superpowers/sdd/2026-09-07-gates-e-rf63/commit-187.txt
```

O hook novo vai rodar neste commit: não há `.go` em stage, então é `allow` por "nenhum .go de producao em stage". Colar o JSON que o hook devolveu, se visível.

---

