# Instalador do gobsidian para Windows.
#
#   iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.ps1)
#
# Para passar opcoes ao instalador, use um scriptblock -- `iex` executa o TEXTO
# do script e um param() nao recebe nada por essa via:
#
#   & ([scriptblock]::Create((irm .../installer/install.ps1))) --vault "C:\Meu Cofre" --yes

$ErrorActionPreference = "Stop"

if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    # ASCII puro: um console PowerShell em CP-850 renderiza emoji como lixo,
    # e esta e a mensagem de quem ja esta com problema.
    Write-Host "[!] Instale Node.js 18+: https://nodejs.org" -ForegroundColor Red
    return
}

# TLS 1.2 explicito: o PowerShell 5.1 do Windows 10 ainda negocia TLS 1.0 por
# padrao, e o raw.githubusercontent.com recusa.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Url = "https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.js"
$f = Join-Path $env:TEMP ("gobsidian-install-" + [guid]::NewGuid().ToString("N") + ".mjs")

try {
    Invoke-WebRequest -Uri $Url -OutFile $f -UseBasicParsing
    # $args repassa o que veio da linha de comando; vazio no caso do iex simples.
    & node $f @args
}
finally {
    Remove-Item $f -Force -ErrorAction SilentlyContinue
}
