#Requires -Version 7.0
<#
.SYNOPSIS
    Acha token entre crases em docs/*.md e README.md que parece identificador
    de codigo e nao aparece em nenhum .go do repositorio.

.DESCRIPTION
    A doc as vezes descreve uma decisao ("vamos persistir index_cache.gob")
    e o codigo nunca chega a implementa-la. `docs/PRD.md` Q3 fechou persistir
    DOIS caches -- so o segundo existe:

        $ grep -rn 'index_cache' --include=*.go .
        (vazio)
        $ ls "$LOCALAPPDATA/gobsidian/560a6a08c9fa8602/"
        inverted_cache.gob

    A decisao continua escrita como fechada em 2026-07-29 e nada aponta pra
    ela ter ficado obsoleta. Este script nao teria evitado a divergencia --
    evita ela ficar invisivel.

    Tres formas de token contam como "parece identificador de codigo":
      - nome de arquivo terminado em .go ou .gob (ex.: `index_cache.gob`)
      - token snake_case (ex.: `create_dirs`, `include_broken`)
      - identificador CamelCase seguido de parenteses (ex.: `MoveNote()`)

    Duas checagens, conforme a classe. `.go` e arquivo-fonte: existe no
    repositorio, ou nao existe -- confere presenca em disco pelo nome-base.
    Os outros tres (`.gob`, snake_case, CamelCase()) nao sao arquivo-fonte, e
    o teste e se o token aparece como substring em algum .go do repositorio
    -- comentario, string literal ou codigo, tanto faz. Nao aparecer em .go
    NENHUM e o sinal: a doc cita um artefato que o codigo nao tem.

    Blocos de codigo cercados por ``` ``` ficam de fora da varredura. Um bloco
    bash ou json de exemplo nao usa crase simples por dentro, e tratar o
    delimitador da cerca como token de crase gera ruido sem achado nenhum.

    NAO E VEREDITO. E uma lista para conferir, no mesmo espirito de
    audit_reports.ps1: token real que a doc cita e o codigo ainda nao tem
    (achado de verdade), mas tambem nome de flag, de comando externo, de
    variavel de ambiente ou de arquivo de outro projeto (ruido esperado) --
    cada linha precisa de uma pessoa decidindo qual e qual.

    CODIGO DE SAIDA:
      0 -> nenhum achado
      1 -> ha achados; a lista esta na saida
      2 -> o script nao conseguiu rodar

.EXAMPLE
    pwsh -File scripts/check_doc_refs.ps1
#>
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $ProjectRoot
try {
    # ---------------------------------------------------------------------
    # Corpus: texto bruto de todo .go do repositorio, producao e teste. Teste
    # entra de proposito -- `inverted_cache.gob` so aparece hoje em
    # persist_test.go, e um corpus so-producao acusaria um artefato que o
    # codigo de fato cria.
    # ---------------------------------------------------------------------
    $FontesGo = @(Get-ChildItem -Path $ProjectRoot -Filter '*.go' -Recurse -File)
    if ($FontesGo.Count -eq 0) {
        Write-Warning "[!] nenhum .go encontrado em $ProjectRoot"
        exit 2
    }
    $Corpo = ($FontesGo | ForEach-Object { Get-Content -Path $_.FullName -Raw }) -join "`n"

    # Nomes-base de todo .go do repositorio, pra checagem de ARQUIVO da
    # extensao .go (ver comentario abaixo do porque .go e .gob nao usam a
    # mesma checagem).
    $NomesGo = [System.Collections.Generic.HashSet[string]]::new(
        [string[]]($FontesGo | ForEach-Object { $_.Name }), [System.StringComparer]::Ordinal)

    Write-Output "[i] corpus: $($FontesGo.Count) arquivos .go."

    # ---------------------------------------------------------------------
    # Docs alvo: docs/*.md (nao recursivo -- docs/superpowers/plans/ fica de
    # fora, e a doc de plano nao e a que Task 84 vai revisar) e README.md.
    # ---------------------------------------------------------------------
    $Docs = @(Get-ChildItem -Path (Join-Path $ProjectRoot 'docs') -Filter '*.md' -File -ErrorAction SilentlyContinue)
    $Readme = Join-Path $ProjectRoot 'README.md'
    if (Test-Path $Readme) { $Docs += Get-Item $Readme }

    if ($Docs.Count -eq 0) {
        Write-Warning "[!] nenhum .md encontrado em docs/ nem README.md"
        exit 2
    }

    # Tres padroes, nesta ordem -- ARQUIVO primeiro porque um nome como
    # `index_cache.gob` tambem bateria em SNAKE se testado antes (o ponto
    # so separa o `_gob` do resto), entao a extensao decide a classe.
    #
    # Todos com (?-i) -- -match do PowerShell e insensivel a maiusculas por
    # padrao, e sem isso SNAKE_CASE casava com constante GRITANTE tipo
    # `ERROR_SHARING_VIOLATION` (API do Windows, nao decisao de doc) e
    # CAMEL_CALL casava com `close()` (inicial minuscula). As duas classes
    # tem regra de caixa que so faz sentido case-sensitive.
    $PadraoArquivo = '(?-i)^[\w][\w./-]*\.(go|gob)$'
    $PadraoSnake = '(?-i)^[a-z][a-z0-9]*(_[a-z0-9]+)+$'
    $PadraoCamelCall = '(?-i)^[A-Z][A-Za-z0-9]*\([^()]*\)$'

    # ---------------------------------------------------------------------
    # Tokens reconhecidos: nao sao artefato deste repositorio, e a doc esta
    # certa ao cita-los. Cada um carrega o MOTIVO, porque uma lista de
    # excecoes sem justificativa vira o lugar onde se esconde achado real.
    #
    # Eles continuam sendo IMPRESSOS no fim, so que separados e sem contar
    # como achado. Suprimir em silencio trocaria um problema por outro: dez
    # achados permanentes ensinam a ignorar a saida inteira -- que e como um
    # binario obsoleto passou tres cenarios do gate de orfaos sem ninguem
    # notar.
    # ---------------------------------------------------------------------
    $Reconhecidos = [ordered]@{
        'helpers.go'          = 'a doc diz que NAO existe -- e a regra do projeto contra arquivo-categoria'
        'utils.go'            = 'idem helpers.go'
        'interfaces.go'       = 'exemplo de categoria sintatica que o projeto evita, nao arquivo real'
        'snake_case.go'       = 'exemplo da convencao de nome, nao arquivo'
        'backend_inotify.go'  = 'arquivo do fsnotify v1.10.1, dependencia externa'
        'backend_windows.go'  = 'arquivo do fsnotify v1.10.1, dependencia externa'
        'total_bytes'         = 'a propria linha afirma que este campo NAO existe no retorno'
        'max_user_watches'    = 'sysctl do Linux (fs.inotify.max_user_watches), nome externo'
        'node_modules'        = 'diretorio do Node, citado para dizer que o instalador nao cria um'
    }

    $Achados = 0
    $Vistos = [System.Collections.Generic.List[string]]::new()

    function Add-Achado {
        param([string]$Arquivo, [int]$Linha, [string]$Regra, [string]$Token)
        $Rel = $Arquivo.Replace($ProjectRoot, '').TrimStart('\', '/')

        if ($Reconhecidos.Contains($Token)) {
            $script:Vistos.Add(("  {0}:{1}: ``{2}`` -- {3}" -f $Rel, $Linha, $Token, $Reconhecidos[$Token]))
            return
        }

        $script:Achados++
        Write-Output ("  {0}:{1}: [{2}] ``{3}``" -f $Rel, $Linha, $Regra, $Token)
    }

    foreach ($Doc in $Docs) {
        $Linhas = @(Get-Content -Path $Doc.FullName -Encoding utf8)
        $DentroDeCerca = $false

        for ($i = 0; $i -lt $Linhas.Count; $i++) {
            $L = $Linhas[$i]

            # Cerca de bloco de codigo. So a linha do delimitador alterna o
            # estado -- o conteudo entre cercas fica fora da varredura.
            if ($L -match '^\s*```') {
                $DentroDeCerca = -not $DentroDeCerca
                continue
            }
            if ($DentroDeCerca) { continue }

            foreach ($m in [regex]::Matches($L, '`([^`]+)`')) {
                $Token = $m.Groups[1].Value.Trim().Trim("`"'")
                if ($Token.Length -eq 0) { continue }

                $Regra = $null
                $Ausente = $false
                if ($Token -match $PadraoArquivo) {
                    $Regra = 'ARQUIVO'
                    if ($Token -match '(?-i)\.go$') {
                        # .go e arquivo-fonte: existe no repositorio ou nao
                        # existe, ponto. Buscar o NOME LITERAL dentro de texto
                        # de outro .go quase sempre da falso positivo -- Go
                        # nao se autorreferencia por nome de arquivo, entao
                        # `persist.go`, `analyzer.go` etc. nunca aparecem como
                        # substring em lugar nenhum apesar de existirem de
                        # verdade. Confere so o nome-base, sem o caminho: a
                        # doc as vezes cita so `persist.go` num diagrama, sem
                        # o diretorio.
                        $Base = [System.IO.Path]::GetFileName($Token)
                        $Ausente = -not $NomesGo.Contains($Base)
                    }
                    else {
                        # .gob e arquivo de DADO gerado em runtime, fora do
                        # repositorio (cache em %LOCALAPPDATA%) -- checagem de
                        # disco nunca acharia nem `inverted_cache.gob`, que
                        # existe de verdade. O sinal aqui e o codigo que cria
                        # o arquivo citar o nome literal, como
                        # `internal/search/persist.go` faz com
                        # "inverted_cache.gob". Documentado na evidencia acima.
                        $Ausente = $Corpo -cnotmatch [regex]::Escape($Token)
                    }
                }
                elseif ($Token -match $PadraoSnake) {
                    $Regra = 'SNAKE_CASE'
                    $Ausente = $Corpo -cnotmatch [regex]::Escape($Token)
                }
                elseif ($Token -match $PadraoCamelCall) {
                    $Regra = 'CAMEL_CALL'
                    $Busca = $Token -replace '\(.*\)$', ''
                    $Ausente = $Corpo -cnotmatch [regex]::Escape($Busca)
                }
                else {
                    continue
                }

                if ($Ausente) {
                    Add-Achado -Arquivo $Doc.FullName -Linha ($i + 1) -Regra $Regra -Token $Token
                }
            }
        }
    }

    # Os reconhecidos saem SEMPRE, com o motivo ao lado. Uma lista de excecoes
    # que ninguem ve deixa de ser revisada, e um token que entrou nela por
    # engano fica escondido para sempre.
    if ($Vistos.Count -gt 0) {
        Write-Output ""
        Write-Output "[i] $($Vistos.Count) token(s) reconhecido(s) -- nao contam como achado:"
        $Vistos | ForEach-Object { Write-Output $_ }
    }

    Write-Output ""
    if ($Achados -eq 0) {
        Write-Output "[OK] nenhum token entre crases parece citar artefato ausente do codigo."
        exit 0
    }

    Write-Output "[!] $Achados achado(s). Nao e veredito -- cada linha e um token que"
    Write-Output "    precisa de uma pessoa confirmando se e artefato real, flag/nome"
    Write-Output "    externo (ruido esperado), ou decisao de doc que o codigo ainda"
    Write-Output "    nao cumpriu."
    exit 1
}
finally {
    Pop-Location
}
