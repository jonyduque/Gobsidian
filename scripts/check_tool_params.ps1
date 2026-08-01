#Requires -Version 7.0
<#
.SYNOPSIS
    Acha parametro de tool declarado no schema que o codigo nunca le.

.DESCRIPTION
    O modelo do outro lado le o schema para decidir o que mandar. Um campo que
    aparece la e que o handler ignora e pior que um campo ausente: o pedido nao
    faz nada e nao ha como o cliente descobrir.

    Este defeito apareceu CINCO vezes neste projeto, sempre com a mesma
    assinatura mecanica -- um campo declarado numa struct `*Input` de
    internal/mcpsrv cujo nome nao aparece em mais lugar nenhum do pacote:

      note_list.fields          declarado e descartado
      note_append.ensure_blank_line   declarado, default documentado, nunca lido
      vault_stats.include_health      declarado, default documentado, nunca lido

    Nenhum gate pegava. O compilador de Go nao reclama de campo de struct sem
    uso, e o golangci-lint tambem nao -- o campo E usado, pelo decodificador de
    JSON, o que basta para todo analisador de uso.

    A checagem: para cada campo de cada struct `*Input`, o nome do campo em Go
    tem de aparecer pelo menos uma vez ALEM da propria declaracao, dentro de
    internal/mcpsrv. Se aparece so uma vez, ninguem leu.

    NAO prova que o campo e honrado corretamente -- so que alguem o leu. O
    contrario disso, o campo lido e mal aplicado, e trabalho de teste, e para
    isso existe internal/mcpsrv/defaults_test.go, que chama cada tool SEM o
    campo e afirma o comportamento documentado.

.PARAMETER Path
    Diretorio a inspecionar. Padrao: internal/mcpsrv.

.EXAMPLE
    .\scripts\check_tool_params.ps1
#>
[CmdletBinding()]
param(
    [string]$Path = "internal/mcpsrv"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $ProjectRoot
try {
    $Alvo = Join-Path $ProjectRoot $Path
    if (-not (Test-Path $Alvo)) {
        Write-Warning "[!] diretorio nao encontrado: $Alvo"
        exit 2
    }

    # Fontes de producao. Os _test.go ficam de fora de proposito: um campo lido
    # so pelo teste continua sendo um campo que a tool ignora.
    $Fontes = Get-ChildItem -Path $Alvo -Filter "*.go" -File |
        Where-Object { $_.Name -notlike "*_test.go" }

    if ($Fontes.Count -eq 0) {
        Write-Warning "[!] nenhum .go de producao em $Path"
        exit 2
    }

    $Corpo = @{}
    foreach ($f in $Fontes) { $Corpo[$f.Name] = Get-Content -Path $f.FullName -Raw }
    $Tudo = ($Corpo.Values) -join "`n"

    $Achados = @()
    $Campos = 0
    $Structs = 0

    foreach ($f in $Fontes) {
        $Linhas = Get-Content -Path $f.FullName
        $DentroDeInput = $false
        $StructAtual = ""

        for ($i = 0; $i -lt $Linhas.Count; $i++) {
            $L = $Linhas[$i]

            if ($L -match '^\s*type\s+(\w*[Ii]nput)\s+struct\s*\{') {
                $DentroDeInput = $true
                $StructAtual = $Matches[1]
                $Structs++
                continue
            }
            if ($DentroDeInput -and $L -match '^\s*\}') {
                $DentroDeInput = $false
                $StructAtual = ""
                continue
            }
            if (-not $DentroDeInput) { continue }

            # Campo exportado com tag json. Campo embutido ou sem tag nao e
            # parametro de schema e nao entra na conta.
            if ($L -notmatch '^\s*([A-Z]\w*)\s+[\w\*\[\]\.]+\s+`.*json:"([^",]+)') { continue }

            $NomeGo = $Matches[1]
            $NomeJson = $Matches[2]
            $Campos++

            # Conta ocorrencias do identificador no pacote inteiro. A propria
            # declaracao e uma; qualquer leitura seria a segunda.
            $Ocorrencias = ([regex]::Matches($Tudo, "\b$([regex]::Escape($NomeGo))\b")).Count

            if ($Ocorrencias -le 1) {
                $Achados += [pscustomobject]@{
                    Arquivo = $f.Name
                    Linha   = $i + 1
                    Struct  = $StructAtual
                    Campo   = $NomeGo
                    Json    = $NomeJson
                }
            }
        }
    }

    Write-Output "[i] $Structs structs de entrada, $Campos parametros declarados."

    if ($Achados.Count -eq 0) {
        Write-Output "[OK] todo parametro declarado e lido em algum lugar."
        exit 0
    }

    Write-Warning "[!] $($Achados.Count) parametro(s) declarado(s) e nunca lido(s):"
    foreach ($a in $Achados) {
        Write-Output "    $($a.Arquivo):$($a.Linha)  $($a.Struct).$($a.Campo)  (json: `"$($a.Json)`")"
    }
    Write-Output ""
    Write-Output "[i] Ou implemente, ou tire do schema E de docs/TOOLS.md. Schema que"
    Write-Output "    promete e codigo que ignora e pior que parametro ausente: o modelo"
    Write-Output "    do outro lado le o schema justamente para decidir o que mandar."
    exit 1
}
finally {
    Pop-Location
}
