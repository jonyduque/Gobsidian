#Requires -Version 7.0
<#
.SYNOPSIS
    Mede indexacao a frio (RNF-01) e heap vivo em repouso (RNF-07) contra um cofre.

.DESCRIPTION
    Existe para que os numeros de docs/OPERACAO.md sejam medidos, e nao
    estimados. Uma versao anterior daquela tabela trazia "ex: 408ms em teste
    local" e "tende a ficar ~30-45 MB" — um exemplo ilustrativo e uma
    expectativa, apresentados como resultado. Este script produz o numero real
    ou nao produz nada.

    index_ms  vem do proprio servidor, do log "servidor pronto". Recorta so a
              construcao do indice — nao inclui boot do runtime do Go, leitura
              de config nem handshake do MCP. E o que RNF-01 nomeia.

    heap vivo e o que RNF-07 nomeia desde 2026-08-28. Vem do gctrace: na linha
              "gc N ... A->B->C MB, D MB goal", C e o heap vivo ao fim do ciclo.
              Nao exige nada do produto — GODEBUG e do runtime do Go.

    # Por que DUAS partidas

    O RNF-07 nomeia dois estados, e este script mede os dois:

      pronto    boot completo, ANTES da primeira busca
      servindo  depois de uma vault_search, que e o que dispara a carga
                preguicosa do indice invertido

    Sao duas PARTIDAS do servidor, e nao dois momentos da mesma, porque separar
    as linhas de gctrace de antes e depois da busca por timestamp e fragil: o
    relogio do gctrace conta do inicio do processo e o do script nao, e um erro
    de alguns milissegundos atribuiria o ciclo errado ao estado errado. Duas
    partidas nao tem ambiguidade nenhuma.

    Ate 2026-08-28 este script media SO o estado "pronto" — e o publicava como
    "RSS em repouso". Desde a carga preguicosa, uma sessao que nunca buscou
    nunca carregou o indice invertido, entao o numero publicado descrevia um
    servidor que ainda nao tinha buscado. Nenhuma sessao real esta nesse estado.

    # Por que RSS saiu do requisito e ficou na saida

    RSS acompanha a META de heap do GC (~2x o heap vivo ao fim de cada ciclo),
    nao o volume de dados. Medido em 2026-08-27: um binario SEM um campo
    consumia 3,6 MB A MAIS de RSS que o binario COM ele, reprodutivelmente e com
    faixas disjuntas — a diferenca era qual ciclo de GC tinha terminado por
    ultimo. Instrumento que inverte de sinal nao serve de orcamento. Continua
    reportado como diagnostico, nunca como veredito.

    O processo e encerrado fechando o stdin, que e o mesmo caminho que um host
    MCP usa. Matar o processo mediria o desligamento errado.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Vault,

    [string]$Binary = (Join-Path $PSScriptRoot ".." "bin" "gobsidian.exe"),

    # Diretorio de cache. Vale a pena apontar para um proprio quando se mede um
    # cofre que o dono usa em sessao viva: um binario de outra versao de formato
    # grava um cache que o instalado recusa, e todas as sessoes reconstroem.
    [string]$CacheDir = "",

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

# UmaPartida sobe o servidor, roda o protocolo e devolve o que a partida mediu.
# As duas medicoes do RNF-07 chamam esta funcao com $ComBusca diferente, e e a
# UNICA que fala com o servidor — dois corpos parecidos divergiriam, e a
# divergencia apareceria no estado menos medido.
function UmaPartida {
    param([bool]$ComBusca)

    $psi = [System.Diagnostics.ProcessStartInfo]::new()
    $psi.FileName = $Binary
    $psi.ArgumentList.Add("serve")
    $psi.ArgumentList.Add("--vault")
    $psi.ArgumentList.Add($Vault)
    if ($CacheDir -ne "") {
        $psi.ArgumentList.Add("--cache-dir")
        $psi.ArgumentList.Add($CacheDir)
    }
    $psi.RedirectStandardInput = $true
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.UseShellExecute = $false
    # gctrace e a fonte do heap vivo. Nao toca no produto: e do runtime do Go.
    $psi.EnvironmentVariables["GODEBUG"] = "gctrace=1"
    # EM PROCESSO, sempre.
    #
    # Sem isto o `serve` tenta o daemon, e quando ha um o processo que este
    # script mede e uma PONTE de ~15 MB — nao o processo que segura o indice.
    # O numero passava a depender de haver ou nao daemon vivo naquele instante,
    # e a ponte nem imprime "servidor pronto", entao o script falhava ou media a
    # coisa errada sem dizer qual das duas. RNF-07 nomeia o servidor que carrega
    # o indice; e ele que tem de ser medido.
    $psi.EnvironmentVariables["GOBSIDIAN_NO_DAEMON"] = "1"

    $proc = [System.Diagnostics.Process]::Start($psi)
    if (-not $proc) { throw "[!] Process.Start nao devolveu um processo" }

    try {
        # O stderr inteiro e drenado em segundo plano. Ler linha a linha ate o
        # "servidor pronto" e parar — que era o que este script fazia — enche o
        # cano com as linhas de gc que vem DEPOIS, e o servidor trava na
        # escrita. As linhas de gc que interessam sao justamente essas.
        $errTask = $proc.StandardError.ReadToEndAsync()

        # O handshake so responde depois de o servidor estar pronto, entao a
        # resposta do initialize e o sinal de prontidao. O "servidor pronto" do
        # log e lido depois, do stderr ja capturado.
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
        $null = $proc.StandardOutput.ReadLine()

        $Hits = -1
        if ($ComBusca) {
            # A consulta importa menos que o fato de haver uma: qualquer
            # vault_search dispara a carga do indice invertido, que e o que
            # separa "pronto" de "servindo". Os hits sao reportados para que
            # uma consulta que nao casou nada seja visivel, e nao suposta.
            $Busca = '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"vault_search","arguments":{"query":"a","limit":20}}}'
            $proc.StandardInput.WriteLine($Busca)
            $proc.StandardInput.Flush()
            $Resp = $proc.StandardOutput.ReadLine()
            if ($Resp -match '"total":(\d+)') { $Hits = [int]$Matches[1] }
        }

        Start-Sleep -Milliseconds $SettleMs

        # Maior amostra, nao a ultima: um pico que cabe no orcamento e
        # informacao; um pico que nao cabe, mascarado por uma amostra tardia, e
        # ficcao.
        $PeakBytes = 0
        for ($i = 0; $i -lt $Samples; $i++) {
            $proc.Refresh()
            if ($proc.HasExited) { throw "[!] servidor morreu durante a medicao - nada foi medido" }
            if ($proc.WorkingSet64 -gt $PeakBytes) { $PeakBytes = $proc.WorkingSet64 }
            Start-Sleep -Milliseconds 200
        }
    }
    finally {
        # Fechar o stdin e o caminho que um host MCP usa. O servidor tem tres
        # mecanismos de encerramento e este exercita o primeiro deles.
        if (-not $proc.HasExited) {
            $proc.StandardInput.Close()
            if (-not $proc.WaitForExit($ReadyTimeoutMs)) {
                Write-Warning "[!] servidor nao encerrou apos EOF de stdin - matando"
                $proc.Kill()
            }
        }
    }

    $Err = $errTask.Result
    $proc.Dispose()

    $Ready = ($Err -split "`n") | Where-Object { $_ -match 'servidor pronto' } | Select-Object -First 1
    if (-not $Ready) { throw "[!] servidor nunca imprimiu 'servidor pronto' - nada foi medido" }

    # slog.TextHandler emite chave=valor. Nao inventar numero: se o campo nao
    # estiver la, dizer que nao estava.
    $IndexMs = if ($Ready -match 'index_ms=(\d+)') { [int]$Matches[1] } else { $null }
    $Notes   = if ($Ready -match 'notes=(\d+)') { [int]$Matches[1] } else { $null }
    $Assets  = if ($Ready -match 'assets=(\d+)') { [int]$Matches[1] } else { $null }
    $Origin  = if ($Ready -match 'index_origin=(\w+)') { $Matches[1] } else { "?" }

    # ULTIMA linha de gc da partida. O terceiro numero de "A->B->C MB" e o heap
    # vivo ao fim daquele ciclo; o "D MB goal" e a meta que o RSS acompanha.
    $HeapVivo = $null; $Meta = $null; $Ciclos = 0
    foreach ($l in ($Err -split "`n")) {
        if ($l -match '^gc (\d+) .*?(\d+)->(\d+)->(\d+) MB, (\d+) MB goal') {
            $Ciclos = [int]$Matches[1]; $HeapVivo = [int]$Matches[4]; $Meta = [int]$Matches[5]
        }
    }

    [pscustomobject]@{
        IndexMs = $IndexMs; Notes = $Notes; Assets = $Assets; Origin = $Origin
        HeapVivoMB = $HeapVivo; MetaMB = $Meta; Ciclos = $Ciclos
        RssMB = [math]::Round($PeakBytes / 1MB, 1); Hits = $Hits
    }
}

$Pronto   = UmaPartida -ComBusca $false
$Servindo = UmaPartida -ComBusca $true

# O teto do RNF-07 escala com o cofre. Teto absoluto num produto que indexa de
# 78 a 5.686 notas mede o tamanho do cofre, nao a qualidade do servidor — ver
# docs/PRD.md secao 6.1.
$BaseMB = 8
$KBPorNota = 32
$TetoMB = if ($null -ne $Servindo.Notes) {
    [math]::Round($BaseMB + ($KBPorNota * $Servindo.Notes / 1024), 1)
} else { $null }

Write-Output ""
Write-Output "=== medicao ==="
Write-Output ("    cofre           : {0}" -f $Vault)
Write-Output ("    notas / anexos  : {0} / {1}" -f $Servindo.Notes, $Servindo.Assets)
Write-Output ("    origem do indice: {0}" -f $Servindo.Origin)
if ($null -ne $Pronto.IndexMs) {
    Write-Output ("    RNF-01 indice   : {0} ms (alvo <= 3000)" -f $Pronto.IndexMs)
} else {
    Write-Output "    RNF-01 indice   : NAO MEDIDO (campo index_ms ausente do log)"
}
if ($null -ne $Pronto.HeapVivoMB -and $null -ne $Servindo.HeapVivoMB) {
    Write-Output ("    RNF-07 heap vivo: pronto {0} MB / servindo {1} MB (alvo <= {2} MB)" -f `
        $Pronto.HeapVivoMB, $Servindo.HeapVivoMB, $TetoMB)
} else {
    Write-Output "    RNF-07 heap vivo: NAO MEDIDO (nenhuma linha de gctrace na saida)"
}
Write-Output ("    RSS (diagnostico): pronto {0} MB / servindo {1} MB - NAO e o requisito" -f `
    $Pronto.RssMB, $Servindo.RssMB)
Write-Output ("    meta de heap     : pronto {0} MB / servindo {1} MB - e o que o RSS segue" -f `
    $Pronto.MetaMB, $Servindo.MetaMB)
Write-Output ("    ciclos de GC     : pronto {0} / servindo {1}" -f $Pronto.Ciclos, $Servindo.Ciclos)
Write-Output ("    busca            : {0} resultados" -f $Servindo.Hits)
Write-Output ("    maquina         : {0} / {1} nucleos" -f $env:COMPUTERNAME, [Environment]::ProcessorCount)
Write-Output ("    binario         : {0}" -f $Binary)

$Fail = $false
if ($null -ne $Pronto.IndexMs -and $Pronto.IndexMs -gt 3000) {
    Write-Warning "[!] RNF-01 estourado: $($Pronto.IndexMs)ms > 3000ms"
    $Fail = $true
}
if ($null -ne $Servindo.HeapVivoMB -and $null -ne $TetoMB -and $Servindo.HeapVivoMB -gt $TetoMB) {
    Write-Warning "[!] RNF-07 estourado no estado servindo: $($Servindo.HeapVivoMB)MB > ${TetoMB}MB"
    $Fail = $true
}
if ($Servindo.Origin -ne "cache") {
    # Um numero medido sobre indice RECONSTRUIDO nao e repouso com cache valido.
    # Isso ja contaminou uma medicao publicada — 57,1 MB onde o valor com cache
    # era 49,5 — porque a origem nao era conferida.
    Write-Warning "[!] index_origin=$($Servindo.Origin), nao 'cache': rode de novo para medir com cache quente"
}
if ($Fail) {
    Write-Output "[!] Alvo estourado - registre o numero assim mesmo em docs/OPERACAO.md"
} else {
    Write-Output "[OK] Dentro dos alvos"
}
