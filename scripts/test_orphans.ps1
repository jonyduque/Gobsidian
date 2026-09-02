param(
    [int]$Cycles = 100,
    [string]$VaultPath,
    # 0 = escolhe pelo cenario. Ver o bloco logo apos os parametros.
    [int]$PidTimeoutMs = 0,
    # 8s > o guarda-chuva de 6s de lifecycle.Shutdown. Esperar menos que o
    # orcamento que o proprio codigo se da conta encerramento lento como
    # orfao. Se este numero mudar, o de la mudou primeiro.
    [int]$SettleMs = 8000,

    # Qual MECANISMO de encerramento o ciclo exercita.
    #
    #   stdin-eof     o host segura a ponta de escrita do pipe; mata-lo fecha
    #                 o pipe e o servidor ve EOF.
    #   parent-death  keeper -> host -> servidor, com o stdin herdado do
    #                 keeper. Matar o host mata o PAI do servidor sem fechar o
    #                 stdin, entao so a vigilia do pai pode encerra-lo.
    #
    # Ate 2026-08-02 so existia o primeiro, e as duas ultimas rodadas deram
    # 100/100 em "stdin-eof". A vigilia do pai nunca tinha sido exercitada
    # ponta a ponta — e ela ja falhou aqui, deixando 5 de 5 orfaos por comparar
    # (pid, creation time) num sistema em que os dois seguem consultaveis
    # depois da morte do processo.
    #
    # Teste de fallback que deixa o caminho principal ligado mede o caminho
    # principal. Por isso o cenario parent-death DESCONECTA o EOF, e o gate
    # abaixo reprova se o "reason=" nao for o do mecanismo que o cenario nomeia.
    #   signal        nada morre e nada fecha: o host fica vivo e o stdin do
    #                 servidor e um console herdado. So o CTRL_BREAK pode
    #                 encerra-lo.
    #   daemon-idle   o daemon de cofre (internal/daemon, Task 92) nao tem
    #                 pai vigiavel nem stdin de host -- os tres mecanismos
    #                 acima nao se aplicam a ele. Este cenario mata a UNICA
    #                 ponte conectada e confere que o daemon sai por
    #                 ociosidade dentro do prazo, sozinho, sem ninguem para
    #                 vigia-lo. Estrutura DIFERENTE dos outros tres (sem
    #                 host/keeper) porque o processo sob teste tambem e
    #                 diferente -- ver o bloco dedicado logo abaixo.
    #
    # "all" roda os QUATRO em sequencia, e e o padrao.
    #
    # Era "stdin-eof", e isso deu um verde que cobria uma fracao do que a
    # linha de comando parecia cobrir. O CI sempre chamou os mecanismos
    # aplicaveis explicitamente (ver .github/workflows/ci.yml), mas quem
    # rodava o comando documentado localmente exercitava so o primeiro
    # mecanismo — e concluia, com o [OK] na tela, que todos estavam
    # verificados. Um gate cujo padrao cobre parte do que ele afirma e pior
    # que um gate ausente.
    [ValidateSet("stdin-eof", "parent-death", "signal", "daemon-idle", "all")]
    [string]$Scenario = "all",

    # Teto de ciclos NAO MEDIDOS tolerados, em porcentagem de -Cycles.
    #
    # "Nao mediu" e diferente de "vazou orfao", e ate 2026-08-11 os dois
    # reprovavam igual. Um ciclo que nao chegou a lancar o processo, ou cujo
    # PID nao apareceu no prazo, nao OBSERVOU nada: nem sucesso nem
    # vazamento. Reprovar por causa dele mede a carga do runner, nao o
    # produto -- e um commit so de documentacao ja reprovou assim, com 1
    # ciclo em 300 e os outros 299 limpos.
    #
    # O risco de tolerar e esconder degradacao progressiva, por isso o
    # numero e pequeno, os ciclos perdidos sao SEMPRE impressos, e zero
    # ciclos medidos continua reprovando qualquer que seja o teto. Use 0
    # para exigir que todos os ciclos midam.
    [ValidateRange(0, 100)]
    [int]$MaxNaoMedidosPct = 2
)

# Binario mais velho que o codigo e recusado, em vez de exercitado em silencio.
#
# Este script NAO compila: ele roda o que estiver em bin\. Sem esta checagem,
# um binario obsoleto passa nos cenarios que nao dependem do codigo novo e
# reprova nos que dependem, com uma mensagem que aponta para o lugar errado.
# Medido: um binario de quatro dias antes deu tres [OK] e depois 100 falhas de
# "daemon nao anunciou prontidao" -- que parece problema de tempo no daemon, e
# era o subcomando `daemon` nao existir naquele build.
#
# Fica AQUI, antes de qualquer despacho, e nao junto da segunda definicao de
# $BinaryPath la embaixo: o bloco de daemon-idle resolve o proprio caminho do
# binario e SAI do script antes de alcancar aquele ponto. Uma guarda colocada
# depois dele passa a valer para tres cenarios e nao para o quarto -- que e
# exatamente o defeito que ela existe para impedir, cometido de novo dentro
# do proprio conserto. Medido: a primeira versao desta checagem nao disparou.
$RaizDoProjeto = Split-Path -Parent $PSScriptRoot
$BinarioDoGate = Join-Path $RaizDoProjeto "bin\gobsidian.exe"

if (Test-Path $BinarioDoGate) {
    $FonteMaisNova = Get-ChildItem -Path $RaizDoProjeto -Recurse -Include *.go, go.mod, go.sum -File -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -notmatch '\\(bin|\.git|testdata)\\' } |
        Sort-Object LastWriteTimeUtc -Descending |
        Select-Object -First 1

    if ($FonteMaisNova -and (Get-Item $BinarioDoGate).LastWriteTimeUtc -lt $FonteMaisNova.LastWriteTimeUtc) {
        Write-Warning "[!] Binario desatualizado: bin\gobsidian.exe e mais antigo que $($FonteMaisNova.Name)"
        Write-Warning "[!] Rode 'pwsh -File scripts/build.ps1' antes do gate -- este script nao compila."
        exit 1
    }
}

# "all" delega a si mesmo, um cenario por vez, em vez de reestruturar o corpo
# inteiro em laco. O corpo depende de $Scenario em uma duzia de pontos; um
# laco por dentro exigiria zerar cada contador entre voltas, e um contador
# esquecido daria verde somando ciclos do cenario anterior.
if ($Scenario -eq "all") {
    $todos = @("stdin-eof", "parent-death", "signal", "daemon-idle")
    $falhou = @()
    foreach ($cen in $todos) {
        Write-Output ""
        Write-Output "########## cenario: $cen ##########"
        # -VaultPath so e repassado quando veio preenchido: mandar string
        # vazia nao e o mesmo que omitir, e o corpo decide gerar um cofre
        # sintetico justamente quando ele esta ausente.
        $argsFilho = @{
            Cycles       = $Cycles
            PidTimeoutMs = $PidTimeoutMs
            SettleMs     = $SettleMs
            Scenario     = $cen
            MaxNaoMedidosPct = $MaxNaoMedidosPct
        }
        if ($VaultPath) { $argsFilho.VaultPath = $VaultPath }
        & $PSCommandPath @argsFilho
        if ($LASTEXITCODE -ne 0) {
            $falhou += $cen
        }
    }
    Write-Output ""
    if ($falhou.Count -gt 0) {
        Write-Warning "[!] FALHA nos cenarios: $($falhou -join ', ')"
        exit 1
    }
    Write-Output "[OK] Nenhum orfao em $Cycles ciclos nos quatro mecanismos: $($todos -join ', ')"
    exit 0
}

# daemon-idle e estruturalmente diferente dos outros tres: nao ha host nem
# keeper, porque o daemon nao tem pai vigiavel (e o ponto do cenario). O
# bloco abaixo lanca o daemon e uma ponte diretamente, mata so a ponte, e
# confere que o daemon -- sozinho, sem pai, sem stdin de host -- sai por
# ociosidade dentro do prazo com "reason=idle". Sai (exit) daqui: nao entra
# no laco generico abaixo, que assume a forma host->servidor que as outras
# tres cenarios compartilham.
if ($Scenario -eq "daemon-idle") {
    Set-StrictMode -Version Latest
    $ErrorActionPreference = "Stop"

    if ($Cycles -lt 1) {
        Write-Warning "[!] Cycles precisa ser >= 1 (recebido $Cycles) - uma rodada de 0 nao lanca daemon nenhum e nao prova nada"
        exit 1
    }

    $ProjectRoot = Split-Path -Parent $PSScriptRoot
    $BinaryPath = Join-Path $ProjectRoot "bin\gobsidian.exe"
    if (-not (Test-Path $BinaryPath)) {
        Write-Warning "[!] Binario nao encontrado: $BinaryPath"
        exit 1
    }
    if (-not $VaultPath) {
        $VaultPath = Join-Path $ProjectRoot "testdata\vault_small"
    }

    $WorkDir = Join-Path $env:TEMP ("gobsidian_orphan_daemonidle_" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $WorkDir | Out-Null
    Write-Output "[...] $Cycles ciclos de ociosidade do daemon (sem cliente conectado, sem pai vigiavel)"
    Write-Output "[i] logs em $WorkDir"

    # Curto de proposito, so para o gate nao levar 15 minutos por ciclo. O
    # PADRAO de producao (internal/daemon.DefaultIdleSeconds, 15 minutos)
    # nunca muda por causa disto: --idle-seconds e passado explicitamente
    # aqui, exatamente o que o brief da Task 92 pede ("confira que o padrao
    # continua 15 minutos" -- um padrao de teste vazado para producao
    # mataria o daemon no meio do uso real).
    $IdleSeconds = 3
    # > 4x IdleSeconds (o intervalo de checagem do daemon e OciosidadeMax/4)
    # mais folga para o handshake da ponte, o taskkill, e o encerramento
    # ordenado do daemon (ate o hardLimit de 6s de lifecycle.Shutdown).
    $DaemonSettleMs = 12000
    $ReadyTimeoutMs = 5000

    $Survivors = 0
    $LaunchFailures = 0
    $WrongReasonCycles = 0
    $MeasuredCycles = 0
    $LaunchedDaemonPids = @()

    # PRECONDICAO: nenhum daemon vivo neste cofre.
    #
    # Os tres cenarios anteriores rodam `serve`, e `serve` SPAWNA um daemon --
    # com a ociosidade PADRAO de 900 s, nao a curta que este cenario usa. Esse
    # daemon sobrevive ao cenario que o criou e segura o socket do cofre.
    #
    # Ate 2026-08-26 isso passava despercebido porque ipc.Listen ROUBAVA o
    # socket do daemon vivo: o daemon novo bindava por cima, e o de 900 s
    # ficava vivo e inalcancavel, com indice em memoria, ate expirar. Era o
    # defeito A5 acontecendo dentro do proprio gate, sem ninguem ver. Com a
    # prova de orfao no lugar, o daemon deste cenario recusa subir -- e a
    # recusa esta certa; o que faltava era a precondicao.
    #
    # Encerra por PID e so o que casa ESTE cofre. Matar por nome ja destruiu a
    # sessao real do usuario uma vez (ver docs/ARMADILHAS.md).
    $Pendentes = @(Get-CimInstance Win32_Process -Filter "Name='gobsidian.exe'" -ErrorAction SilentlyContinue |
        Where-Object { $_.CommandLine -match ' daemon ' -and $_.CommandLine -match [regex]::Escape($VaultPath) })
    if ($Pendentes.Count -gt 0) {
        Write-Output "[i] encerrando $($Pendentes.Count) daemon(s) deixado(s) pelos cenarios anteriores neste cofre"
        $Pendentes | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
        $Limite = [DateTime]::UtcNow.AddSeconds(10)
        while ([DateTime]::UtcNow -lt $Limite) {
            $Restantes = @(Get-CimInstance Win32_Process -Filter "Name='gobsidian.exe'" -ErrorAction SilentlyContinue |
                Where-Object { $_.CommandLine -match ' daemon ' -and $_.CommandLine -match [regex]::Escape($VaultPath) })
            if ($Restantes.Count -eq 0) { break }
            Start-Sleep -Milliseconds 100
        }
    }

    for ($i = 1; $i -le $Cycles; $i++) {
        $DaemonLog = Join-Path $WorkDir "cycle_$i.daemon.log"

        $DaemonProc = Start-Process -FilePath $BinaryPath `
            -ArgumentList "daemon", "--vault", $VaultPath, "--idle-seconds", $IdleSeconds, "--log-level", "debug" `
            -PassThru -WindowStyle Hidden `
            -RedirectStandardError $DaemonLog

        if (-not $DaemonProc) {
            $LaunchFailures++
            Write-Warning "[!] Ciclo ${i}: Start-Process nao devolveu um processo daemon"
            continue
        }

        # Poll do proprio log (redirecionado do stderr do processo, ver o
        # comentario de novoLoggerDoDaemon em cmd/gobsidian/daemon.go) ate
        # "daemon iniciado" aparecer -- o socket esta aberto e pronto.
        $Deadline = [DateTime]::UtcNow.AddMilliseconds($ReadyTimeoutMs)
        $DaemonPronto = $false
        while ([DateTime]::UtcNow -lt $Deadline) {
            if ((Test-Path $DaemonLog) -and (Select-String -Path $DaemonLog -Pattern 'daemon iniciado' -Quiet -ErrorAction SilentlyContinue)) {
                $DaemonPronto = $true
                break
            }
            Start-Sleep -Milliseconds 50
        }
        if (-not $DaemonPronto) {
            $LaunchFailures++
            Write-Warning "[!] Ciclo ${i}: daemon nao anunciou prontidao (linha 'daemon iniciado') em ${ReadyTimeoutMs}ms"
            Stop-Process -Id $DaemonProc.Id -Force -ErrorAction SilentlyContinue
            continue
        }
        $DaemonAlive = Get-Process -Id $DaemonProc.Id -ErrorAction SilentlyContinue
        if (-not $DaemonAlive) {
            $LaunchFailures++
            Write-Warning "[!] Ciclo ${i}: daemon anunciou prontidao mas o processo ja nao existe"
            continue
        }
        $LaunchedDaemonPids += [PSCustomObject]@{ Id = $DaemonProc.Id; StartTime = $DaemonAlive.StartTime }

        # A ponte: harness segura a ponta de escrita do stdin (mesmo papel
        # que orphan_host.ps1 tem nos outros cenarios, so que aqui o proprio
        # harness e o host -- nao ha keeper nem indirecao, porque o que se
        # mata em seguida e a ponte, nao um ancestral dela).
        $psi = New-Object System.Diagnostics.ProcessStartInfo
        $psi.FileName = $BinaryPath
        $psi.ArgumentList.Add("serve")
        $psi.ArgumentList.Add("--vault")
        $psi.ArgumentList.Add($VaultPath)
        $psi.ArgumentList.Add("--log-level")
        $psi.ArgumentList.Add("debug")
        $psi.UseShellExecute = $false
        $psi.RedirectStandardInput = $true
        $psi.RedirectStandardError = $true

        $BridgeProc = [System.Diagnostics.Process]::Start($psi)

        # Le o stderr da ponte LINHA A LINHA (Peek/ReadLine, nunca
        # ReadToEnd*) ate "conectado ao daemon via socket" aparecer ou o
        # prazo estourar -- ReadToEndAsync so completa no EOF do processo, e
        # a ponte fica viva de proposito ate ser morta abaixo.
        # ReadLineAsync com espera limitada, NUNCA Peek(). StreamReader.Peek()
        # BLOQUEIA num pipe sem dados disponiveis -- ele nao e uma consulta
        # sem espera, como o nome sugere. Com ele, o teste do prazo logo acima
        # deixava de ser alcancado e o laco ficava presto para sempre: medido,
        # o ciclo 82 de uma rodada de 100 ficou 15h44m parado, com o daemon
        # vivo e ativos=1, ate alguem matar a ponte a mao. Um gate que pode
        # travar indefinidamente acaba sendo pulado, e gate pulado nao e gate.
        $BridgeConectou = $false
        $BridgeDeadline = [DateTime]::UtcNow.AddMilliseconds($ReadyTimeoutMs)
        $sr = $BridgeProc.StandardError
        while ($true) {
            $RestanteMs = [int]([DateTime]::UtcNow - $BridgeDeadline).TotalMilliseconds * -1
            if ($RestanteMs -le 0) { break }

            $Leitura = $sr.ReadLineAsync()
            if (-not $Leitura.Wait($RestanteMs)) { break }

            $line = $Leitura.Result
            if ($null -eq $line) { break }
            if ($line -match 'conectado ao daemon via socket') {
                $BridgeConectou = $true
                break
            }
        }

        if (-not $BridgeConectou) {
            $LaunchFailures++
            Write-Warning "[!] Ciclo ${i}: a ponte nao confirmou conexao ao daemon via socket em ${ReadyTimeoutMs}ms -- ela pode ter caido para o modo em processo, o que nao exercitaria a ociosidade do daemon"
            Stop-Process -Id $BridgeProc.Id -Force -ErrorAction SilentlyContinue
            Stop-Process -Id $DaemonProc.Id -Force -ErrorAction SilentlyContinue
            continue
        }

        $MeasuredCycles++

        # Mata SO a ponte -- "matar todas as pontes" com N=1 pontes. O
        # daemon nao tem pai; nada mais precisa morrer para ele ficar sem
        # nenhum cliente.
        taskkill /F /PID $BridgeProc.Id 2>$null | Out-Null

        Start-Sleep -Milliseconds $DaemonSettleMs

        $DaemonDepois = Get-Process -Id $DaemonProc.Id -ErrorAction SilentlyContinue
        if ($DaemonDepois -and $DaemonDepois.ProcessName -eq "gobsidian" -and $DaemonDepois.StartTime -eq $DaemonAlive.StartTime) {
            $Survivors++
            Write-Warning "[!] Ciclo ${i}: daemon PID $($DaemonProc.Id) sobreviveu alem de ${DaemonSettleMs}ms sem cliente"
            Stop-Process -Id $DaemonProc.Id -Force -ErrorAction SilentlyContinue
        }
        else {
            # Reason= tem de ser "idle" -- a mesma regra dos outros tres
            # cenarios: encerrar pelo motivo certo por acidente nao conta.
            $Found = @(Select-String -Path $DaemonLog -Pattern 'reason=(\S+)' -ErrorAction SilentlyContinue |
                ForEach-Object { $_.Matches[0].Groups[1].Value })
            if ($Found.Count -eq 0) {
                $WrongReasonCycles++
                Write-Warning "[!] Ciclo ${i}: daemon encerrou sem registrar 'reason=' -- nenhum mecanismo nomeado disparou"
            }
            elseif ($Found[0] -ne "idle") {
                $WrongReasonCycles++
                Write-Warning "[!] Ciclo ${i}: reason=$($Found[0]), esperado 'idle'"
            }
        }

        if ($i % 10 -eq 0) { Write-Output "[i] $i/$Cycles" }
    }

    $Remaining = @($LaunchedDaemonPids | ForEach-Object {
        $Proc = Get-Process -Id $_.Id -ErrorAction SilentlyContinue
        if ($Proc -and $Proc.ProcessName -eq "gobsidian" -and $Proc.StartTime -eq $_.StartTime) {
            $Proc
        }
    })
    if ($Remaining.Count -gt 0) {
        Write-Warning "[!] $($Remaining.Count) processo(s) gobsidian (daemon) lancado(s) por este script ainda em execucao"
        $Remaining | ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
        $Survivors += $Remaining.Count
    }

    if ($MeasuredCycles -eq 0) {
        Write-Warning "[!] FALHA: nenhum ciclo mediu nada - todos os $Cycles ciclo(s) falharam no lancamento"
    }

    # Mesma regra do laco generico (ver o comentario de -MaxNaoMedidosPct):
    # ciclo que nao lancou nao observou nada, e reprovar por causa dele mede a
    # carga da maquina. A perda sai impressa de qualquer jeito.
    $TetoNaoMedidos = [math]::Floor($Cycles * $MaxNaoMedidosPct / 100)
    if ($LaunchFailures -gt 0) {
        Write-Output "[i] $LaunchFailures de $Cycles ciclo(s) nao mediram nada (lancamento, prontidao ou conexao da ponte); teto tolerado: $TetoNaoMedidos"
    }
    if ($LaunchFailures -gt $TetoNaoMedidos) {
        Write-Warning "[!] FALHA: $LaunchFailures de $Cycles ciclo(s) nao conseguiram medir nada, acima do teto de $TetoNaoMedidos"
    }

    $Failed = ($Survivors -gt 0) -or ($LaunchFailures -gt $TetoNaoMedidos) -or ($WrongReasonCycles -gt 0) -or ($MeasuredCycles -eq 0)

    if ($Failed) {
        Write-Output "[i] work dir preservado para inspecao: $WorkDir"
    }
    else {
        Remove-Item -Path $WorkDir -Recurse -Force -ErrorAction SilentlyContinue
    }

    if ($WrongReasonCycles -gt 0) {
        Write-Warning "[!] FALHA: $WrongReasonCycles de $MeasuredCycles ciclo(s) nao encerraram com reason=idle"
    }
    if ($Survivors -gt 0) {
        Write-Warning "[!] FALHA: $Survivors orfao(s) de daemon em $Cycles ciclos"
    }

    if ($Failed) {
        exit 1
    }

    Write-Output "[OK] Nenhum daemon orfao em $Cycles ciclos -- todos sairam por ociosidade (reason=idle) apos a unica ponte ser morta"
    exit 0
}

# O motivo que o cenario TEM de produzir. Ciclo que encerra pelo motivo errado
# nao testou o mecanismo que o cenario nomeia — encerrou por outra coisa e deu
# verde.
# O cenario stdin-eof lanca DOIS processos ate o servidor (harness -> host ->
# servidor); parent-death e signal lancam TRES, e cada nivel e um pwsh, que
# custa cerca de um segundo para subir. Num runner de 2 vCPU o poll de 5 s
# estourou em 9 de 100 ciclos do parent-death — sem defeito nenhum no
# mecanismo, que acertou o motivo nos 91 restantes e nao deixou orfao. Um
# valor explicito na linha de comando continua vencendo.
if ($PidTimeoutMs -le 0) {
    $PidTimeoutMs = if ($Scenario -eq "stdin-eof") { 5000 } else { 20000 }
}

$ReasonEsperado = switch ($Scenario) {
    "stdin-eof" { "stdin-eof" }
    "parent-death" { "parent-gone" }
    "signal" { "signal" }
}

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
$KeeperScript = Join-Path $PSScriptRoot "orphan_keeper.ps1"
$SignalHostScript = Join-Path $PSScriptRoot "orphan_signal_host.ps1"
$SignalSendScript = Join-Path $PSScriptRoot "orphan_signal_send.ps1"

if (-not (Test-Path $BinaryPath)) {
    Write-Warning "[!] Binario nao encontrado: $BinaryPath"
    exit 1
}

if (-not (Test-Path $HostScript)) {
    Write-Warning "[!] Script do host nao encontrado: $HostScript"
    exit 1
}
if ($Scenario -eq "parent-death" -and -not (Test-Path $KeeperScript)) {
    Write-Warning "[!] Script do keeper nao encontrado: $KeeperScript"
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
$WrongReasonCycles = 0
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
    # No cenario parent-death o processo lancado aqui e o KEEPER, e quem tem de
    # morrer no meio do ciclo e o HOST que ele cria. $VictimPid guarda quem
    # matar; $HostProc guarda quem limpar no fim.
    if ($Scenario -eq "parent-death") {
        $HostPidFile = Join-Path $WorkDir "cycle_$i.hostpid"
        $HostProc = Start-Process -FilePath "pwsh.exe" `
            -ArgumentList "-NoProfile", "-NonInteractive", "-File", $KeeperScript, `
                          "-HostScript", $HostScript, "-BinaryPath", $BinaryPath, `
                          "-VaultPath", $VaultPath, "-PidFile", $PidFile, "-HostPidFile", $HostPidFile `
            -PassThru -WindowStyle Hidden `
            -RedirectStandardError $LogFile
    }
    elseif ($Scenario -eq "signal") {
        $HostPidFile = $null
        # -NoNewWindow, e nao -WindowStyle Hidden: o host precisa COMPARTILHAR o
        # console deste harness, senao o servidor herda um console ao qual o
        # processo sinalizador nao consegue se ligar.
        $HostProc = Start-Process -FilePath "pwsh.exe" `
            -ArgumentList "-NoProfile", "-NonInteractive", "-File", $SignalHostScript, `
                          "-BinaryPath", $BinaryPath, "-VaultPath", $VaultPath, "-PidFile", $PidFile `
            -PassThru -NoNewWindow `
            -RedirectStandardError $LogFile
    }
    else {
        $HostPidFile = $null
        $HostProc = Start-Process -FilePath "pwsh.exe" `
            -ArgumentList "-NoProfile", "-NonInteractive", "-File", $HostScript, `
                          "-BinaryPath", $BinaryPath, "-VaultPath", $VaultPath, "-PidFile", $PidFile `
            -PassThru -WindowStyle Hidden `
            -RedirectStandardError $LogFile
    }

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

    # Mata o PAI do servidor, NAO o servidor, e sem /T. Matar a arvore inteira
    # provaria so que o taskkill sabe matar processos, nao que o lifecycle do
    # filho funciona.
    #
    # No stdin-eof o pai e o host, que o harness lancou. No parent-death o
    # harness lancou o keeper, e o pai do servidor e o host que o keeper criou:
    # matar o keeper aqui fecharia a ponta de escrita do pipe e o ciclo voltaria
    # a encerrar por stdin-eof — que e exatamente o mecanismo que este cenario
    # existe para desconectar.
    $VictimPid = $HostProc.Id
    if ($Scenario -eq "parent-death") {
        $VictimPid = $null
        $HostDeadline = [DateTime]::UtcNow.AddMilliseconds($PidTimeoutMs)
        while ([DateTime]::UtcNow -lt $HostDeadline) {
            if (Test-Path $HostPidFile) {
                $HostContent = Get-Content -Path $HostPidFile -Raw -ErrorAction SilentlyContinue
                if ($HostContent -and $HostContent.Trim() -match '^\d+$') {
                    $VictimPid = [int]$HostContent.Trim()
                    break
                }
            }
            Start-Sleep -Milliseconds 50
        }
        if (-not $VictimPid) {
            $KillFailures++
            Write-Warning "[!] Ciclo ${i}: PID do host intermediario nao apareceu em ${PidTimeoutMs}ms"
            Stop-Process -Id $HostProc.Id -Force -ErrorAction SilentlyContinue
            continue
        }
    }

    if ($Scenario -eq "signal") {
        # Nada e morto aqui. O host FICA VIVO — se ele morresse, a vigilia do
        # pai dispararia e o ciclo mediria o mecanismo errado.
        & pwsh.exe -NoProfile -NonInteractive -File $SignalSendScript -TargetPid $ServerPid
        if ($LASTEXITCODE -ne 0) {
            $KillFailures++
            Write-Warning "[!] Ciclo ${i}: envio de CTRL_BREAK ao PID $ServerPid falhou (exit $LASTEXITCODE)"
        }
    }
    else {
        taskkill /F /PID $VictimPid 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            $KillFailures++
            Write-Warning "[!] Ciclo ${i}: taskkill falhou ao matar o pai PID $VictimPid (exit $LASTEXITCODE)"
        }
    }

    # Espera o processo SAIR, ate o prazo -- e nao dorme o prazo inteiro.
    #
    # O prazo do parent-death precisa caber DUAS coisas em serie, e a versao
    # anterior contava so uma. A vigilia do pai consulta a cada 5 s
    # (lifecycle.ParentCheckInterval), entao um pai que morre logo depois de uma
    # consulta leva ate 5 s para ser notado; so entao comeca o encerramento, que
    # tem guarda-chuva de 6 s. Pior caso 11 s, contra os 8 s que a janela dava --
    # e o comentario que a dimensionava dizia "> o guarda-chuva de 6s", sem somar
    # a deteccao. Resultado medido no runner do CI: 2 orfaos em 100 ciclos, com
    # o produto dentro do orcamento dele.
    #
    # Esperar por EVENTO em vez de por relogio tambem devolve tempo: o caso comum
    # sai em ~1 s, e antes pagava a janela inteira em todos os 100 ciclos.
    $PrazoMs = $SettleMs
    if ($Scenario -eq "parent-death") { $PrazoMs = $SettleMs + 7000 }
    $Relogio = [System.Diagnostics.Stopwatch]::StartNew()
    while ($Relogio.ElapsedMilliseconds -lt $PrazoMs) {
        $P = Get-Process -Id $ServerPid -ErrorAction SilentlyContinue
        if (-not ($P -and $P.ProcessName -eq "gobsidian" -and $P.StartTime -eq $ServerAlive.StartTime)) { break }
        Start-Sleep -Milliseconds 150
    }

    # Mesma checagem de identidade da varredura final (linhas 174-179): so o
    # PID reaparecer nao basta. O filho pode ter morrido nos ~100ms apos o
    # taskkill e deixado o numero livre pelo resto de $SettleMs — uma rodada
    # de 100 ciclos cria 200+ processos nesse intervalo, entao um PID
    # reatribuido a outro processo qualquer seria contado como orfao e levaria
    # Stop-Process -Force.
    $Alive = Get-Process -Id $ServerPid -ErrorAction SilentlyContinue
    if ($Alive -and $Alive.ProcessName -eq "gobsidian" -and $Alive.StartTime -eq $ServerAlive.StartTime) {
        $Survivors++
        Write-Warning "[!] Ciclo ${i}: PID $ServerPid sobreviveu"
        Stop-Process -Id $ServerPid -Force -ErrorAction SilentlyContinue
    }

    # No parent-death o keeper continua vivo de proposito ate aqui: e ele que
    # segura a ponta de escrita do stdin. Deixa-lo para tras vazaria um pwsh
    # por ciclo.
    if ($Scenario -eq "parent-death" -or $Scenario -eq "signal") {
        Stop-Process -Id $HostProc.Id -Force -ErrorAction SilentlyContinue
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
        # Um ciclo que encerra pelo motivo ERRADO nao exercitou o mecanismo que
        # o cenario nomeia. Sem esta conferencia, bastava o parent-death cair em
        # stdin-eof — que e como a lacuna sobreviveu a todos os marcos: o
        # harness contava "encerrou" e nunca "encerrou por qual mecanismo".
        if ($Found[0] -ne $ReasonEsperado) {
            $WrongReasonCycles++
            Write-Warning "[!] Ciclo ${Idx}: reason=$($Found[0]), esperado '$ReasonEsperado' no cenario '$Scenario'"
        }
    }

    # O guarda-chuva de limite rigido (lifecycle.Shutdown, hardLimit) registra
    # esta mensagem e forca os.Exit(1) quando alguma etapa do encerramento
    # trava. Isso produz "reason=" (a etapa que travou pode ja ter sido
    # logada antes de travar) e zero orfaos — a rodada ficaria verde com o
    # encerramento ordenado quebrado e so o botao de panico tendo salvo o
    # ciclo. A janela medida (8 s, e 15 s no parent-death, onde a deteccao pode
    # levar 5 s antes de o encerramento comecar) da o espaco pra isso acontecer
    # dentro dela, entao esta checagem precisa estar na mesma varredura.
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
if ($WrongReasonCycles -gt 0) {
    Write-Warning "[!] FALHA: $WrongReasonCycles de $MeasuredCycles ciclo(s) encerraram por um motivo diferente de '$ReasonEsperado' - o cenario '$Scenario' nao exercitou o mecanismo que nomeia"
}

# Teto de ciclos nao medidos. Piso zero quando -Cycles e pequeno: com 5
# ciclos, um perdido e 20% da amostra e pode muito bem estar escondendo o
# defeito que o cenario procura.
$TetoNaoMedidos = [math]::Floor($Cycles * $MaxNaoMedidosPct / 100)

# Impresso SEMPRE que houver perda, tolerada ou nao. Corte silencioso e o que
# faz um gate parecer cobrir mais do que cobriu.
if ($LaunchFailures -gt 0) {
    Write-Output "[i] $LaunchFailures de $Cycles ciclo(s) nao mediram nada (lancamento ou PID indeterminado); teto tolerado: $TetoNaoMedidos"
}

$Failed = ($Survivors -gt 0) -or ($LaunchFailures -gt $TetoNaoMedidos) -or ($KillFailures -gt 0) `
    -or ($ReasonlessCycles -gt 0) -or ($HardLimitCycles -gt 0) -or ($MeasuredCycles -eq 0) `
    -or ($WrongReasonCycles -gt 0)

if ($Failed) {
    Write-Output "[i] work dir preservado para inspecao: $WorkDir"
} else {
    Remove-Item -Path $WorkDir -Recurse -Force -ErrorAction SilentlyContinue
}

if ($LaunchFailures -gt $TetoNaoMedidos) {
    Write-Warning "[!] FALHA: $LaunchFailures de $Cycles ciclo(s) nao conseguiram medir nada (lancamento ou PID indeterminado), acima do teto de $TetoNaoMedidos"
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
