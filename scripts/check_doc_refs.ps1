#Requires -Version 7.0
<#
.SYNOPSIS
    Localiza referencias a artefatos de codigo em documentacao que nao existem
    no repositorio.

.DESCRIPTION
    Procura em docs/*.md e README.md por tokens entre crases que parecem
    identificadores de codigo — nomes de arquivo, funcoes, constantes — e
    verifica se existem em algum arquivo .go do repositorio.

    Padroes:
    - Nome de arquivo terminado em .gob ou .go
    - Token snake_case (ex: create_dirs, include_broken)
    - Identificador CamelCase seguido de parenteses (ex: MoveNote())

    CODIGO DE SAIDA:
      0 -> nenhum achado
      1 -> ha achados; a lista esta na saida
      2 -> o script nao conseguiu rodar

.EXAMPLE
    pwsh -File scripts/check_doc_refs.ps1
#>
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot

# Caminho para buscar .go files
$GoFiles = @(Get-ChildItem -Path $ProjectRoot -Filter '*.go' -Recurse | Where-Object {
    -not $_.FullName.Contains('\.git')
} | ForEach-Object { $_.FullName })

if ($GoFiles.Count -eq 0) {
    Write-Output "[!] Nenhum arquivo .go encontrado em $ProjectRoot"
    exit 2
}

# Carrega todo conteudo dos .go files uma unica vez
$AllGoContent = @{}
foreach ($File in $GoFiles) {
    $AllGoContent[$File.ToLower()] = Get-Content -Path $File -Raw -Encoding utf8
}
$GoFilesJoined = ($AllGoContent.Values) -join "`n"

# Extrai lista de nomes de arquivo .go existentes (basename e path relativo)
$ExistingGoFiles = @{}
$ExistingGobFiles = @{}
foreach ($File in $GoFiles) {
    $Basename = [System.IO.Path]::GetFileName($File)
    $Relative = $File.Replace($ProjectRoot, '').TrimStart('\', '/').Replace('\', '/')
    $ExistingGoFiles[$Basename.ToLower()] = $true
    $ExistingGoFiles[$Relative.ToLower()] = $true
}
# Adiciona .gob files se encontrados
Get-ChildItem -Path $ProjectRoot -Filter '*.gob' -Recurse -ErrorAction SilentlyContinue | ForEach-Object {
    $Basename = $_.Name
    $Relative = $_.FullName.Replace($ProjectRoot, '').TrimStart('\', '/').Replace('\', '/')
    $ExistingGobFiles[$Basename.ToLower()] = $true
    $ExistingGobFiles[$Relative.ToLower()] = $true
}

$Findings = 0

function Add-Finding {
    param([string]$File, [int]$Line, [string]$Token)
    $script:Findings++
    $Rel = $File.Replace($ProjectRoot, '').TrimStart('\', '/')
    Write-Output ("  {0}:{1}: token `{2}` nao encontrado em arquivo .go" -f $Rel, $Line, $Token)
}

# Arquivos markdown para verificar
$DocFiles = @(
    Join-Path $ProjectRoot 'README.md'
)
$DocFiles += @(Get-ChildItem -Path (Join-Path $ProjectRoot 'docs') -Filter '*.md' -File | ForEach-Object { $_.FullName })

Write-Output "=== Verificacao de referencias a codigo ==="

foreach ($DocFile in $DocFiles) {
    if (-not (Test-Path $DocFile)) {
        continue
    }

    $Lines = @(Get-Content -Path $DocFile -Encoding utf8)

    for ($i = 0; $i -lt $Lines.Count; $i++) {
        $Line = $Lines[$i]

        # Pattern 1: arquivos .gob ou .go entre crases
        # Ex: `inverted_cache.gob`, `persist_codec.go`
        foreach ($Match in [regex]::Matches($Line, '`([a-zA-Z0-9_/.%-]+\.gob)`')) {
            $Token = $Match.Groups[1].Value
            $TokenLower = $Token.ToLower()
            if (-not $ExistingGobFiles.ContainsKey($TokenLower)) {
                Add-Finding -File $DocFile -Line ($i + 1) -Token $Token
            }
        }
        foreach ($Match in [regex]::Matches($Line, '`([a-zA-Z0-9_/.%-]+\.go)`')) {
            $Token = $Match.Groups[1].Value
            $TokenLower = $Token.ToLower()
            if (-not $ExistingGoFiles.ContainsKey($TokenLower)) {
                Add-Finding -File $DocFile -Line ($i + 1) -Token $Token
            }
        }

        # Pattern 2: snake_case tokens entre crases
        # Detecta apenas snake_case com pelo menos um underscore
        # Ex: `create_dirs`, `include_broken`
        foreach ($Match in [regex]::Matches($Line, '`([a-z][a-z0-9]*_[a-z0-9_]*)`')) {
            $Token = $Match.Groups[1].Value
            # Verifica se existe em algum arquivo .go
            $Found = $false
            foreach ($GoFile in $GoFiles) {
                $Content = $AllGoContent[$GoFile.ToLower()]
                if ($Content -match "\b$([regex]::Escape($Token))\b") {
                    $Found = $true
                    break
                }
            }
            if (-not $Found) {
                Add-Finding -File $DocFile -Line ($i + 1) -Token $Token
            }
        }

        # Pattern 3: CamelCase com parenteses entre crases
        # Ex: `MoveNote()`, `Canonicalize()`
        foreach ($Match in [regex]::Matches($Line, '`([A-Z][a-zA-Z0-9]*)\(\)`')) {
            $Token = $Match.Groups[1].Value
            $Found = $false
            foreach ($GoFile in $GoFiles) {
                $Content = $AllGoContent[$GoFile.ToLower()]
                if ($Content -match "\b$([regex]::Escape($Token))\s*\(") {
                    $Found = $true
                    break
                }
            }
            if (-not $Found) {
                Add-Finding -File $DocFile -Line ($i + 1) -Token "${Token}()"
            }
        }
    }
}

Write-Output ""
if ($Findings -eq 0) {
    Write-Output "[OK] Nenhum achado."
    exit 0
}

Write-Output "[!] $Findings achado(s). Cada um e uma referencia a um identificador"
Write-Output "    de codigo que nao foi localizado no repositorio."
exit 1
