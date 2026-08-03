#Requires -Version 5.1
<#
.SYNOPSIS
    Instala o gobsidian e o registra nos hosts de IA presentes na maquina.

.DESCRIPTION
    Baixa o binario da release do GitHub, CONFERE O SHA-256 contra o
    SHA256SUMS.txt publicado junto, instala em Program Files (ou em
    %LOCALAPPDATA%\Programs quando nao ha elevacao), acrescenta o diretorio ao
    PATH e registra o servidor MCP nos hosts que o usuario escolher.

    Uso tipico, sem argumentos:

        iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/install.ps1)

    `iex` executa o TEXTO do script, e um bloco param() nao recebe nada por essa
    via. Para passar opcoes, crie um scriptblock:

        & ([scriptblock]::Create((irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/install.ps1))) -Vault "C:\Meu Cofre" -Yes

    Ou use variaveis de ambiente, que funcionam nas duas formas:

        $env:GOBSIDIAN_VAULT = "C:\Meu Cofre"; iex (irm ...)

.PARAMETER Vault
    Raiz do cofre Obsidian. Sem isto, o script procura os cofres que o proprio
    Obsidian registrou e pergunta.

.PARAMETER Version
    Tag da release. Padrao: a mais recente.

.PARAMETER InstallDir
    Onde por o binario. Padrao: Program Files com elevacao, senao
    %LOCALAPPDATA%\Programs\gobsidian.

.PARAMETER Hosts
    Lista de chaves de host a configurar, sem perguntar. Ex.:
    -Hosts claude-desktop,claude-code

.PARAMETER Yes
    Nao pergunta nada. Exige -Vault.

.PARAMETER ReadOnly
    Registra o servidor com --read-only nos hosts.

.PARAMETER NoPath
    Nao mexe no PATH.

.EXAMPLE
    iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/install.ps1)
#>
[CmdletBinding()]
param(
    [string]$Vault = $env:GOBSIDIAN_VAULT,
    [string]$Version = $env:GOBSIDIAN_VERSION,
    [string]$InstallDir = $env:GOBSIDIAN_INSTALL_DIR,
    [string[]]$Hosts,
    [switch]$Yes,
    [switch]$ReadOnly,
    [switch]$NoPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"   # sem isto, Invoke-WebRequest fica ordens de grandeza mais lento

$Repo = "jonyduque/Gobsidian"
$ExeName = "gobsidian.exe"
# A v0.1.0 nomeou com hifen; a v1.0.0 com sublinhado. Aceita-se as duas.
$AssetNames = @("gobsidian_windows_amd64.exe", "gobsidian-windows-amd64.exe")
$ServerKey = "gobsidian"

# Saida em ASCII puro. Console PowerShell em CP-850 renderiza o resto como lixo,
# e um instalador e justamente onde alguem ja esta com um problema.
function Write-Step { param([string]$m) Write-Host "[...] $m" }
function Write-Ok { param([string]$m) Write-Host "[OK] $m" -ForegroundColor Green }
function Write-Info { param([string]$m) Write-Host "[i] $m" -ForegroundColor DarkGray }
function Write-Warn { param([string]$m) Write-Host "[!] $m" -ForegroundColor Yellow }
function Write-Err { param([string]$m) Write-Host "[!] $m" -ForegroundColor Red }

function Test-Interactive {
    # -Yes desliga toda pergunta. Sem console (pipeline de CI, tarefa agendada)
    # tambem: Read-Host ali le EOF e devolve string vazia, o que faria o script
    # "escolher" silenciosamente em vez de falhar.
    if ($Yes) { return $false }
    return [Environment]::UserInteractive -and -not [Console]::IsInputRedirected
}

function Test-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    (New-Object Security.Principal.WindowsPrincipal($id)).IsInRole(
        [Security.Principal.WindowsBuiltInRole]::Administrator)
}

if ($env:OS -ne "Windows_NT") {
    Write-Err "Este instalador e de Windows. Em Linux e macOS, baixe o binario da release e coloque no PATH."
    return
}

Write-Host ""
Write-Host "  gobsidian - servidor MCP para cofres Obsidian" -ForegroundColor Cyan
Write-Host ""

# ----------------------------------------------------------------------------
# 1. Qual versao
# ----------------------------------------------------------------------------

# TLS 1.2 explicito: o PowerShell 5.1 do Windows 10 ainda negocia TLS 1.0 por
# padrao, e a API do GitHub recusa.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Repositorio privado devolve 404 — nao 401 — para quem nao esta autenticado, o
# que faz "nao existe" e "voce nao pode ver" ficarem indistinguiveis. Com token,
# tudo passa pela API autenticada; sem token, pelas URLs publicas.
$Token = $env:GITHUB_TOKEN
if (-not $Token) { $Token = $env:GH_TOKEN }

$Headers = @{ "User-Agent" = "gobsidian-installer" }
if ($Token) {
    $Headers["Authorization"] = "Bearer $Token"
    Write-Info "Usando token do ambiente para acessar $Repo."
}

$Release = $null
try {
    $uri = if ($Version) {
        "https://api.github.com/repos/$Repo/releases/tags/$Version"
    }
    else {
        Write-Step "Consultando a release mais recente de $Repo"
        "https://api.github.com/repos/$Repo/releases/latest"
    }
    $Release = Invoke-RestMethod -Uri $uri -Headers $Headers
    $Version = $Release.tag_name
}
catch {
    Write-Err "Nao foi possivel consultar as releases de $Repo."
    Write-Info "Detalhe: $($_.Exception.Message)"
    if (-not $Token) {
        Write-Info "O GitHub devolve 404 tanto para release inexistente quanto para"
        Write-Info "repositorio privado. Se este for privado, defina um token e rode de novo:"
        Write-Info "  `$env:GITHUB_TOKEN = '<token com escopo repo>'"
    }
    return
}
Write-Ok "Versao: $Version"

# Baixa um asset da release resolvendo o ID pela LISTA de assets, sempre — nunca
# montando a URL de /releases/download a mao.
#
# Dois motivos, os dois medidos. Primeiro: num repositorio privado a URL publica
# nao aceita o cabecalho de autorizacao e ficaria em 404. Segundo, e pior: quando
# o nome nao existe, o GitHub REDIRECIONA para uma pagina HTML que responde 200,
# e o Invoke-WebRequest grava esse HTML no arquivo de destino com toda a
# aparencia de sucesso. A v0.1.0 deste projeto nomeia os binarios com HIFEN e a
# v1.0.0 com SUBLINHADO; pedir o nome errado escrevia uma pagina do GitHub por
# cima do que deveria ser um executavel.
#
# $Alternativas existe por causa dessa mesma divergencia de nomenclatura entre
# releases: aceita-se o que a release realmente publicou.
function Get-ReleaseAsset {
    param([string[]]$Alternativas, [string]$Destino)

    $asset = $null
    foreach ($n in $Alternativas) {
        $asset = $Release.assets | Where-Object { $_.name -eq $n } | Select-Object -First 1
        if ($asset) { break }
    }
    if (-not $asset) {
        $tem = ($Release.assets | ForEach-Object { $_.name }) -join ", "
        throw ("a release $Version nao traz nenhum de: $($Alternativas -join ', '). " +
               "Ela publica: $(if ($tem) { $tem } else { '(nenhum asset)' })")
    }

    $h = $Headers.Clone()
    $h["Accept"] = "application/octet-stream"
    Invoke-WebRequest -Uri $asset.url -Headers $h -OutFile $Destino -UseBasicParsing
    return $asset.name
}

# ----------------------------------------------------------------------------
# 2. Baixar e CONFERIR O HASH
# ----------------------------------------------------------------------------

$Tmp = Join-Path ([IO.Path]::GetTempPath()) ("gobsidian_install_" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $Tmp | Out-Null

try {
    $TmpExe = Join-Path $Tmp "gobsidian.exe"
    Write-Step "Baixando o binario de Windows"
    try {
        $AssetName = Get-ReleaseAsset -Alternativas $AssetNames -Destino $TmpExe
    }
    catch {
        # Sem isto, um asset ausente sai como excecao crua do PowerShell, com
        # stack trace e sublinhado de codigo — o oposto do que serve a quem so
        # queria instalar.
        Write-Err "Nao foi possivel baixar o binario."
        Write-Info $_.Exception.Message
        return
    }
    Write-Ok "Baixado: $AssetName"

    # A conferencia de hash nao e opcional. Um instalador que baixa um
    # executavel e o poe no PATH sem verificar o que baixou e um buraco de
    # cadeia de suprimentos com aparencia de conveniencia.
    Write-Step "Conferindo SHA-256 contra SHA256SUMS.txt"
    $TmpSums = Join-Path $Tmp "SHA256SUMS.txt"
    try {
        $null = Get-ReleaseAsset -Alternativas @("SHA256SUMS.txt") -Destino $TmpSums
        $Sums = Get-Content -Path $TmpSums -Raw
    }
    catch {
        Write-Err "SHA256SUMS.txt nao foi encontrado nesta release. Instalacao abortada."
        Write-Info "Um binario sem soma publicada nao pode ser verificado, e instalar assim"
        Write-Info "seria pior do que nao instalar."
        return
    }

    $Esperado = $null
    foreach ($linha in ($Sums -split "`n")) {
        if ($linha -match '^\s*([0-9a-fA-F]{64})\s+\*?(?:.*[\\/])?(.+?)\s*$') {
            if ($Matches[2] -eq $AssetName) { $Esperado = $Matches[1].ToLower(); break }
        }
    }
    if (-not $Esperado) {
        Write-Err "SHA256SUMS.txt nao traz uma linha para $AssetName. Instalacao abortada."
        return
    }

    $Obtido = (Get-FileHash -Path $TmpExe -Algorithm SHA256).Hash.ToLower()
    if ($Obtido -ne $Esperado) {
        Write-Err "SHA-256 NAO CONFERE. Instalacao abortada."
        Write-Info "  esperado: $Esperado"
        Write-Info "  obtido:   $Obtido"
        return
    }
    Write-Ok "SHA-256 confere ($($Esperado.Substring(0,16))...)"

    # ------------------------------------------------------------------------
    # 3. Onde instalar
    # ------------------------------------------------------------------------

    $Admin = Test-Admin
    if (-not $InstallDir) {
        if ($Admin) {
            $InstallDir = Join-Path $env:ProgramFiles "gobsidian"
        }
        else {
            # Sem elevacao, Program Files nao e gravavel. Instalar por usuario e
            # melhor do que pedir UAC para uma ferramenta de linha de comando.
            $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\gobsidian"
            Write-Info "Sem elevacao: instalando por usuario. Rode como Administrador para instalar em Program Files."
        }
    }

    $Destino = Join-Path $InstallDir $ExeName

    # Um gobsidian.exe em execucao mantem o arquivo travado, e o host MCP
    # costuma estar com um aberto. Encerra so os que apontam para ESTE destino.
    $EmUso = @(Get-Process -Name "gobsidian" -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -eq $Destino })
    if ($EmUso.Count -gt 0) {
        Write-Warn "$($EmUso.Count) processo(s) gobsidian em execucao a partir de $Destino; encerrando"
        $EmUso | Stop-Process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Milliseconds 500
    }

    Write-Step "Instalando em $InstallDir"
    if (-not (Test-Path $InstallDir)) { New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null }
    Copy-Item -Path $TmpExe -Destination $Destino -Force
    Write-Ok "Binario em $Destino"

    $Versao = & $Destino version 2>&1
    Write-Info "$Versao"

    # ------------------------------------------------------------------------
    # 4. PATH
    # ------------------------------------------------------------------------

    if (-not $NoPath) {
        $Alvo = if ($Admin -and $InstallDir.StartsWith($env:ProgramFiles)) { "Machine" } else { "User" }
        $Atual = [Environment]::GetEnvironmentVariable("Path", $Alvo)
        if (-not $Atual) { $Atual = "" }

        # Idempotente: rodar o instalador tres vezes nao pode deixar tres
        # entradas. Compara sem barra final e sem diferenciar caixa.
        $Norm = { param($p) $p.TrimEnd('\').ToLowerInvariant() }
        $Ja = $Atual -split ';' | Where-Object { $_ } | ForEach-Object { & $Norm $_ }
        if ($Ja -contains (& $Norm $InstallDir)) {
            Write-Info "PATH ($Alvo) ja contem $InstallDir"
        }
        else {
            $Novo = if ($Atual.TrimEnd(';')) { $Atual.TrimEnd(';') + ";" + $InstallDir } else { $InstallDir }
            [Environment]::SetEnvironmentVariable("Path", $Novo, $Alvo)
            Write-Ok "PATH ($Alvo) recebeu $InstallDir"
            Write-Info "Terminais ja abertos so enxergam a mudanca depois de reabrir."
        }
        # Vale para o resto DESTA sessao tambem.
        if (($env:Path -split ';' | ForEach-Object { & $Norm $_ }) -notcontains (& $Norm $InstallDir)) {
            $env:Path = "$env:Path;$InstallDir"
        }
    }
}
finally {
    Remove-Item -Path $Tmp -Recurse -Force -ErrorAction SilentlyContinue
}

# ----------------------------------------------------------------------------
# 5. Qual cofre
# ----------------------------------------------------------------------------

function Get-ObsidianVaults {
    # O proprio Obsidian mantem o registro dos cofres abertos em
    # %APPDATA%\obsidian\obsidian.json. Ler dali evita pedir ao usuario um
    # caminho que ele teria de ir buscar.
    $reg = Join-Path $env:APPDATA "obsidian\obsidian.json"
    if (-not (Test-Path $reg)) { return @() }
    try {
        $j = Get-Content -Path $reg -Raw -Encoding UTF8 | ConvertFrom-Json
        if (-not $j.PSObject.Properties.Name.Contains("vaults")) { return @() }
        return @($j.vaults.PSObject.Properties |
            ForEach-Object { $_.Value.path } |
            Where-Object { $_ -and (Test-Path $_) })
    }
    catch { return @() }
}

if (-not $Vault) {
    $Encontrados = @(Get-ObsidianVaults)
    if ($Encontrados.Count -eq 1 -and -not (Test-Interactive)) {
        $Vault = $Encontrados[0]
    }
    elseif ($Encontrados.Count -gt 0 -and (Test-Interactive)) {
        Write-Host ""
        Write-Host "  Cofres que o Obsidian conhece:" -ForegroundColor Cyan
        for ($i = 0; $i -lt $Encontrados.Count; $i++) {
            Write-Host ("   [{0}] {1}" -f ($i + 1), $Encontrados[$i])
        }
        Write-Host "   [o] outro caminho"
        Write-Host ""
        $r = Read-Host "  Qual cofre"
        if ($r -match '^\d+$' -and [int]$r -ge 1 -and [int]$r -le $Encontrados.Count) {
            $Vault = $Encontrados[[int]$r - 1]
        }
    }
}

if (-not $Vault -and (Test-Interactive)) {
    $Vault = (Read-Host "  Caminho do cofre Obsidian").Trim('"', ' ')
}

if (-not $Vault) {
    Write-Err "Nenhum cofre indicado. Passe -Vault ou defina GOBSIDIAN_VAULT."
    return
}

if (-not (Test-Path -Path $Vault -PathType Container)) {
    Write-Err "Cofre nao encontrado: $Vault"
    return
}
$Vault = (Resolve-Path -Path $Vault).Path

if (-not (Test-Path (Join-Path $Vault ".obsidian"))) {
    # Aviso, nao erro: um cofre recem-criado, ou aberto por outra ferramenta,
    # pode nao ter .obsidian ainda, e o gobsidian indexa Markdown de qualquer
    # diretorio.
    Write-Warn "Nao ha .obsidian em $Vault - pode nao ser um cofre do Obsidian."
}
Write-Ok "Cofre: $Vault"

# ----------------------------------------------------------------------------
# 6. Quais hosts de IA existem
# ----------------------------------------------------------------------------

$ArgsServe = @("serve", "--vault", $Vault)
if ($ReadOnly) { $ArgsServe += "--read-only" }

function Merge-McpJson {
    <#
        Acrescenta o servidor a um arquivo {"mcpServers": {...}} PRESERVANDO
        todo o resto. Sobrescrever o arquivo apagaria as preferencias do host e
        os outros servidores que o usuario ja tem - o claude_desktop_config.json
        desta maquina, por exemplo, carrega 'preferences' e outro servidor MCP
        ao lado.
    #>
    param([string]$Path, [string]$Exe, [string[]]$ServerArgs)

    $dir = Split-Path -Parent $Path
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }

    $doc = $null
    if (Test-Path $Path) {
        # Backup antes de tocar. Config de host e coisa que o usuario editou a
        # mao e nao tem copia.
        Copy-Item -Path $Path -Destination "$Path.gobsidian-backup" -Force
        $bruto = Get-Content -Path $Path -Raw -Encoding UTF8
        if ($bruto.Trim()) { $doc = $bruto | ConvertFrom-Json }
    }
    if (-not $doc) { $doc = [pscustomobject]@{} }

    if (-not $doc.PSObject.Properties.Name.Contains("mcpServers")) {
        $doc | Add-Member -MemberType NoteProperty -Name "mcpServers" -Value ([pscustomobject]@{})
    }

    $entry = [pscustomobject]@{ command = $Exe; args = $ServerArgs }
    if ($doc.mcpServers.PSObject.Properties.Name -contains $ServerKey) {
        $doc.mcpServers.$ServerKey = $entry
    }
    else {
        $doc.mcpServers | Add-Member -MemberType NoteProperty -Name $ServerKey -Value $entry
    }

    # UTF8 sem BOM: o PowerShell 5.1 escreve BOM com -Encoding UTF8, e alguns
    # leitores de JSON engasgam com ele.
    $json = $doc | ConvertTo-Json -Depth 20
    [IO.File]::WriteAllText($Path, $json, (New-Object Text.UTF8Encoding($false)))
}

function Test-Exe { param([string]$n) [bool](Get-Command $n -ErrorAction SilentlyContinue) }

$AppData = $env:APPDATA
$Home_ = $env:USERPROFILE
$LocalProgs = Join-Path $env:LOCALAPPDATA "Programs"

# A deteccao olha para o EXECUTAVEL ou para a instalacao, nunca so para o
# diretorio de configuracao. Nesta maquina existem ~/.cursor e ~/.codex sem que
# nenhum dos dois produtos esteja instalado - foram criados por outra
# ferramenta. Detectar por diretorio ofereceria hosts que nao existem.
$Candidatos = @(
    [pscustomobject]@{
        Key      = "claude-desktop"
        Nome     = "Claude Desktop"
        Detecta  = { (Test-Path (Join-Path $env:APPDATA "Claude")) -or
                     (Test-Path (Join-Path $env:LOCALAPPDATA "AnthropicClaude")) }
        Configura = {
            param($exe, $a)
            Merge-McpJson -Path (Join-Path $env:APPDATA "Claude\claude_desktop_config.json") -Exe $exe -ServerArgs $a
            "Reinicie o Claude Desktop para carregar o servidor."
        }
    }
    [pscustomobject]@{
        Key      = "claude-code"
        Nome     = "Claude Code (CLI)"
        Detecta  = { Test-Exe "claude" }
        Configura = {
            param($exe, $a)
            # O proprio CLI escreve a config dele. Melhor do que eu adivinhar o
            # formato e o arquivo, que mudam entre versoes.
            #
            # A lista inteira vai como ARRAY splatado, e nao escrita na linha de
            # comando. O PowerShell trata "--" como fim-de-parametros e o REMOVE
            # quando ele aparece literalmente, entao `claude mcp add nome -- exe
            # serve --vault X` chegava ao claude sem o separador e falhava com
            # "unknown option '--vault'". Dentro de um array splatado o "--"
            # atravessa intacto.
            $cli = @("mcp", "add", $ServerKey, "--scope", "user", "--", $exe) + $a
            $saida = & claude @cli 2>&1
            if ($LASTEXITCODE -ne 0) { throw "claude mcp add saiu $LASTEXITCODE`: $($saida -join ' ')" }
            "Registrado no escopo de usuario."
        }
    }
    [pscustomobject]@{
        Key      = "gemini-cli"
        Nome     = "Gemini CLI"
        Detecta  = { Test-Exe "gemini" }
        Configura = {
            param($exe, $a)
            # O gemini nao usa "--", entao nao sofre do problema do claude.
            $cli = @("mcp", "add", $ServerKey, $exe) + $a + @("--scope", "user")
            $saida = & gemini @cli 2>&1
            if ($LASTEXITCODE -ne 0) { throw "gemini mcp add saiu $LASTEXITCODE`: $($saida -join ' ')" }
            "Registrado no escopo de usuario."
        }
    }
    [pscustomobject]@{
        Key      = "antigravity"
        Nome     = "Antigravity"
        Detecta  = { Test-Path (Join-Path $env:LOCALAPPDATA "Programs\Antigravity") }
        Configura = {
            param($exe, $a)
            Merge-McpJson -Path (Join-Path $env:USERPROFILE ".gemini\antigravity\mcp_config.json") -Exe $exe -ServerArgs $a
            "Reinicie o Antigravity."
        }
    }
    [pscustomobject]@{
        Key      = "antigravity-ide"
        Nome     = "Antigravity IDE"
        Detecta  = { Test-Path (Join-Path $env:LOCALAPPDATA "Programs\Antigravity IDE") }
        Configura = {
            param($exe, $a)
            Merge-McpJson -Path (Join-Path $env:USERPROFILE ".gemini\antigravity-ide\mcp_config.json") -Exe $exe -ServerArgs $a
            "Reinicie o Antigravity IDE."
        }
    }
    [pscustomobject]@{
        Key      = "codex"
        Nome     = "Codex CLI"
        Detecta  = { Test-Exe "codex" }
        Configura = {
            param($exe, $a)
            # Mesmo cuidado do claude com o "--": array splatado, nunca
            # literal. Nao foi possivel verificar este caminho — o codex nao
            # esta instalado na maquina onde o instalador foi desenvolvido —,
            # entao a falha aqui e reportada por host e nao derruba os demais.
            $cli = @("mcp", "add", $ServerKey, "--", $exe) + $a
            $saida = & codex @cli 2>&1
            if ($LASTEXITCODE -ne 0) { throw "codex mcp add saiu $LASTEXITCODE`: $($saida -join ' ')" }
            "Registrado em ~/.codex/config.toml."
        }
    }
    [pscustomobject]@{
        Key      = "vscode"
        Nome     = "VS Code"
        Detecta  = { Test-Exe "code" }
        Configura = {
            param($exe, $a)
            $def = @{ name = $ServerKey; command = $exe; args = $a } | ConvertTo-Json -Compress
            $saida = & code --add-mcp $def 2>&1
            if ($LASTEXITCODE -ne 0) { throw "code --add-mcp saiu $LASTEXITCODE`: $($saida -join ' ')" }
            "Registrado na configuracao de usuario do VS Code."
        }
    }
    [pscustomobject]@{
        Key      = "cursor"
        Nome     = "Cursor"
        Detecta  = { (Test-Exe "cursor") -or (Test-Path (Join-Path $env:LOCALAPPDATA "Programs\cursor")) }
        Configura = {
            param($exe, $a)
            Merge-McpJson -Path (Join-Path $env:USERPROFILE ".cursor\mcp.json") -Exe $exe -ServerArgs $a
            "Reinicie o Cursor."
        }
    }
    [pscustomobject]@{
        Key      = "windsurf"
        Nome     = "Windsurf"
        Detecta  = { Test-Path (Join-Path $env:USERPROFILE ".codeium\windsurf") }
        Configura = {
            param($exe, $a)
            Merge-McpJson -Path (Join-Path $env:USERPROFILE ".codeium\windsurf\mcp_config.json") -Exe $exe -ServerArgs $a
            "Reinicie o Windsurf."
        }
    }
)

Write-Host ""
Write-Step "Procurando hosts de IA"
$Detectados = @($Candidatos | Where-Object { & $_.Detecta })

if ($Detectados.Count -eq 0) {
    Write-Warn "Nenhum host conhecido foi detectado."
    Write-Info "O binario esta instalado e no PATH. Configure seu host manualmente com:"
    Write-Info "  comando: $Destino"
    Write-Info "  args:    $($ArgsServe -join ' ')"
    return
}

foreach ($h in $Detectados) { Write-Ok "encontrado: $($h.Nome)" }

# ----------------------------------------------------------------------------
# 7. Escolha
# ----------------------------------------------------------------------------

$Escolhidos = @()
if ($Hosts) {
    $Escolhidos = @($Detectados | Where-Object { $Hosts -contains $_.Key })
    $NaoAchados = @($Hosts | Where-Object { $Detectados.Key -notcontains $_ })
    foreach ($n in $NaoAchados) { Write-Warn "host pedido mas nao detectado: $n" }
}
elseif (-not (Test-Interactive)) {
    Write-Warn "Modo nao interativo sem -Hosts: nenhum host sera configurado."
    Write-Info "Passe -Hosts com as chaves: $($Detectados.Key -join ', ')"
}
else {
    Write-Host ""
    Write-Host "  Em quais registrar o gobsidian?" -ForegroundColor Cyan
    for ($i = 0; $i -lt $Detectados.Count; $i++) {
        Write-Host ("   [{0}] {1}" -f ($i + 1), $Detectados[$i].Nome)
    }
    Write-Host "   [t] todos"
    Write-Host "   [n] nenhum"
    Write-Host ""
    Write-Host "  Numeros separados por virgula, ou t/n." -ForegroundColor DarkGray
    $r = (Read-Host "  Escolha").Trim()

    if ($r -eq "t" -or $r -eq "") { $Escolhidos = $Detectados }
    elseif ($r -eq "n") { $Escolhidos = @() }
    else {
        foreach ($p in ($r -split '[,\s]+' | Where-Object { $_ })) {
            if ($p -match '^\d+$' -and [int]$p -ge 1 -and [int]$p -le $Detectados.Count) {
                $Escolhidos += $Detectados[[int]$p - 1]
            }
            else { Write-Warn "ignorando entrada invalida: $p" }
        }
    }
}

# ----------------------------------------------------------------------------
# 8. Configurar
# ----------------------------------------------------------------------------

$Ok = @(); $Falhou = @()
foreach ($h in $Escolhidos) {
    Write-Step "Configurando $($h.Nome)"
    try {
        $msg = & $h.Configura $Destino $ArgsServe
        Write-Ok "$($h.Nome) - $msg"
        $Ok += $h.Nome
    }
    catch {
        # Um host que falha nao pode derrubar os outros: quem tem quatro hosts e
        # ve o instalador morrer no segundo fica sem saber o que foi feito.
        Write-Err "$($h.Nome): $($_.Exception.Message)"
        $Falhou += $h.Nome
    }
}

# ----------------------------------------------------------------------------
# 9. Resumo
# ----------------------------------------------------------------------------

Write-Host ""
Write-Host "  Resumo" -ForegroundColor Cyan
Write-Host "   binario   $Destino"
Write-Host "   versao    $Versao"
Write-Host "   cofre     $Vault"
if ($ReadOnly) { Write-Host "   modo      somente leitura (--read-only)" }
if ($Ok.Count -gt 0) { Write-Host "   ok        $($Ok -join ', ')" -ForegroundColor Green }
if ($Falhou.Count -gt 0) { Write-Host "   falhou    $($Falhou -join ', ')" -ForegroundColor Red }
Write-Host ""

if ($Ok.Count -gt 0) {
    Write-Info "Reinicie os hosts configurados para que carreguem o servidor."
}
Write-Info "Se algo nao funcionar, o primeiro comando e:"
Write-Host "   gobsidian doctor --vault `"$Vault`""
Write-Host ""
