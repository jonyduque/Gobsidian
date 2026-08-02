param(
    [Parameter(Mandatory = $true)][string]$BinaryPath,
    [Parameter(Mandatory = $true)][string]$VaultPath,
    [Parameter(Mandatory = $true)][string]$PidFile
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Host do cenario SIGNAL. Ele existe porque os outros dois nao servem:
#
#   stdin-eof     mata o host, o pipe fecha, e o EOF vence a corrida.
#   parent-death  mata o host, e a vigilia do pai vence.
#
# Aqui NADA morre e NADA fecha: o host fica vivo (entao a vigilia do pai nao
# dispara) e o stdin do servidor e o console herdado (que nunca produz EOF).
# So sobra o sinal. Teste de fallback que deixa o caminho principal ligado mede
# o caminho principal — os dois caminhos principais ficam desconectados aqui.
#
# O servidor precisa estar no PROPRIO grupo de processos. Sem
# CREATE_NEW_PROCESS_GROUP, GenerateConsoleCtrlEvent entrega o CTRL_BREAK ao
# grupo inteiro do console, o que mataria este host junto — e a morte do host
# fecharia handles e devolveria a corrida aos outros dois mecanismos, que e
# exatamente o que o cenario existe para evitar.
#
# System.Diagnostics.ProcessStartInfo NAO expoe dwCreationFlags, entao a
# criacao vai por CreateProcessW. bInheritHandles = TRUE e STARTUPINFO SEM
# STARTF_USESTDHANDLES: o servidor herda stdin, stdout e stderr DESTE processo
# — o stderr que o harness apontou para o log do ciclo, e onde a linha
# "reason=" precisa aparecer.

Add-Type -Namespace Gobsidian -Name Win32 -MemberDefinition @'
[StructLayout(LayoutKind.Sequential)]
public struct PROCESS_INFORMATION {
    public IntPtr hProcess; public IntPtr hThread;
    public uint dwProcessId; public uint dwThreadId;
}

[StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
public struct STARTUPINFO {
    public int cb;
    public string lpReserved, lpDesktop, lpTitle;
    public int dwX, dwY, dwXSize, dwYSize;
    public int dwXCountChars, dwYCountChars, dwFillAttribute, dwFlags;
    public short wShowWindow, cbReserved2;
    public IntPtr lpReserved2, hStdInput, hStdOutput, hStdError;
}

// lpApplicationName e lpCurrentDirectory sao IntPtr, nao string. Passar $null
// para um parametro `string` faz o marshalling entregar uma string VAZIA, e
// CreateProcessW rejeita diretorio corrente vazio com ERROR_INVALID_NAME (123)
// -- que foi exatamente como esta funcao falhou na primeira tentativa.
//
// lpCommandLine e StringBuilder porque CreateProcessW ESCREVE nesse buffer.
[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
public static extern bool CreateProcessW(
    IntPtr lpApplicationName, System.Text.StringBuilder lpCommandLine,
    IntPtr lpProcessAttributes, IntPtr lpThreadAttributes,
    bool bInheritHandles, uint dwCreationFlags,
    IntPtr lpEnvironment, IntPtr lpCurrentDirectory,
    ref STARTUPINFO lpStartupInfo, out PROCESS_INFORMATION lpProcessInformation);

[DllImport("kernel32.dll", SetLastError = true)]
public static extern bool CloseHandle(IntPtr hObject);

[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
public static extern IntPtr CreateFileW(
    string lpFileName, uint dwDesiredAccess, uint dwShareMode,
    IntPtr lpSecurityAttributes, uint dwCreationDisposition,
    uint dwFlagsAndAttributes, IntPtr hTemplateFile);

[DllImport("kernel32.dll", SetLastError = true)]
public static extern IntPtr GetStdHandle(int nStdHandle);

[DllImport("kernel32.dll", SetLastError = true)]
public static extern bool SetHandleInformation(IntPtr hObject, uint dwMask, uint dwFlags);
'@

$CREATE_NEW_PROCESS_GROUP = 0x00000200
$STARTF_USESTDHANDLES = 0x00000100
$GENERIC_READ = [uint32]2147483648   # 0x80000000
$GENERIC_WRITE = [uint32]1073741824  # 0x40000000
$FILE_SHARE_READ = 0x00000001
$FILE_SHARE_WRITE = 0x00000002
$OPEN_EXISTING = 3
$STD_OUTPUT_HANDLE = -11
$STD_ERROR_HANDLE = -12
$HANDLE_FLAG_INHERIT = 0x00000001
$INVALID_HANDLE_VALUE = [IntPtr](-1)

# O stdin do servidor precisa EXISTIR e NUNCA produzir EOF, senao o cenario
# volta a medir stdin-eof. Herdar o stdin deste processo nao serve: o harness
# roda sem stdin interativo, o servidor via EOF na hora e encerrava antes de
# qualquer sinal — foi o que a primeira tentativa mediu, e o gate de reason=
# acusou ("reason=stdin-eof, esperado 'signal'").
#
# CONIN$ e o buffer de entrada do console. Ele nao fecha, e por isso o unico
# mecanismo que sobra e o CTRL_BREAK.
$hConIn = [Gobsidian.Win32]::CreateFileW(
    "CONIN$", [uint32]($GENERIC_READ -bor $GENERIC_WRITE),
    [uint32]($FILE_SHARE_READ -bor $FILE_SHARE_WRITE), [IntPtr]::Zero,
    $OPEN_EXISTING, 0, [IntPtr]::Zero)

if ($hConIn -eq $INVALID_HANDLE_VALUE) {
    $err = [System.Runtime.InteropServices.Marshal]::GetLastWin32Error()
    Write-Error "[!] CreateFileW(CONIN$) falhou: erro $err - sessao sem console, o cenario signal nao pode rodar aqui"
    exit 2
}
[void][Gobsidian.Win32]::SetHandleInformation($hConIn, $HANDLE_FLAG_INHERIT, $HANDLE_FLAG_INHERIT)

$si = New-Object Gobsidian.Win32+STARTUPINFO
$si.cb = [System.Runtime.InteropServices.Marshal]::SizeOf($si)
# Com STARTF_USESTDHANDLES os TRES handles tem de ser fornecidos. stdout e
# stderr saem de GetStdHandle deste processo: o stderr e o arquivo de log do
# ciclo, que o harness apontou, e e nele que a linha "reason=" precisa cair.
$si.dwFlags = $STARTF_USESTDHANDLES
$si.hStdInput = $hConIn
$si.hStdOutput = [Gobsidian.Win32]::GetStdHandle($STD_OUTPUT_HANDLE)
$si.hStdError = [Gobsidian.Win32]::GetStdHandle($STD_ERROR_HANDLE)

$pi = New-Object Gobsidian.Win32+PROCESS_INFORMATION

# Aspas em volta de cada caminho: o cofre de teste pode estar sob "Program
# Files" ou qualquer diretorio com espaco, e CreateProcessW recebe UMA linha de
# comando, nao uma lista de tokens.
$cmdline = New-Object System.Text.StringBuilder(
    ('"{0}" serve --vault "{1}" --log-level debug' -f $BinaryPath, $VaultPath), 32768)

$ok = [Gobsidian.Win32]::CreateProcessW(
    [IntPtr]::Zero, $cmdline, [IntPtr]::Zero, [IntPtr]::Zero,
    $true, $CREATE_NEW_PROCESS_GROUP, [IntPtr]::Zero, [IntPtr]::Zero,
    [ref]$si, [ref]$pi)

if (-not $ok) {
    $err = [System.Runtime.InteropServices.Marshal]::GetLastWin32Error()
    Write-Error "[!] CreateProcessW falhou: erro $err"
    exit 1
}

[void][Gobsidian.Win32]::CloseHandle($pi.hThread)
[void][Gobsidian.Win32]::CloseHandle($pi.hProcess)

Set-Content -Path $PidFile -Value $pi.dwProcessId -NoNewline -Encoding ascii

# Fica vivo. Se este processo morresse, a vigilia do pai dispararia e o
# cenario mediria o mecanismo errado.
while ($true) {
    Start-Sleep -Milliseconds 500
}
