<#
    Bootstrap do gobsidian para Windows.

    Ele faz UMA coisa: baixa o executavel da ultima versao e o roda. Detectar
    cofre, mexer no PATH e configurar hosts de IA e trabalho do proprio binario
    desde 2026-09-08 (decisao D-01 do dono) -- la isso e testavel, e aqui nao
    era: as 729 linhas que este arquivo substitui nao tinham um unico teste.

    A conferencia de SHA-256 NAO acontece aqui, e isso e deliberado (decisao
    D-03): quem confere e o `gobsidian update`, o binario VELHO sobre o NOVO.

        iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install.ps1)

    Argumentos passam adiante:
        & ([scriptblock]::Create((irm ...))) --vault "C:\Meu Cofre" --yes
#>
[CmdletBinding()]
param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Args)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
# Sem isto o Invoke-WebRequest fica ordens de grandeza mais lento.
$ProgressPreference = 'SilentlyContinue'
# O PowerShell 5.1 do Windows 10 ainda negocia TLS 1.0 por padrao, e o GitHub recusa.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Repo = 'jonyduque/Gobsidian'
$Ativo = 'gobsidian-windows-amd64.exe'
$Url = "https://github.com/$Repo/releases/latest/download/$Ativo"

$Tmp = Join-Path ([IO.Path]::GetTempPath()) ("gobsidian_bootstrap_" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $Tmp | Out-Null
$Exe = Join-Path $Tmp 'gobsidian.exe'

Write-Host "[...] baixando $Ativo"
Invoke-WebRequest -Uri $Url -OutFile $Exe

Write-Host "[OK] baixado; entregando ao instalador do proprio binario"
& $Exe install @Args
exit $LASTEXITCODE
