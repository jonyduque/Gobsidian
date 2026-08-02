#Requires -Version 7.0
<#
.SYNOPSIS
    Compara a saida de `go test -bench` com uma referencia versionada e reprova
    regressao acima da tolerancia.

.DESCRIPTION
    A referencia e um ARQUIVO NO REPOSITORIO, nao a execucao anterior. Comparar
    com "a ultima rodada" torna degradacao lenta invisivel: 5% por semana nunca
    dispara um gate de 20%, e em dez semanas o numero dobrou. Atualizar a
    referencia e commit deliberado, e o diff mostra a piora que alguem aceitou.

    A comparacao e POR BENCHMARK. Um agregado esconde a regressao de um caminho
    atras da melhora de outro.

    MELHORA acima da tolerancia avisa e NAO reprova. Melhora grande costuma
    significar que o benchmark parou de medir o que media -- este projeto ja
    registrou uma medicao de RNF-04 de 0,58 ms porque chamava a camada errada.
    O aviso existe para alguem olhar; reprovar por ficar mais rapido seria
    hostil.

    Benchmark que esta na referencia e NAO aparece na saida REPROVA. Benchmark
    que sumiu, foi renomeado ou pulou (cofre ausente) nao mede nada, e tratar
    ausencia como sucesso e o modo mais barato de ter um gate que nunca falha.

.PARAMETER BenchOutput
    Arquivo com a saida bruta de `go test -bench`.

.PARAMETER Baseline
    JSON de referencia. Padrao: docs/bench-baseline.json.

.PARAMETER TolerancePct
    Piora percentual que reprova. Padrao: 20.

.PARAMETER UpdateBaseline
    Reescreve o arquivo de referencia com os numeros medidos, em vez de
    comparar. Usar so em commit deliberado.

.PARAMETER Runner
    Texto que identifica a maquina, gravado junto com os numeros. Obrigatorio
    com -UpdateBaseline: numero sem maquina nomeada nao e comparavel.

.EXAMPLE
    go test -run '^$' -bench . ./internal/service/ > bench.txt
    .\scripts\bench_compare.ps1 -BenchOutput bench.txt
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$BenchOutput,

    [string]$Baseline = "docs/bench-baseline.json",

    [int]$TolerancePct = 20,

    [switch]$UpdateBaseline,

    [string]$Runner = "",

    [string]$Commit = "",

    # Texto livre gravado na referencia. Existe para dizer COMO os numeros
    # foram obtidos -- uma amostra so, ou a mediana de varias. Numa maquina
    # compartilhada a diferenca decide se o gate pisca: a primeira referencia
    # deste projeto veio de uma rodada unica em que a busca por frase saiu
    # rapida, e as duas rodadas limpas seguintes ficaram a menos de um ponto
    # percentual de reprovar sem mudanca nenhuma no codigo.
    [string]$Nota = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $ProjectRoot
try {
    if (-not (Test-Path $BenchOutput)) {
        Write-Warning "[!] saida de benchmark nao encontrada: $BenchOutput"
        exit 2
    }

    # Linha do go test: "BenchmarkNome-12   	     100	  12345678 ns/op".
    # O sufixo -N e GOMAXPROCS e nao faz parte do nome do benchmark.
    $Medidos = [ordered]@{}
    foreach ($linha in Get-Content -Path $BenchOutput) {
        if ($linha -match '^(Benchmark[A-Za-z0-9_]+)(-\d+)?\s+\d+\s+([\d.]+)\s+ns/op') {
            $Medidos[$Matches[1]] = [double]$Matches[3]
        }
    }

    if ($Medidos.Count -eq 0) {
        Write-Warning "[!] nenhuma linha 'ns/op' em $BenchOutput."
        Write-Output "    Provavel causa: todos os benchmarks pularam por falta do cofre"
        Write-Output "    sintetico. Gere com scripts/gen_vault.ps1 e aponte"
        Write-Output "    GOBSIDIAN_BENCH_VAULT para ele."
        exit 1
    }

    Write-Output "[i] $($Medidos.Count) benchmark(s) medido(s) em $BenchOutput."

    if ($UpdateBaseline) {
        if (-not $Runner) {
            Write-Warning "[!] -UpdateBaseline exige -Runner: numero sem maquina nomeada nao e comparavel."
            exit 2
        }
        $Novo = [ordered]@{
            runner         = $Runner
            commit         = $Commit
            nota           = $Nota
            tolerancia_pct = $TolerancePct
            unidade        = "ns/op"
            benchmarks     = $Medidos
        }
        $Novo | ConvertTo-Json -Depth 5 | Set-Content -Path $Baseline -Encoding utf8
        Write-Output "[OK] referencia gravada em $Baseline ($Runner)."
        foreach ($k in $Medidos.Keys) {
            Write-Output ("    {0,-32} {1,15:N0} ns/op" -f $k, $Medidos[$k])
        }
        Write-Output ""
        Write-Output "[i] Isto NAO substitui a medicao local de docs/OPERACAO.md."
        Write-Output "    Runner compartilhado tem variancia alta; este gate guarda contra"
        Write-Output "    regressao de ordem de grandeza, nao contra alguns por cento."
        exit 0
    }

    if (-not (Test-Path $Baseline)) {
        Write-Warning "[!] referencia ausente: $Baseline"
        Write-Output ""
        Write-Output "    Os numeros medidos nesta maquina foram:"
        foreach ($k in $Medidos.Keys) {
            Write-Output ("    {0,-32} {1,15:N0} ns/op" -f $k, $Medidos[$k])
        }
        Write-Output ""
        Write-Output "    Para adotar, rode o mesmo comando com:"
        Write-Output "      -UpdateBaseline -Runner '<maquina>' -Commit '<sha>'"
        Write-Output "    e COMMITE o arquivo. A referencia e deliberada de proposito."
        exit 1
    }

    $Ref = Get-Content -Path $Baseline -Raw | ConvertFrom-Json
    $Tol = $TolerancePct
    if ($Ref.PSObject.Properties.Name -contains "tolerancia_pct") {
        $Tol = [int]$Ref.tolerancia_pct
    }

    Write-Output "[i] referencia: $Baseline (runner: $($Ref.runner)), tolerancia +$Tol%."
    Write-Output ""
    Write-Output ("    {0,-32} {1,13} {2,13} {3,9}" -f "benchmark", "referencia", "medido", "delta")

    $Reprovas = @()
    $Avisos = @()

    foreach ($nome in $Ref.benchmarks.PSObject.Properties.Name) {
        $base = [double]$Ref.benchmarks.$nome

        if (-not $Medidos.Contains($nome)) {
            $Reprovas += "$nome esta na referencia e NAO foi medido"
            Write-Output ("    {0,-32} {1,13:N0} {2,13} {3,9}" -f $nome, $base, "AUSENTE", "-")
            continue
        }

        $medido = $Medidos[$nome]
        $delta = if ($base -gt 0) { (($medido - $base) / $base) * 100 } else { 0 }
        Write-Output ("    {0,-32} {1,13:N0} {2,13:N0} {3,8:+0.0;-0.0;0.0}%" -f $nome, $base, $medido, $delta)

        if ($delta -gt $Tol) {
            $Reprovas += ("{0}: {1:N0} -> {2:N0} ns/op ({3:+0.0}%, acima da tolerancia de {4}%)" -f `
                    $nome, $base, $medido, $delta, $Tol)
        }
        elseif ($delta -lt (-1 * $Tol)) {
            $Avisos += ("{0}: {1:N0} -> {2:N0} ns/op ({3:0.0}%)" -f $nome, $base, $medido, $delta)
        }
    }

    foreach ($nome in $Medidos.Keys) {
        if ($Ref.benchmarks.PSObject.Properties.Name -notcontains $nome) {
            $Avisos += "$nome foi medido e nao esta na referencia; acrescente-o com -UpdateBaseline"
        }
    }

    Write-Output ""

    if ($Avisos.Count -gt 0) {
        Write-Warning "[!] $($Avisos.Count) aviso(s):"
        foreach ($a in $Avisos) { Write-Output "    $a" }
        Write-Output ""
        Write-Output "    Melhora acima da tolerancia nao reprova, mas merece um olhar:"
        Write-Output "    benchmark que fica muito mais rapido de repente costuma ter"
        Write-Output "    parado de medir o que media."
        Write-Output ""
    }

    if ($Reprovas.Count -gt 0) {
        Write-Warning "[!] $($Reprovas.Count) regressao(oes) acima de $Tol%:"
        foreach ($r in $Reprovas) { Write-Output "    $r" }
        exit 1
    }

    Write-Output "[OK] nenhum benchmark regrediu acima de $Tol%."
    exit 0
}
finally {
    Pop-Location
}
