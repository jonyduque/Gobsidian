#Requires -Version 7.0
<#
.SYNOPSIS
    Acha parametro de tool declarado no schema que o codigo nunca le.

.DESCRIPTION
    O modelo do outro lado le o schema para decidir o que mandar. Um campo que
    aparece la e que o handler ignora e pior que um campo ausente: o pedido nao
    faz nada e nao ha como o cliente descobrir.

    A checagem opera em dois niveis:
      1. Por struct em internal/mcpsrv: o campo in.<Campo> tem de aparecer no
         corpo do handler da respectiva tool, verificado estritamente por
         in.<Campo> (-cmatch). Se a struct nao tiver handler resolvido, e um
         achado HANDLER-NAO-RESOLVIDO (exit 1).
      2. Seguir ate o dominio (internal/): quando o campo e repassado, ele tem de
         ser lido por acesso .$Campo em alguma funcao de dominio alem da
         propria declaracao de struct.

    LIMITE CONHECIDO DO NIVEL 2: ele casa `.Campo` em QUALQUER lugar de
    internal/, sem escopo da struct de destino. Dois campos de nome igual em
    structs diferentes se cobrem um ao outro -- `TagRequest.Sort` passa se
    `ListRequest.Sort` for lido em algum lugar, e vice-versa. O nivel 2 so
    dispara para um nome que nao aparece em NENHUM acesso por ponto em
    internal/, que foi o caso de Hierarchical. Fechar isso exige seguir o campo
    ate o tipo de destino e procurar o acesso nos metodos que o recebem; ate
    entao, este nivel e uma rede de seguranca, nao uma garantia.

    O nivel 1 NAO tem esse limite: ele exige `$pVar.$Campo` dentro do corpo do
    handler daquela tool, com -cmatch. A versao anterior deste script tinha um
    segundo disjunto que casava o nome nu no corpo do handler, e ele anulava o
    escopo: apagado o unico leitor de noteListInput.Sort, o gate continuava
    reportando 2 campos mortos em vez de 3, porque o corpo ainda continha
    `sort := "path"`. Nao reintroduza disjunto de nome nu em nivel nenhum.

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

    # Fontes de producao em mcpsrv. Os _test.go ficam de fora de proposito: um campo lido
    # so pelo teste continua sendo um campo que a tool ignora.
    $Fontes = Get-ChildItem -Path $Alvo -Filter "*.go" -File |
        Where-Object { $_.Name -notlike "*_test.go" }

    if ($Fontes.Count -eq 0) {
        Write-Warning "[!] nenhum .go de producao em $Path"
        exit 2
    }

    # Carrega arquivos de dominio em internal/ (service, index, search, vault, writer, parser, config)
    $DomainSubdirs = @("internal/service", "internal/index", "internal/search", "internal/vault", "internal/writer", "internal/parser", "internal/config")
    $DomainFiles = @()
    foreach ($sub in $DomainSubdirs) {
        $p = Join-Path $ProjectRoot $sub
        if (Test-Path $p) {
            $DomainFiles += Get-ChildItem -Path $p -Filter "*.go" -File | Where-Object { $_.Name -notlike "*_test.go" }
        }
    }
    $DomainLines = @()
    foreach ($df in $DomainFiles) {
        $DomainLines += Get-Content -Path $df.FullName
    }

    # Mapeia handlers de cada struct Input em mcpsrv
    $StructHandlers = @{}
    foreach ($f in $Fontes) {
        $content = [System.IO.File]::ReadAllText($f.FullName)
        $pattern = "(?s)func\s*\([^)]*?,\s*(\w+)\s+([A-Za-z0-9_]+Input)\s*\)"
        $matches = [regex]::Matches($content, $pattern)
        foreach ($m in $matches) {
            $paramVar = $m.Groups[1].Value
            $structName = $m.Groups[2].Value
            $startIdx = $m.Index + $m.Length
            $braceCount = 0
            $bodyStart = -1
            for ($i = $startIdx; $i -lt $content.Length; $i++) {
                if ($content[$i] -eq "{") {
                    if ($braceCount -eq 0) { $bodyStart = $i }
                    $braceCount++
                } elseif ($content[$i] -eq "}") {
                    $braceCount--
                    if ($braceCount -eq 0 -and $bodyStart -ne -1) {
                        $bodyEnd = $i
                        $body = $content.Substring($bodyStart, $bodyEnd - $bodyStart + 1)
                        $StructHandlers[$structName] = @{
                            File     = $f.Name
                            ParamVar = $paramVar
                            Body     = $body
                        }
                        break
                    }
                }
            }
        }
    }

    $Achados = @()
    $Campos = 0
    $Structs = 0
    $Coverage = @()

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

            if ($L -notmatch '^\s*([A-Z]\w*)\s+.+?`.*json:"([^",]+)') { continue }

            $NomeGo = $Matches[1]
            $NomeJson = $Matches[2]
            $Campos++

            $h = $StructHandlers[$StructAtual]
            if (-not $h) {
                $Achados += [pscustomobject]@{
                    Arquivo = $f.Name
                    Linha   = $i + 1
                    Struct  = $StructAtual
                    Campo   = $NomeGo
                    Json    = $NomeJson
                    Motivo  = "HANDLER-NAO-RESOLVIDO (nenhum handler encontrado para $StructAtual)"
                }
                $Coverage += "    $StructAtual.$NomeGo -> HANDLER-NAO-RESOLVIDO"
                continue
            }

            $handlerBody = $h.Body
            $pVar = $h.ParamVar

            # Nivel 1: Leitura estrita in.Campo no handler (case-sensitive com -cmatch)
            $level1Read = ($handlerBody -cmatch "\b$([regex]::Escape($pVar))\.$([regex]::Escape($NomeGo))\b")

            if (-not $level1Read) {
                $Achados += [pscustomobject]@{
                    Arquivo = $f.Name
                    Linha   = $i + 1
                    Struct  = $StructAtual
                    Campo   = $NomeGo
                    Json    = $NomeJson
                    Motivo  = "nunca lido no handler da tool (mcpsrv)"
                }
                $Coverage += "    $StructAtual.$NomeGo -> NENHUMA LEITURA no handler (mcpsrv)"
                continue
            }

            # Nivel 2: Leitura por acesso .$NomeGo nas funcoes de dominio em internal/ ou consumo direto no handler
            $domainMatchCount = 0
            foreach ($dline in $DomainLines) {
                if ($dline -match '^\s*//') { continue }
                if ($dline -cmatch "\.$([regex]::Escape($NomeGo))\b") {
                    $domainMatchCount++
                }
            }

            $mcpsrvLogic = ($handlerBody -cmatch "if\s+.*?\b$([regex]::Escape($pVar))\.$([regex]::Escape($NomeGo))\b") -or 
                           ($handlerBody -cmatch "for\s+.*?\b$([regex]::Escape($pVar))\.$([regex]::Escape($NomeGo))\b") -or
                           ($handlerBody -cmatch "len\(\s*$([regex]::Escape($pVar))\.$([regex]::Escape($NomeGo))\s*\)")

            if ($domainMatchCount -eq 0 -and -not $mcpsrvLogic) {
                $Achados += [pscustomobject]@{
                    Arquivo = $f.Name
                    Linha   = $i + 1
                    Struct  = $StructAtual
                    Campo   = $NomeGo
                    Json    = $NomeJson
                    Motivo  = "repassado ao dominio mas nunca lido em internal/"
                }
                $Coverage += "    $StructAtual.$NomeGo -> $($f.Name) (mcpsrv), NENHUMA LEITURA em internal/"
            } else {
                $loc = "$($f.Name) (mcpsrv)"
                if ($domainMatchCount -gt 0) { $loc += ", internal/ (dominio)" }
                $Coverage += "    $StructAtual.$NomeGo -> $loc"
            }
        }
    }

    Write-Output "[i] $Structs structs de entrada, $Campos parametros declarados."
    Write-Output ""
    Write-Output "[i] Tabela de cobertura de parametros:"
    foreach ($c in $Coverage) {
        Write-Output $c
    }
    Write-Output ""

    if ($Achados.Count -eq 0) {
        Write-Output "[OK] todo parametro declarado e lido em algum lugar."
        exit 0
    }

    Write-Output "[!] $($Achados.Count) parametro(s) declarado(s) e nunca lido(s):"
    foreach ($a in $Achados) {
        Write-Output "    $($a.Arquivo):$($a.Linha)  $($a.Struct).$($a.Campo)  (json: `"$($a.Json)`") -> $($a.Motivo)"
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
