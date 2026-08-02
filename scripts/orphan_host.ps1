param(
    [Parameter(Mandatory = $true)][string]$BinaryPath,
    [Parameter(Mandatory = $true)][string]$VaultPath,
    [Parameter(Mandatory = $true)][string]$PidFile,

    # Cenario parent-death: em vez de criar o proprio pipe de stdin, PASSA
    # ADIANTE o stdin que este processo recebeu do keeper. O filho herda o
    # handle, e a ponta de escrita fica com o keeper — que sobrevive a morte
    # deste processo. Sem isso, matar este host fecharia o pipe e o servidor
    # encerraria por stdin-eof antes de a vigilia do pai ter chance.
    #
    # Tambem redireciona stderr para arquivo: sem o RedirectStandardError que
    # o harness aplica no cenario stdin-eof, o log do filho nao chegaria a
    # lugar nenhum, e o gate confere o "reason=" no log.
    [switch]$InheritStdin
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Este processo E o "host sintetico". Ele reproduz o que Claude Desktop faz
# de verdade: constroi o filho com um PIPE de verdade no stdin (nao um
# console herdado) e SEGURA a ponta de escrita pelo tempo de vida da sessao.
# E o fechamento dessa ponta — quando este processo morre — que deve
# produzir o EOF que o lifecycle observa. O harness anterior lancava o
# filho via "Start-Process cmd /c ...", que herda um console; um console
# nunca produz EOF, entao o mecanismo primario documentado no plano nunca
# era exercitado.
#
# ArgumentList (nao a string .Arguments) para os mesmos motivos que levaram
# o script externo a abandonar aspas embutidas manualmente: cada token vai
# separado, e o .NET aplica seu proprio escapamento por token.
$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = $BinaryPath
$psi.ArgumentList.Add("serve")
$psi.ArgumentList.Add("--vault")
$psi.ArgumentList.Add($VaultPath)
$psi.ArgumentList.Add("--log-level")
$psi.ArgumentList.Add("debug")
$psi.UseShellExecute = $false
# No cenario parent-death NAO se redireciona stdin: o filho herda o stdin deste
# processo, que e a ponta de leitura do pipe do keeper. Ver o cabecalho de
# orphan_keeper.ps1.
$psi.RedirectStandardInput = -not $InheritStdin

# StandardOutput/Error ficam SEMPRE herdados. Nos dois cenarios quem aponta o
# stderr para o arquivo de log por ciclo e o processo mais externo que
# sobrevive ao ciclo — o harness, no stdin-eof; o keeper, no parent-death — e
# o handle desce por heranca ate o servidor.
#
# Bombear o stderr aqui seria pior: um pipe criado por ESTE processo morre com
# ele, e e depois da morte deste processo que o servidor escreve justamente a
# linha "reason=" que o gate confere.

$proc = [System.Diagnostics.Process]::Start($psi)

# O script externo faz poll deste arquivo; so escrever depois que o processo
# de fato existe evita uma corrida em que o PID e lido antes de nascer.
Set-Content -Path $PidFile -Value $proc.Id -NoNewline -Encoding ascii

# Segura a ponta de escrita do pipe pelo tempo de vida da sessao sintetica.
# Este host so termina quando o script externo o mata com taskkill /F; nesse
# momento o SO fecha todos os handles dele, inclusive a ponta de escrita do
# pipe de stdin do filho — e isso, nao qualquer acao explicita deste script,
# e o que produz o EOF do lado do filho.
while ($true) {
    Start-Sleep -Milliseconds 500
}
