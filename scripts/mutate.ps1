#Requires -Version 7.0
<#
.SYNOPSIS
    Aplica UMA mutacao, roda UM teste, e restaura. O codigo de saida diz se a
    regra estava verificada ou apenas escrita.

.DESCRIPTION
    Este projeto exige prova de mutacao por regra: apague a regra, confirme que
    um teste reprova nomeando a falha, restaure. Tres coisas davam errado quando
    isso era feito a mao, e as tres ja aconteceram aqui:

    1. A ancora nao casava e a "mutacao" nao mutava nada. `str.replace` que nao
       casa nao falha — segue em silencio, e a suite verde vira prova de que a
       regra esta coberta quando na verdade ela nunca foi tocada. Aqui a ancora
       precisa ocorrer EXATAMENTE uma vez, ou o script para antes de escrever.

    2. O arquivo nao voltava byte a byte. Ler com encoding de texto e gravar em
       modo texto converte o arquivo inteiro para CRLF no Windows, e o gofmt
       reprova um .go que estava perfeitamente formatado — ja custou dois
       commits. Aqui o backup e a restauracao sao em bytes crus, e o hash do
       arquivo restaurado e conferido contra o original.

    3. A prova era escrita no condicional: "se removermos X, o teste falha".
       Isso nao e prova, e uma delas estava factualmente errada — o reconciliador
       foi removido e a suite continuou verde. Aqui a saida real do `go test` e
       impressa para ser colada no relatorio.

    CODIGO DE SAIDA, e a parte que importa:
      0  -> o teste REPROVOU sob mutacao. A regra esta verificada.
      1  -> o teste PASSOU sob mutacao. A regra esta escrita, nao verificada.
      2  -> o script nao conseguiu rodar (ancora ambigua, arquivo ausente).

    Teste que sobrevive a mutacao e o resultado ruim, entao ele e o que falha.

.EXAMPLE
    pwsh -File scripts/mutate.ps1 `
        -Path internal/watcher/apply.go `
        -Anchor 'if n, ok := idx.Get(path); ok {' `
        -Replacement 'if n, ok := idx.Get(path); ok && false {' `
        -Test TestApply -Package ./internal/watcher/

.EXAMPLE
    # Mutacao que apaga uma linha inteira: Replacement vazio.
    pwsh -File scripts/mutate.ps1 -Path internal/index/update.go `
        -Anchor "`tbacklinks := idx.Backlinks(oldPath)`n" -Replacement '' `
        -Test TestCorrelateRenames_ReportsBacklinkCandidates -Package ./internal/watcher/
#>
[CmdletBinding()]
param(
    # Caminho do arquivo a mutar, relativo a raiz do projeto ou absoluto.
    [Parameter(Mandatory)]
    [string]$Path,

    # Texto exato a substituir. Precisa ocorrer exatamente uma vez.
    [Parameter(Mandatory)]
    [string]$Anchor,

    # Texto que entra no lugar. Vazio apaga a ancora.
    [Parameter()]
    [string]$Replacement = '',

    # Nome do teste que DEVE reprovar. Passado ao -run do go test.
    [Parameter(Mandatory)]
    [string]$Test,

    # Pacote onde o teste vive.
    [Parameter()]
    [string]$Package = './...',

    # Roda sem -race. Dois casos legitimos:
    #
    #   1. A mutacao provoca deadlock conhecido.
    #   2. O teste cobra um TETO DE TEMPO. Esses testes guardam a assercao atras
    #      de `!raceEnabled`, porque o detector multiplica a latencia por 2 a 6 e
    #      o numero deixa de ser comparavel. Sob -race a assercao e compilada
    #      fora, e o teste NAO CONSEGUE reprovar — a mutacao volta "[!] o teste
    #      PASSOU" com a regra intacta. Rodar esse caso com -race ja produziu, na
    #      Task 72, um relatorio com saida de FALHA que -race torna impossivel.
    [switch]$NoRace
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$Full = if ([System.IO.Path]::IsPathRooted($Path)) { $Path } else { Join-Path $ProjectRoot $Path }

if (-not (Test-Path $Full -PathType Leaf)) {
    Write-Output "[!] Arquivo nao encontrado: $Full"
    exit 2
}

# Bytes crus dos dois lados. Nada de encoding de texto no backup: o objetivo e
# devolver o arquivo identico, nao equivalente.
$Original = [System.IO.File]::ReadAllBytes($Full)
$OriginalHash = [System.BitConverter]::ToString(
    [System.Security.Cryptography.SHA256]::HashData($Original))

$Text = [System.Text.Encoding]::UTF8.GetString($Original)

# Contagem antes de escrever. Zero ocorrencias e a falha silenciosa que este
# script existe para tornar barulhenta; mais de uma torna a mutacao ambigua e
# o revisor nao saberia qual foi exercitada.
$Occurrences = ([regex]::Matches($Text, [regex]::Escape($Anchor))).Count
if ($Occurrences -ne 1) {
    Write-Output "[!] Ancora ocorre $Occurrences vez(es) em $Path; precisa ocorrer exatamente 1."
    if ($Occurrences -eq 0) {
        Write-Output "    O texto nao casou. Confira espacos, tabulacao e acentos — copie do arquivo, nao digite."
    } else {
        Write-Output "    Amplie a ancora com as linhas vizinhas ate ela ficar unica."
    }
    exit 2
}

# [string[]] e obrigatorio. `$x = if (...) { @() } else { @('-race') }` desenrola
# o array de um elemento para escalar, e `@x` sobre uma string a espalha caractere
# a caractere: o go test recebe pacotes "r", "a", "c", "e", falha no setup, e o
# script conta essa falha como "a regra esta verificada". Falso PASS produzido
# pelo proprio script que existe para impedir falso PASS.
[string[]]$RaceFlag = if ($NoRace) { @() } else { @('-race') }

Write-Output "[...] Mutando $Path"
Write-Output "      - $($Anchor  -replace "`r?`n", '\n')"
Write-Output "      + $($Replacement -replace "`r?`n", '\n')"

$TestExit = $null
try {
    $Mutated = $Text.Replace($Anchor, $Replacement)
    [System.IO.File]::WriteAllBytes($Full, [System.Text.Encoding]::UTF8.GetBytes($Mutated))

    Write-Output ""
    Write-Output "[...] go test $RaceFlag -run $Test $Package"
    Write-Output "----------------------------------------------------------------------"
    # A saida vai inteira para o console: ela E a evidencia que vai para o
    # relatorio. Resumir aqui recriaria o problema que o script resolve.
    # Captura primeiro, imprime depois. `Write-Output` dentro de um
    # ForEach-Object cujo resultado esta sendo atribuido nao imprime nada: os
    # dois vao para o mesmo pipeline e a variavel engole a saida inteira.
    $TestOutput = & go test @RaceFlag -run $Test $Package 2>&1
    $TestExit = $LASTEXITCODE
    $TestOutput | ForEach-Object { Write-Output $_ }
    Write-Output "----------------------------------------------------------------------"

    # Um teste que nao COMPILA tambem "reprova", e contar isso como cobertura e
    # o mesmo erro pelo outro lado: a mutacao pode ter quebrado o build em vez de
    # ser pega por uma assercao. Idem para teste que nao existe — `-run` sem
    # correspondencia sai com 0 e nao prova nada.
    $Joined = ($TestOutput | Out-String)
    $Inconclusive =
        $Joined -match '\[setup failed\]' -or
        $Joined -match '\[build failed\]' -or
        $Joined -match 'no test files' -or
        $Joined -match 'is not in std' -or
        $Joined -match 'undefined:' -or
        $Joined -match 'testing: warning: no tests to run'
}
finally {
    # Restaura sempre: Ctrl+C, excecao, teste travado. Um arquivo deixado mutado
    # e pior que a mutacao nao ter rodado, porque o proximo comando verde mente.
    [System.IO.File]::WriteAllBytes($Full, $Original)

    $RestoredHash = [System.BitConverter]::ToString(
        [System.Security.Cryptography.SHA256]::HashData(
            [System.IO.File]::ReadAllBytes($Full)))

    if ($RestoredHash -ne $OriginalHash) {
        Write-Output ""
        Write-Output "[!] RESTAURACAO FALHOU. $Path nao voltou byte a byte."
        Write-Output "    antes:  $OriginalHash"
        Write-Output "    depois: $RestoredHash"
        Write-Output "    Rode 'git diff -- $Path' e conserte a mao ANTES de qualquer commit."
        exit 2
    }
    Write-Output "[OK] $Path restaurado byte a byte (SHA-256 confere)."
}

Write-Output ""
if ($Inconclusive) {
    Write-Output "[!] INCONCLUSIVO: o teste nao chegou a rodar como assercao."
    Write-Output "    A saida traz falha de build/setup, ou nenhum teste casou com -run $Test."
    Write-Output "    Uma mutacao que quebra a compilacao nao prova cobertura: ela prova que"
    Write-Output "    o codigo nao compila. Ajuste a mutacao para ser valida em Go, ou confira"
    Write-Output "    o nome do teste e o pacote."
    exit 2
}

if ($TestExit -eq 0) {
    Write-Output "[!] O teste PASSOU com a regra mutada."
    Write-Output "    $Test nao consegue reprovar sem essa regra: ela esta escrita, nao verificada."
    Write-Output "    Escreva o teste que falta antes de dizer que a regra tem cobertura."
    exit 1
}

Write-Output "[OK] O teste REPROVOU com a regra mutada — a regra esta verificada."
Write-Output "[i] Cole a saida acima no relatorio. Prova de mutacao no condicional"
Write-Output "    ('se removermos X, o teste falha') nao conta: uma delas ja estava errada."
exit 0
