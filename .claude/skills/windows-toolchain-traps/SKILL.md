---
name: windows-toolchain-traps
description: Diagnose tooling failures specific to this Windows repo - a wrapper script that cannot find a file that exists, artifacts silently missing from git, a base commit that goes stale every time you fix it, PowerShell flags exploding into single characters, and script output vanishing. Use when a script fails with a path error, when git ignores something you expect to be tracked, when a PowerShell helper behaves impossibly, or before writing any new script under scripts/.
---

# Armadilhas de toolchain neste repositório

Todas aconteceram aqui. Cada uma produz um sintoma que aponta para o lugar errado, que é o motivo de estarem escritas.

## 1. `bash` no PATH é o do WSL, e ele não enxerga `C:/`

**Sintoma:** um script existe, `Test-Path` confirma, e o bash diz que não existe.

```
/bin/bash: C:/Users/jonyd/.claude/plugins/.../task-brief: No such file or directory
```

**Causa:** `Get-Command bash` resolve, nesta máquina, para `WindowsApps\bash.exe` — o bash do WSL, que quer `/mnt/c/Users/...`, não `C:/Users/...`. A mensagem culpa o arquivo; o culpado é qual bash respondeu.

**Antes disso ainda há uma segunda camada:** passar `C:\Users\...` para qualquer bash faz as contrabarras serem consumidas como escape, e o caminho vira `C:Usersjonyd...`.

**Correção**, já aplicada em `scripts/sdd.ps1`:

```powershell
function ConvertTo-BashPath([string]$Path) { return $Path -replace '\\', '/' }

function Get-GitBash {
    $GitCmd = Get-Command git -ErrorAction SilentlyContinue
    if ($GitCmd) {
        $Candidate = Join-Path (Split-Path (Split-Path $GitCmd.Source)) "bin\bash.exe"
        if (Test-Path $Candidate) { return $Candidate }
    }
    return "bash"
}
```

**Diagnóstico rápido:** `Get-Command bash -All | ForEach-Object { $_.Source }`. Se o primeiro for `WindowsApps` ou `System32`, é o WSL.

## 2. `.gitignore` aninhado não é cancelável pelo pai, e o plugin o recria

**Sintoma:** artefatos que você espera versionados não aparecem em `git status`, e remover a linha do `.gitignore` da raiz não resolve.

**Causa:** havia um segundo `.gitignore` em `.superpowers/sdd/` com `*`. Um `.gitignore` mais fundo tem precedência, e **negação no diretório pai não o cancela** — não existe padrão no `.gitignore` da raiz capaz de desfazer um `*` aninhado.

**A parte que faz voltar:** `sdd-workspace` do plugin superpowers faz `printf '*\n' > "$base/.gitignore"` (linha 39). Apagar sem guarda volta ao estado anterior na próxima chamada.

**Correção:** `scripts/sdd.ps1` apaga o arquivo em toda invocação (`Remove-PluginGitignore`).

**A parte traiçoeira:** quando o arquivo volta, os artefatos **já rastreados continuam rastreados** — só os novos somem. Falha parcial não parece falha.

**Diagnóstico:** `git check-ignore -v <caminho>` diz qual arquivo e qual linha decidiram.

## 3. Arquivo de base versionado recursa

**Sintoma:** você grava a base da tarefa, commita, e a base fica um commit atrás. Regrava, commita, mesma coisa.

**Causa:** `.superpowers/` passou a ser versionado, então `task-N-base.txt` é rastreado. Commitá-lo move o HEAD, e a base que acabou de ser gravada aponta para antes desse commit.

**Correção: deixe o arquivo sujo.** O primeiro commit da tarefa o recolhe, e aí `base..HEAD` é exatamente o trabalho da tarefa. `sdd.ps1 base` imprime isso, porque a reação natural a uma árvore suja é commitar.

Uma árvore suja por esse arquivo é o estado **correto** antes de despachar uma tarefa.

## 4. Array de um elemento vira escalar, e `@splat` explode a string

**Sintoma:** `go test` reclama de pacotes chamados `r`, `a`, `c`, `e`.

```
package r is not in std (C:\Program Files\Go\src\r)
package a is not in std ...
```

**Causa:** `$x = if ($c) { @() } else { @('-race') }` desenrola o array de um elemento para **escalar**. `@x` sobre uma string a espalha caractere a caractere.

**Correção:** tipar explicitamente.

```powershell
[string[]]$RaceFlag = if ($NoRace) { @() } else { @('-race') }
```

**Por que isto importa mais que um bug de sintaxe:** aconteceu dentro de `scripts/mutate.ps1`, o script cuja função é impedir falso PASS. O teste nunca rodou, o `go test` saiu diferente de zero por causa do setup quebrado, e o script reportou "a regra está verificada". Falso PASS produzido pela ferramenta anti-falso-PASS. Por isso `mutate.ps1` hoje trata falha de build como inconclusivo, não como cobertura.

## 5. `Write-Output` dentro de pipeline atribuído não imprime

**Sintoma:** o bloco de saída do comando aparece vazio, mas o comando rodou.

```powershell
# ERRADO: os dois vao para o mesmo pipeline, a variavel engole tudo
$out = & go test ... | ForEach-Object { Write-Output $_; $_ }

# CERTO: captura primeiro, imprime depois
$out = & go test ... 2>&1
$exit = $LASTEXITCODE
$out | ForEach-Object { Write-Output $_ }
```

## 6. Regras já pagas que continuam valendo

- **`$LASTEXITCODE` some ao passar comando nativo por cmdlet.** `go list -m | Select-Object -First 1` sob `Set-StrictMode -Version Latest` deixa `$LASTEXITCODE` **não definido** — não `$null`, não definido — e ler dispara `InvalidOperation`. Atribua direto: `$x = go list -m 2>$null`.
- **Script Python que edita arquivo versionado precisa de `newline=""` na leitura E na escrita.** Modo texto converte o arquivo inteiro para CRLF e o `gofmt` reprova. Custou dois commits. Em PowerShell, o equivalente seguro é `[System.IO.File]::ReadAllBytes` / `WriteAllBytes`.
- **`str.replace` que não casa não falha.** Toda edição por script leva `assert` do texto-âncora antes de substituir, e conferência do resultado no disco depois.
- **Here-string do PowerShell (`@'...'@`) não funciona na ferramenta Bash.** O `@` entra no texto. Para mensagem de commit multilinha ali, use heredoc (`-F - <<'EOF'`). Já produziu um commit com `@` no assunto nesta sessão.
- **Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`. Console em CP-850 renderiza o resto como lixo.
- **`.gitattributes` congela o que não pode ser normalizado.** `*.diff -text` existe porque `* text=auto` reescreveria os fins de linha dos pacotes de revisão, e um diff que não é mais byte a byte o que o revisor leu não é evidência de nada.

## 7. Antes de escrever qualquer script novo em `scripts/`

- Rode-o contra um caso que **deve** falhar e confirme que ele falha. Um verificador que não consegue reprovar é o mesmo defeito de um teste que não consegue reprovar — e este projeto já produziu os dois.
- Se ele tem código de saída com significado, teste os três: sucesso, falha, e inconclusivo.
- Se ele reporta achados, rode contra o repositório inteiro e olhe o volume. Um checador que dispara em prosa legítima vira ruído e para de ser lido, que é pior que não existir.
