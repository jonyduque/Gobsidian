param(
    [int]$Cycles = 100,
    [string]$VaultPath,
    [int]$PidTimeoutMs = 5000,
    [int]$SettleMs = 2000
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$BinaryPath = Join-Path $ProjectRoot "bin\gobsidian.exe"
$HostScript = Join-Path $PSScriptRoot "orphan_host.ps1"

if (-not (Test-Path $BinaryPath)) {
    Write-Warning "[!] Binario nao encontrado: $BinaryPath"
    exit 1
}
if (-not (Test-Path $HostScript)) {
    Write-Warning "[!] Script do host nao encontrado: $HostScript"
    exit 1
}

if (-not $VaultPath) {
    $VaultPath = Join-Path $ProjectRoot "testdata\vault_small"
}

$WorkDir = Join-Path $env:TEMP ("gobsidian_orphan_" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $WorkDir | Out-Null

Write-Output "[...] $Cycles ciclos de encerramento abrupto (host com pipe real em stdin)"
Write-Output "[i] logs e PIDs em $WorkDir"

$Survivors = 0
$LaunchFailures = 0
$KillFailures = 0

for ($i = 1; $i -le $Cycles; $i++) {
    $PidFile = Join-Path $WorkDir "cycle_$i.pid"
    $LogFile = Join-Path $WorkDir "cycle_$i.log"

    # O "host" e um processo pwsh separado (orphan_host.ps1) que constroi o
    # filho via ProcessStartInfo com RedirectStandardInput = $true e SEGURA a
    # ponta de escrita do pipe — exatamente o que Claude Desktop faz. Matar
    # ESTE processo (o host), nao o filho, e o que exercita os mecanismos de
    # encerramento do filho.
    $HostProc = Start-Process -FilePath "pwsh.exe" `
        -ArgumentList "-NoProfile", "-NonInteractive", "-File", $HostScript, `
                      "-BinaryPath", $BinaryPath, "-VaultPath", $VaultPath, "-PidFile", $PidFile `
        -PassThru -WindowStyle Hidden `
        -RedirectStandardError $LogFile

    if (-not $HostProc) {
        $LaunchFailures++
        Write-Warning "[!] Ciclo ${i}: Start-Process nao devolveu um processo host"
        continue
    }

    # O PID do filho so existe depois que o host o cria; ler o arquivo cedo
    # demais e uma corrida, nao uma falha. Poll com timeout — e um ciclo cujo
    # filho nunca apareceu e um ERRO, nunca uma passagem silenciosa. Sem
    # isso, um harness quebrado que nunca lanca nada mediria zero orfaos por
    # nunca ter medido coisa alguma, e o gate passaria sem provar nada (foi
    # exatamente o defeito que o predecessor encontrou na forma anterior
    # deste script).
    $Deadline = [DateTime]::UtcNow.AddMilliseconds($PidTimeoutMs)
    $ServerPid = $null
    while ([DateTime]::UtcNow -lt $Deadline) {
        if (Test-Path $PidFile) {
            $Content = Get-Content -Path $PidFile -Raw -ErrorAction SilentlyContinue
            if ($Content -and $Content.Trim() -match '^\d+$') {
                $ServerPid = [int]$Content.Trim()
                break
            }
        }
        Start-Sleep -Milliseconds 50
    }

    if (-not $ServerPid) {
        $LaunchFailures++
        Write-Warning "[!] Ciclo ${i}: PID do servidor nao apareceu em ${PidTimeoutMs}ms - ciclo nao mediu nada"
        Stop-Process -Id $HostProc.Id -Force -ErrorAction SilentlyContinue
        continue
    }

    $ServerAlive = Get-Process -Id $ServerPid -ErrorAction SilentlyContinue
    if (-not $ServerAlive) {
        $LaunchFailures++
        Write-Warning "[!] Ciclo ${i}: PID $ServerPid registrado mas o processo ja nao existe"
        Stop-Process -Id $HostProc.Id -Force -ErrorAction SilentlyContinue
        continue
    }

    # Mata o host, NAO o filho, e sem /T. E a morte do host que deve fechar
    # o pipe e disparar o EOF (ou, na falta desse mecanismo, a vigilia do
    # processo pai). Matar a arvore inteira provaria so que taskkill sabe
    # matar processos, nao que o lifecycle do filho funciona.
    taskkill /F /PID $HostProc.Id 2>$null | Out-Null
    if ($LASTEXITCODE -ne 0) {
        $KillFailures++
        Write-Warning "[!] Ciclo ${i}: taskkill falhou ao matar o host PID $($HostProc.Id) (exit $LASTEXITCODE)"
    }

    Start-Sleep -Milliseconds $SettleMs

    $Alive = Get-Process -Id $ServerPid -ErrorAction SilentlyContinue
    if ($Alive) {
        $Survivors++
        Write-Warning "[!] Ciclo ${i}: PID $ServerPid sobreviveu"
        Stop-Process -Id $ServerPid -Force -ErrorAction SilentlyContinue
    }

    if ($i % 10 -eq 0) { Write-Output "[i] $i/$Cycles" }
}

# Varredura final: nenhum gobsidian.exe pode sobreviver ao script, sob pena
# de "zero orfaos" ser mentira por causa de um PID que escapou da contagem
# por-ciclo (por exemplo, um host morto entre o poll do PID e o taskkill).
$Remaining = @(Get-Process -Name "gobsidian" -ErrorAction SilentlyContinue)
if ($Remaining.Count -gt 0) {
    Write-Warning "[!] $($Remaining.Count) processo(s) gobsidian.exe ainda em execucao"
    $Remaining | ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
    $Survivors += $Remaining.Count
}

# Amostra dos motivos observados nos logs de debug (reason=...), para
# diagnostico humano de qual mecanismo efetivamente disparou.
$LogFiles = @(Get-ChildItem -Path $WorkDir -Filter "*.log" -ErrorAction SilentlyContinue)
$Reasons = @()
if ($LogFiles.Count -gt 0) {
    # @(...) e obrigatorio aqui: se nenhum log tiver "reason=" (por exemplo,
    # quando o mecanismo sob teste esta quebrado e nada dispara a tempo), o
    # pipeline devolve $null em vez de uma colecao vazia, e sob
    # Set-StrictMode ler .Count em $null e erro fatal — o que faria o
    # proprio diagnostico de uma falha mascarar a falha original.
    $Reasons = @(Select-String -Path $LogFiles.FullName -Pattern 'reason=(\S+)' -ErrorAction SilentlyContinue |
        ForEach-Object { $_.Matches[0].Groups[1].Value } |
        Group-Object |
        Sort-Object Count -Descending)
}

if ($Reasons.Count -gt 0) {
    Write-Output "[i] motivos observados nos logs de debug:"
    $Reasons | ForEach-Object { Write-Output ("    {0}: {1}x" -f $_.Name, $_.Count) }
} else {
    Write-Warning "[!] nenhum 'reason=' encontrado nos logs de debug"
}

$Failed = ($Survivors -gt 0) -or ($LaunchFailures -gt 0) -or ($KillFailures -gt 0)

if ($Failed) {
    Write-Output "[i] work dir preservado para inspecao: $WorkDir"
} else {
    Remove-Item -Path $WorkDir -Recurse -Force -ErrorAction SilentlyContinue
}

if ($LaunchFailures -gt 0) {
    Write-Warning "[!] FALHA: $LaunchFailures ciclo(s) nao conseguiram medir nada (lancamento ou PID indeterminado)"
}
if ($KillFailures -gt 0) {
    Write-Warning "[!] FALHA: $KillFailures ciclo(s) em que o host nao pode ser morto"
}
if ($Survivors -gt 0) {
    Write-Warning "[!] FALHA: $Survivors orfao(s) em $Cycles ciclos"
}

if ($Failed) {
    exit 1
}

Write-Output "[OK] Nenhum orfao em $Cycles ciclos"
exit 0
