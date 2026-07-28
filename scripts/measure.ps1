#Requires -Version 7.0
<#
.SYNOPSIS
    Mede indexacao a frio (RNF-01) e RSS em repouso (RNF-07) contra um cofre.

.DESCRIPTION
    Existe para que os numeros de docs/OPERACAO.md sejam medidos, e nao
    estimados. Uma versao anterior daquela tabela trazia "ex: 408ms em teste
    local" e "tende a ficar ~30-45 MB" — um exemplo ilustrativo e uma
    expectativa, apresentados como resultado. Este script produz o numero real
    ou nao produz nada.

    Duas medicoes, com recortes diferentes e deliberados:

    index_ms  vem do proprio servidor, do log "servidor pronto". Recorta so a
              construcao do indice — nao inclui boot do runtime do Go, leitura
              de config nem handshake do MCP. E o que RNF-01 nomeia.

    RSS       e o WorkingSet64 do processo depois de o indice estar pronto, do
              handshake completo e de um periodo de acomodacao. Amostrado
              varias vezes; o script reporta o MAIOR valor observado, nao o
              ultimo. Um pico que cabe no orcamento e informacao; um pico que
              nao cabe, mascarado por uma amostra tardia, e ficcao.

    O processo e encerrado fechando o stdin, que e o mesmo caminho que um host
    MCP usa. Matar o processo mediria o desligamento errado.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Vault,

    [string]$Binary = (Join-Path $PSScriptRoot ".." "bin" "gobsidian.exe"),

    # Tempo de acomodacao antes de amostrar RSS. O runtime do Go devolve
    # memoria ao sistema operacional de forma preguicosa, e amostrar cedo
    # demais mede o pico da indexacao, nao o repouso.
    [int]$SettleMs = 3000,

    [int]$Samples = 5,

    # Um servidor que nunca imprime "servidor pronto" e uma falha, nao uma
    # espera. Sem teto, este script ficaria pendurado para sempre.
    [int]$ReadyTimeoutMs = 60000
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $Binary)) {
    throw "[!] Binario nao encontrado em $Binary - rode scripts/build.ps1 antes"
}
if (-not (Test-Path $Vault -PathType Container)) {
    throw "[!] Cofre nao encontrado em $Vault"
}

$Binary = (Resolve-Path $Binary).Path
$Vault = (Resolve-Path $Vault).Path

Write-Output "[...] medindo contra $Vault"

$psi = [System.Diagnostics.ProcessStartInfo]::new()
$psi.FileName = $Binary
$psi.ArgumentList.Add("serve")
$psi.ArgumentList.Add("--vault")
$psi.ArgumentList.Add($Vault)
$psi.RedirectStandardInput = $true
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$psi.UseShellExecute = $false

$proc = [System.Diagnostics.Process]::Start($psi)
if (-not $proc) { throw "[!] Process.Start nao devolveu um processo" }

try {
    # O log "servidor pronto" so sai depois de idx.Build terminar. Ler ate ele
    # e o que separa "indice pronto" de "processo existe".
    $ReadyLine = $null
    $Deadline = [DateTime]::UtcNow.AddMilliseconds($ReadyTimeoutMs)
    while ([DateTime]::UtcNow -lt $Deadline) {
        $line = $proc.StandardError.ReadLine()
        if ($null -eq $line) { break }
        Write-Verbose $line
        if ($line -match 'servidor pronto') { $ReadyLine = $line; break }
    }

    if (-not $ReadyLine) {
        throw "[!] servidor nao ficou pronto em ${ReadyTimeoutMs}ms - nada foi medido"
    }

    # slog.TextHandler emite chave=valor. Nao inventar numero: se o campo nao
    # estiver la, dizer que nao estava.
    $IndexMs = if ($ReadyLine -match 'index_ms=(\d+)') { [int]$Matches[1] } else { $null }
    $Notes = if ($ReadyLine -match 'notes=(\d+)') { [int]$Matches[1] } else { $null }
    $Assets = if ($ReadyLine -match 'assets=(\d+)') { [int]$Matches[1] } else { $null }

    # Handshake MCP completo antes de medir repouso: o servidor em repouso e o
    # servidor que ja atendeu, nao o que nunca recebeu nada.
    $Init = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"measure","version":"1.0"}}}'
    $Inited = '{"jsonrpc":"2.0","method":"notifications/initialized"}'
    $Stats = '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"vault_stats","arguments":{"include_runtime":true}}}'

    $proc.StandardInput.WriteLine($Init)
    $proc.StandardInput.Flush()
    $null = $proc.StandardOutput.ReadLine()

    $proc.StandardInput.WriteLine($Inited)
    $proc.StandardInput.Flush()

    $proc.StandardInput.WriteLine($Stats)
    $proc.StandardInput.Flush()
    $StatsLine = $proc.StandardOutput.ReadLine()

    Start-Sleep -Milliseconds $SettleMs

    # Maior amostra, nao a ultima. Ver comentario no cabecalho.
    $PeakBytes = 0
    for ($i = 0; $i -lt $Samples; $i++) {
        $proc.Refresh()
        if ($proc.HasExited) { throw "[!] servidor morreu durante a medicao - nada foi medido" }
        if ($proc.WorkingSet64 -gt $PeakBytes) { $PeakBytes = $proc.WorkingSet64 }
        Start-Sleep -Milliseconds 200
    }
    $RssMB = [math]::Round($PeakBytes / 1MB, 1)

    Write-Output ""
    Write-Output "=== medicao ==="
    Write-Output ("    cofre           : {0}" -f $Vault)
    Write-Output ("    notas / anexos  : {0} / {1}" -f $Notes, $Assets)
    if ($null -ne $IndexMs) {
        Write-Output ("    RNF-01 indice   : {0} ms (alvo <= 3000)" -f $IndexMs)
    } else {
        Write-Output "    RNF-01 indice   : NAO MEDIDO (campo index_ms ausente do log)"
    }
    Write-Output ("    RNF-07 RSS pico : {0} MB (alvo <= 60)" -f $RssMB)
    Write-Output ("    maquina         : {0} / {1} nucleos" -f $env:COMPUTERNAME, [Environment]::ProcessorCount)
    Write-Output ("    binario         : {0}" -f $Binary)
    Write-Verbose "vault_stats: $StatsLine"

    $Fail = $false
    if ($null -ne $IndexMs -and $IndexMs -gt 3000) {
        Write-Warning "[!] RNF-01 estourado: ${IndexMs}ms > 3000ms"
        $Fail = $true
    }
    if ($RssMB -gt 60) {
        Write-Warning "[!] RNF-07 estourado: ${RssMB}MB > 60MB"
        $Fail = $true
    }
    if ($Fail) {
        Write-Output "[!] Alvo estourado - registre o numero assim mesmo em docs/OPERACAO.md"
    } else {
        Write-Output "[OK] Dentro dos alvos"
    }
}
finally {
    # Fechar o stdin e o caminho que um host MCP usa. O servidor tem tres
    # mecanismos de encerramento e este exercita o primeiro deles.
    if (-not $proc.HasExited) {
        $proc.StandardInput.Close()
        if (-not $proc.WaitForExit(10000)) {
            Write-Warning "[!] servidor nao encerrou em 10s apos EOF de stdin - matando"
            $proc.Kill()
        }
    }
    $proc.Dispose()
}
