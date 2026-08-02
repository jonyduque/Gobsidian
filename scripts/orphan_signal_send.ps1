param(
    [Parameter(Mandatory = $true)][int]$TargetPid
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Envia CTRL_BREAK ao grupo de processos do servidor.
#
# Roda como processo SEPARADO de proposito. GenerateConsoleCtrlEvent so alcanca
# processos do console a que quem chama esta ligado, e a sequencia exige
# FreeConsole/AttachConsole — mexer no console do harness no meio de uma rodada
# de 100 ciclos quebraria a saida dele. Aqui o console descartado e o deste
# processo, que morre em seguida.
#
# SetConsoleCtrlHandler(NULL, TRUE) antes do envio: o evento vai para o console
# inteiro, e sem isso ESTE processo tambem o receberia e morreria antes de
# devolver o codigo de saida que o harness le.

Add-Type -Namespace Gobsidian -Name Ctrl -MemberDefinition @'
[DllImport("kernel32.dll", SetLastError = true)]
public static extern uint GetConsoleProcessList(uint[] lpdwProcessList, uint dwProcessCount);

[DllImport("kernel32.dll", SetLastError = true)]
public static extern bool SetConsoleCtrlHandler(IntPtr handler, bool add);

[DllImport("kernel32.dll", SetLastError = true)]
public static extern bool GenerateConsoleCtrlEvent(uint dwCtrlEvent, uint dwProcessGroupId);
'@

$CTRL_BREAK_EVENT = 1

# Nao ha FreeConsole/AttachConsole aqui. Harness, host e servidor compartilham o
# MESMO console, entao este processo — filho do harness — ja esta ligado a ele.
# A primeira versao chamava FreeConsole seguido de AttachConsole($TargetPid) e
# recebia ERROR_INVALID_PARAMETER (87), que e o que AttachConsole devolve quando
# quem chama JA tem console.
$buf = New-Object uint32[] 4
if ([Gobsidian.Ctrl]::GetConsoleProcessList($buf, 4) -eq 0) {
    # Sem console nao ha a quem entregar o evento. Sessao headless de verdade
    # (um agendador, um servico) cai aqui, e o cenario nao pode rodar.
    exit 90
}

# O evento vai para o grupo do alvo. Sem isto, ESTE processo tambem o receberia
# e morreria antes de devolver o codigo de saida que o harness le.
[void][Gobsidian.Ctrl]::SetConsoleCtrlHandler([IntPtr]::Zero, $true)

# dwProcessGroupId = PID do servidor. Ele foi criado com
# CREATE_NEW_PROCESS_GROUP, entao o grupo dele e so ele: o host que o criou NAO
# recebe este evento, e continua vivo. E isso que mantem a vigilia do pai
# desligada durante o cenario.
$ok = [Gobsidian.Ctrl]::GenerateConsoleCtrlEvent([uint32]$CTRL_BREAK_EVENT, [uint32]$TargetPid)
$err = [System.Runtime.InteropServices.Marshal]::GetLastWin32Error()

if (-not $ok) { exit (200 + $err) }
exit 0
