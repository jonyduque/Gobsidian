param(
    [Parameter(Mandatory = $true)][string]$HostScript,
    [Parameter(Mandatory = $true)][string]$BinaryPath,
    [Parameter(Mandatory = $true)][string]$VaultPath,
    [Parameter(Mandatory = $true)][string]$PidFile,
    [Parameter(Mandatory = $true)][string]$HostPidFile
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# O KEEPER existe por um motivo so: manter o stdin do servidor ABERTO depois
# que o pai dele morre.
#
# No cenario stdin-eof, quem segura a ponta de escrita do pipe e o proprio
# host; matar o host fecha o pipe e o EOF chega antes de qualquer outro
# mecanismo. Foi por isso que 100/100 ciclos das duas ultimas rodadas
# terminaram em "stdin-eof" e a vigilia do pai nunca foi exercitada ponta a
# ponta -- exatamente a lacuna registrada em CLAUDE.md.
#
# A cadeia aqui e keeper -> host -> gobsidian:
#
#   1. O keeper lanca o HOST com RedirectStandardInput. A ponta de ESCRITA do
#      pipe fica com o keeper; o host recebe a ponta de leitura como stdin.
#   2. O host lanca o gobsidian SEM redirecionar stdin. No Windows o filho
#      herda o handle de stdin do pai: o gobsidian passa a ler da MESMA ponta
#      de leitura.
#   3. Matar o host fecha a copia de handle DELE. O gobsidian tem a propria
#      copia, e o keeper continua segurando a ponta de escrita -- entao NAO ha
#      EOF.
#
# O pai do gobsidian morreu e o stdin continua aberto. So sobra a vigilia do
# pai para encerrar. E teste de fallback com o caminho principal DESCONECTADO,
# que e a unica forma de teste de fallback que mede alguma coisa: o
# TestOverflowReconciliationFull deste projeto injetava overflow com o watcher
# ativo, os eventos comuns aplicavam as mudancas, e o reconciliador podia ser
# removido inteiro sem a suite reclamar.

$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = (Get-Process -Id $PID).Path
$psi.ArgumentList.Add("-NoProfile")
$psi.ArgumentList.Add("-NonInteractive")
$psi.ArgumentList.Add("-File")
$psi.ArgumentList.Add($HostScript)
$psi.ArgumentList.Add("-BinaryPath")
$psi.ArgumentList.Add($BinaryPath)
$psi.ArgumentList.Add("-VaultPath")
$psi.ArgumentList.Add($VaultPath)
$psi.ArgumentList.Add("-PidFile")
$psi.ArgumentList.Add($PidFile)
$psi.ArgumentList.Add("-InheritStdin")
$psi.UseShellExecute = $false
# stderr NAO e redirecionado: o host herda o stderr DESTE processo, que o
# harness apontou para o arquivo de log do ciclo, e o servidor herda dele. O
# handle e de arquivo e sobrevive a morte do host — que e quando o servidor
# escreve a linha "reason=" que o gate confere.
# A ponta de escrita fica AQUI, e este processo sobrevive a morte do host.
$psi.RedirectStandardInput = $true

$hostProc = [System.Diagnostics.Process]::Start($psi)
Set-Content -Path $HostPidFile -Value $hostProc.Id -NoNewline -Encoding ascii

# Nunca escreve nada no pipe. Escrever daria ao servidor bytes de JSON-RPC
# invalidos; o que se quer e um stdin que existe e nao fecha.
while ($true) {
    Start-Sleep -Milliseconds 500
}
