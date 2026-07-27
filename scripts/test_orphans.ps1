param(
    [int]$Cycles = 100,
    [string]$VaultPath,
    [int]$PidTimeoutMs = 5000,
    # 8s > o guarda-chuva de 6s de lifecycle.Shutdown. Esperar menos que o
    # orcamento que o proprio codigo se da conta encerramento lento como
    # orfao. Se este numero mudar, o de la mudou primeiro.
    [int]$SettleMs = 8000
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ($Cycles -lt 1) {
    # -Cycles 0 (ex.: uma variavel de ambiente de CI nao definida, coagida a
    # inteiro) faria o resto do script rodar zero iteracoes e ainda assim
    # imprimir [OK] no final - a mesma classe do bug de aspas que este script
    # ja teve: verde sem ter lancado servidor nenhum.
    Write-Warning "[!] Cycles precisa ser >= 1 (recebido $Cycles) - uma rodada de 0 nao lanca servidor nenhum e nao prova nada"
    exit 1
}

# Descobre filhos gobsidian.exe de um host ainda quando o poll do arquivo de
# PID estoura: orphan_host.ps1 chama Process.Start ANTES de escrever o
# arquivo (Set-Content), entao um poll estourado nao significa "nada foi
# lancado" - so que o registro por arquivo chegou tarde ou nunca chegou. Sem
# isso, aquele filho escapa da varredura final inteira: nem e contado como
# orfao, nem e morto.
function Get-ChildGobsidianPids {
    param([int]$ParentId)
    @(Get-CimInstance -ClassName Win32_Process -Filter "ParentProcessId=$ParentId" -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -eq "gobsidian.exe" } |
        ForEach-Object { $_.ProcessId })
}

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
# Cada elemento e { Id; StartTime }, nao so o PID cru: o Windows reatribui
# PIDs, e uma rodada de 100 ciclos ao longo de minutos da tempo de sobra pra
# isso acontecer. StartTime junto do PID identifica o processo especifico
# que este script lancou, nao qualquer processo que esteja ocupando aquele
# numero na hora da varredura final.
$LaunchedPids = @()
$MeasuredCycleIndices = @()

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
        # O arquivo de PID pode ter atrasado sem o filho ter deixado de
        # existir (ver comentario de Get-ChildGobsidianPids acima). Descobre
        # pelo processo pai ANTES de matar o host, senao aquele filho vira um
        # orfao invisivel para a varredura final.
        foreach ($ChildId in (Get-ChildGobsidianPids -ParentId $HostProc.Id)) {
            $ChildProc = Get-Process -Id $ChildId -ErrorAction SilentlyContinue
            if ($ChildProc) {
                $LaunchedPids += [PSCustomObject]@{ Id = $ChildId; StartTime = $ChildProc.StartTime }
            }
        }
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

    $LaunchedPids += [PSCustomObject]@{ Id = $ServerPid; StartTime = $ServerAlive.StartTime }
    $MeasuredCycleIndices += $i

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

# Varredura final: nenhum gobsidian.exe que este script lancou pode
# sobreviver a ele, sob pena de "zero orfaos" ser mentira por causa de um PID
# que escapou da contagem por-ciclo (por exemplo, um host morto entre o poll
# do PID e o taskkill, ou o cenario coberto por Get-ChildGobsidianPids acima).
#
# Restrito aos PIDs que ESTE script lancou. Varrer por nome mataria um
# "gobsidian serve" que o desenvolvedor deixou aberto noutro terminal e o
# contaria como orfao — rodada vermelha por um processo que ninguem pediu
# para medir. Mas o PID sozinho nao basta: o Windows reatribui PIDs, e um
# host morto ha minutos deixa tempo de sobra pra isso acontecer numa rodada
# longa. Confere nome + StartTime exatos do processo que foi registrado, nao
# so o numero — assim um PID reciclado para outro processo qualquer (o
# editor do desenvolvedor, por exemplo) nunca e contado nem morto.
$Remaining = @($LaunchedPids | ForEach-Object {
    $Proc = Get-Process -Id $_.Id -ErrorAction SilentlyContinue
    if ($Proc -and $Proc.ProcessName -eq "gobsidian" -and $Proc.StartTime -eq $_.StartTime) {
        $Proc
    }
})

if ($Remaining.Count -gt 0) {
    Write-Warning "[!] $($Remaining.Count) processo(s) gobsidian lancado(s) por este script ainda em execucao"
    $Remaining | ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
    $Survivors += $Remaining.Count
}

# Cada ciclo tem seu proprio log. Um ciclo sem "reason=" e um ciclo em que
# nenhum mecanismo de encerramento disparou — o filho morreu por outro
# motivo, ou nao chegou a viver. Contar o agregado nao basta: 100 motivos
# vindos de 50 ciclos passaria despercebido.
#
# Varre so os logs dos ciclos em $MeasuredCycleIndices — os que passaram das
# duas checagens de LaunchFailure e efetivamente tiveram o host morto e o
# assentamento esperado. Um ciclo de LaunchFailure pode ter um "reason=" no
# proprio log (o filho que ja estava rodando ve o EOF do handle herdado
# quando o host e morto, mesmo que o arquivo de PID nunca tenha aparecido a
# tempo) — varrer esse log tambem infla $CyclesWithReason acima do numero de
# ciclos medidos e pode derrubar $ReasonlessCycles abaixo de zero, mascarando
# o proprio diagnostico que esta contagem existe para produzir. Contar por
# indice, e nao por subtracao, elimina essa classe de erro por construcao.
$CyclesWithReason = 0
$Reasons = @()
$HardLimitCycles = 0

foreach ($Idx in $MeasuredCycleIndices) {
    $Log = Join-Path $WorkDir "cycle_$Idx.log"
    if (-not (Test-Path $Log)) { continue }

    # @(...) e obrigatorio: sem correspondencia o pipeline devolve $null, e
    # sob Set-StrictMode ler .Count em $null e erro fatal — o proprio
    # diagnostico da falha mascararia a falha original.
    $Found = @(Select-String -Path $Log -Pattern 'reason=(\S+)' -ErrorAction SilentlyContinue |
        ForEach-Object { $_.Matches[0].Groups[1].Value })
    if ($Found.Count -gt 0) {
        $CyclesWithReason++
        $Reasons += $Found
    }

    # O guarda-chuva de limite rigido (lifecycle.Shutdown, hardLimit) registra
    # esta mensagem e forca os.Exit(1) quando alguma etapa do encerramento
    # trava. Isso produz "reason=" (a etapa que travou pode ja ter sido
    # logada antes de travar) e zero orfaos — a rodada ficaria verde com o
    # encerramento ordenado quebrado e so o botao de panico tendo salvo o
    # ciclo. $SettleMs = 8000 (> o guarda-chuva de 6s) da exatamente o espaco
    # pra isso acontecer dentro da janela medida, entao esta checagem precisa
    # estar na mesma varredura.
    if (Select-String -Path $Log -Pattern 'encerramento travou alem do limite rigido' -Quiet -ErrorAction SilentlyContinue) {
        $HardLimitCycles++
    }
}

$MeasuredCycles = $MeasuredCycleIndices.Count
$ReasonlessCycles = $MeasuredCycles - $CyclesWithReason

if ($Reasons.Count -gt 0) {
    Write-Output "[i] motivos observados nos logs de debug:"
    $Reasons | Group-Object | Sort-Object Count -Descending | ForEach-Object {
        Write-Output ("    {0}: {1}x" -f $_.Name, $_.Count)
    }
}

if ($ReasonlessCycles -gt 0) {
    Write-Warning "[!] FALHA: $ReasonlessCycles de $MeasuredCycles ciclo(s) encerraram sem registrar 'reason=' - nenhum mecanismo de encerramento disparou, e zero orfaos nao prova nada nesse caso"
}
if ($HardLimitCycles -gt 0) {
    Write-Warning "[!] FALHA: $HardLimitCycles ciclo(s) so foram salvos pelo guarda-chuva de limite rigido - o encerramento ordenado esta quebrado mesmo sem orfao"
}
if ($MeasuredCycles -eq 0) {
    Write-Warning "[!] FALHA: nenhum ciclo mediu nada - todos os $Cycles ciclo(s) falharam no lancamento"
}

$Failed = ($Survivors -gt 0) -or ($LaunchFailures -gt 0) -or ($KillFailures -gt 0) `
    -or ($ReasonlessCycles -gt 0) -or ($HardLimitCycles -gt 0) -or ($MeasuredCycles -eq 0)

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
