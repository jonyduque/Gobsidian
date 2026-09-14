#Requires -Version 7.0
<#
.SYNOPSIS
    Prova que nenhum teste redireciona caminho de maquina por variavel de
    ambiente.

.DESCRIPTION
    Em 2026-09-09 dois testes escreveram fora do proprio t.TempDir(), pela
    MESMA causa: acharam que apontar uma variavel de ambiente para um diretorio
    temporario bastava para conter a producao.

      - Os testes do log usavam t.Setenv("XDG_CACHE_HOME", ...) para desviar
        os.UserCacheDir(). Funciona no Linux. os.UserCacheDir honra
        LOCALAPPDATA no Windows, XDG_CACHE_HOME no Linux e IGNORA AS DUAS no
        macOS -- la ela devolve ~/Library/Caches sem consultar env nenhuma. O
        teste passava verde nas tres plataformas e escrevia no cache real numa.

      - Um teste do instalador chegou na funcao que instala de verdade e
        reescreveu seis configs de host MCP da maquina do dono, alem do PATH.
        O sintoma foi a duracao: 25,6 s num subteste. A correcao foi injetar a
        funcao por variavel de pacote e substitui-la por um gravador (0,24 s).

    A licao das duas, escrita em ARMADILHAS.md: um teste nao pode alcancar a
    funcao que escreve fora do t.TempDir(). A metade mecanizavel dela e esta --
    variavel de ambiente NAO e isolamento. O que isola e injecao: uma variavel
    de pacote com a funcao ou a raiz, que o teste troca.

    Isto recusa t.Setenv e os.Setenv das variaveis que a stdlib consulta para
    resolver casa, cache, config e runtime do USUARIO. Variavel do proprio
    produto (GOBSIDIAN_*) e variavel de apresentacao (NO_COLOR, TERM) nao
    mudam caminho nenhum e passam.

    LIMITE CONHECIDO: so pega o nome escrito como literal na chamada.
    config_test.go passa um `k` de uma tabela; os valores de la sao todos
    GOBSIDIAN_*, e um gate que tentasse seguir a variavel estaria escrevendo
    um verificador de fluxo de dados para cobrir um caso que nao existe.

.NOTES
    Saida em ASCII puro. Sai 1 se algum teste redirecionar caminho de maquina.

    -Raiz existe para o check_gates.ps1 rodar contra copias mutadas, em vez de
    contra o repositorio.
#>
[CmdletBinding()]
param(
    [string]$Raiz,
    [switch]$Silencioso
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not $Raiz) { $Raiz = Split-Path -Parent $PSScriptRoot }

# Cada uma com a funcao da stdlib que a consulta -- a mensagem tem de dizer o
# que exatamente o teste estava desviando.
$Redirecionadoras = [ordered]@{
    'LOCALAPPDATA'    = 'os.UserCacheDir/os.UserConfigDir no Windows'
    'APPDATA'         = 'os.UserConfigDir no Windows'
    'XDG_CACHE_HOME'  = 'os.UserCacheDir no Linux (IGNORADA no macOS)'
    'XDG_CONFIG_HOME' = 'os.UserConfigDir no Linux (IGNORADA no macOS)'
    'XDG_DATA_HOME'   = 'diretorio de dados no Linux'
    'XDG_RUNTIME_DIR' = 'o diretorio de runtime do ipc'
    'HOME'            = 'os.UserHomeDir fora do Windows'
    'USERPROFILE'     = 'os.UserHomeDir no Windows'
}

$Arquivos = @(Get-ChildItem -Path $Raiz -Recurse -Filter '*_test.go' -File -ErrorAction SilentlyContinue)
$Problemas = @()

# Le o arquivo inteiro, e nao linha a linha: o argumento pode estar na linha
# seguinte a abertura do parenteses, e uma varredura por linha nao veria.
# Comentario de linha sai antes -- ARMADILHAS.md cita a chamada em prosa.
foreach ($arq in $Arquivos) {
    $texto = Get-Content -Path $arq.FullName -Raw -Encoding UTF8
    if (-not $texto) { continue }
    $codigo = [regex]::Replace($texto, '(?m)//.*$', '')
    foreach ($m in [regex]::Matches($codigo, '(?s)(?:\bt|\bos)\.Setenv\(\s*"([A-Z_]+)"')) {
        $nome = $m.Groups[1].Value
        if (-not $Redirecionadoras.Contains($nome)) { continue }
        $n = ($codigo.Substring(0, $m.Index) -split "`n").Count
        $rel = $arq.FullName.Substring($Raiz.Length).TrimStart('\', '/')
        $Problemas += "${rel}:${n}: Setenv de $nome desvia $($Redirecionadoras[$nome])"
    }
}

# Segunda regra: o desvio do diretorio de runtime so pode ser armado por teste.
#
# ipc.RodarComRuntimeIsolado troca o diretorio de runtime de TODO o processo por
# um temporario. Existe para o TestMain dos pacotes que abrem socket e trava
# (medido em 2026-09-14: a suite deixava 15 travas, `instalacao.lock` e uma
# presenca no %LOCALAPPDATA% do usuario por rodada). Chamado em codigo de
# producao, poria a ponte e o daemon em diretorios temporarios diferentes --
# dois processos servindo um cofre sem se enxergar, o incidente de 2026-09-08.
$ProblemasDesvio = @()
$Producao = @(Get-ChildItem -Path $Raiz -Recurse -Filter '*.go' -File -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -notlike '*_test.go' })
foreach ($arq in $Producao) {
    $texto = Get-Content -Path $arq.FullName -Raw -Encoding UTF8
    if (-not $texto -or $texto -notmatch 'RodarComRuntimeIsolado') { continue }
    $codigo = [regex]::Replace($texto, '(?m)//.*$', '')
    # A definicao nao e chamada.
    $codigo = [regex]::Replace($codigo, 'func\s+RodarComRuntimeIsolado\s*\(', 'func _(')
    foreach ($m in [regex]::Matches($codigo, '\bRodarComRuntimeIsolado\s*\(')) {
        $n = ($codigo.Substring(0, $m.Index) -split "`n").Count
        $rel = $arq.FullName.Substring($Raiz.Length).TrimStart('\', '/')
        $ProblemasDesvio += "${rel}:${n}: RodarComRuntimeIsolado fora de arquivo _test.go"
    }
}

if ($Problemas.Count -gt 0) {
    Write-Output "[!] teste redirecionando caminho de maquina por variavel de ambiente:"
    $Problemas | ForEach-Object { Write-Output "     $_" }
    Write-Output ""
    Write-Output "     Variavel de ambiente nao isola: os.UserCacheDir ignora LOCALAPPDATA e"
    Write-Output "     XDG_CACHE_HOME no macOS. Injete a raiz ou a funcao por variavel de"
    Write-Output "     pacote e troque-a no teste -- ver docs/ARMADILHAS.md."
}
if ($ProblemasDesvio.Count -gt 0) {
    Write-Output "[!] desvio do diretorio de runtime armado fora de teste:"
    $ProblemasDesvio | ForEach-Object { Write-Output "     $_" }
    Write-Output ""
    Write-Output "     ipc.RodarComRuntimeIsolado e so para TestMain. Em producao ele poria"
    Write-Output "     ponte e daemon em diretorios de runtime diferentes."
}
if ($Problemas.Count -gt 0 -or $ProblemasDesvio.Count -gt 0) { exit 1 }

if (-not $Silencioso) {
    Write-Output "[OK] nenhum dos $($Arquivos.Count) arquivos _test.go redireciona caminho de maquina, e o desvio de runtime so e armado por teste"
}
exit 0
