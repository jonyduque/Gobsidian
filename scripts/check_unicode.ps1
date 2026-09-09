#Requires -Version 7.0
<#
.SYNOPSIS
    Prova que nenhuma chave derivada PERSISTIDA depende de tabela Unicode que a
    toolchain move sozinha.

.DESCRIPTION
    Ha duas fontes de Unicode neste binario e so uma e fixada:

      - norm.NFC e norm.NFD vem de golang.org/x/text, modulo com versao no
        go.mod. Trocar de toolchain NAO os move.
      - strings.ToLower, unicode.IsLetter e unicode.IsDigit vem da stdlib.
        Trocar de toolchain move os tres, e ninguem e avisado.

    O Go 1.27 subiu as tabelas da geracao 15 para a 17, com a v1.6.0 ja
    publicada nessa toolchain. Duas contas passavam por la:

      config.VaultKey        nomeia o diretorio de cache E o caminho do socket
      search/analyzer.go     decide o que e termo no indice invertido persistido

    A primeira nao pode se mover NUNCA: chave que muda entre duas versoes do
    binario deixa daemon velho num caminho e cliente novo noutro, que e "dois
    processos servindo um cofre" -- o incidente de 2026-09-08 por outra porta.
    Ela passou a usar cases.Lower do x/text.

    A segunda PODE se mover, desde que o cache saiba: CacheAnalyzerVersion
    carrega text.VersaoDasTabelas() dentro do numero, entao o cache se invalida
    sozinho na proxima geracao.

    Este script cobra as duas. Ele NAO probe strings.ToLower no arquivo inteiro:
    config.go usa a funcao para ler nivel de log e valor booleano de env, e
    isso nao deriva chave nenhuma. Um gate que reprovasse aquilo seria
    desligado na primeira semana. A cobranca e sobre o CORPO das funcoes que
    calculam a chave.

.NOTES
    Saida em ASCII puro. Sai 1 se alguma das duas invariantes quebrar.

    Le o texto, nao a AST: as duas sao sobre o corpo de funcoes de topo, o
    gofmt e etapa do proprio verify.ps1 e garante a chave que fecha na coluna 0.

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

$Config = Join-Path $Raiz 'internal/config/config.go'
$Persist = Join-Path $Raiz 'internal/search/persist.go'

foreach ($f in @($Config, $Persist)) {
    if (-not (Test-Path $f)) {
        Write-Output "[!] arquivo ausente: $f"
        exit 1
    }
}

# CorpoDaFuncao devolve as linhas de codigo de uma funcao de topo, sem
# comentario -- o comentario de caixaEstavel CITA strings.ToLower para explicar
# por que ela nao o usa, e um gate que lesse comentario reprovaria a propria
# documentacao do conserto.
function CorpoDaFuncao {
    param([string]$Caminho, [string]$Nome)
    $linhas = Get-Content -Path $Caminho -Encoding UTF8
    $dentro = $false
    $saida = @()
    foreach ($l in $linhas) {
        if (-not $dentro) {
            if ($l -match ("^func\s+" + [regex]::Escape($Nome) + "\s*\(")) { $dentro = $true }
            continue
        }
        if ($l -match '^\}') { break }
        $limpa = ($l -replace '//.*$', '')
        if ($limpa.Trim()) { $saida += $limpa }
    }
    return $saida
}

# As chamadas que consultam tabela Unicode da stdlib.
$Moveis = 'strings\.(ToLower|ToUpper|ToTitle|EqualFold)\(|unicode\.(To|Is)[A-Za-z]*\('

$Problemas = @()

# ----------------------------------------------------------------------
# 1. A chave do cofre nao pode se mover
# ----------------------------------------------------------------------
$AchouAlguma = $false
foreach ($fn in @('VaultKey', 'caixaEstavel')) {
    # @() porque o PowerShell desenrola array de um elemento no return, e ai
    # .Count nao existe -- com StrictMode isso e erro, nao zero.
    $corpo = @(CorpoDaFuncao $Config $fn)
    if ($corpo.Count -eq 0) { continue }
    $AchouAlguma = $true
    foreach ($l in $corpo) {
        if ($l -match $Moveis) {
            $Problemas += "config.$fn usa tabela Unicode da stdlib: $($l.Trim())"
            $Problemas += "    VaultKey nomeia o cache E o socket; ela nao pode mudar com a toolchain."
        }
    }
}
if (-not $AchouAlguma) {
    $Problemas += "nao achei VaultKey nem caixaEstavel em internal/config/config.go -- a conta da chave mudou de lugar?"
}

# ----------------------------------------------------------------------
# 2. A versao do analisador tem de carregar a geracao das tabelas
# ----------------------------------------------------------------------
$TextoPersist = (Get-Content -Path $Persist -Encoding UTF8 | Where-Object { $_ -notmatch '^\s*//' }) -join "`n"
$m = [regex]::Match($TextoPersist, '(?m)^var\s+CacheAnalyzerVersion\s*=\s*(?<expr>.+)$')
if (-not $m.Success) {
    $Problemas += "CacheAnalyzerVersion nao e mais um var com expressao em internal/search/persist.go"
    $Problemas += "    ela precisa ser calculada: uma constante nao acompanha a geracao Unicode da toolchain."
}
elseif ($m.Groups['expr'].Value -notmatch 'VersaoDasTabelas\(\)') {
    $Problemas += "CacheAnalyzerVersion nao carrega text.VersaoDasTabelas(): $($m.Groups['expr'].Value.Trim())"
    $Problemas += "    sem isso, cache gravado com uma geracao Unicode e lido como valido por outra."
}

if ($Problemas.Count -gt 0) {
    Write-Output "[!] chave derivada dependendo de tabela que a toolchain move:"
    $Problemas | ForEach-Object { Write-Output "     $_" }
    Write-Output ""
    Write-Output "     Ver docs/ARMADILHAS.md, 'A tabela Unicode da stdlib se move quando a"
    Write-Output "     toolchain se move' -- e internal/text/tabelas.go para o mecanismo."
    exit 1
}

if (-not $Silencioso) {
    Write-Output "[OK] a chave do cofre nao usa tabela movel, e a versao do analisador carrega a geracao Unicode"
}
exit 0
