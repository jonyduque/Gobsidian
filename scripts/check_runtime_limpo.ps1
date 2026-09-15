#Requires -Version 7.0
<#
.SYNOPSIS
    Prova que nenhum teste alcancou o diretorio do USUARIO onde o gobsidian
    guarda runtime e cache.

.DESCRIPTION
    Medido em 2026-09-14: `go test -count=1` sobre ipc, daemon, doctor,
    instalar e cmd/gobsidian levou %LOCALAPPDATA%\gobsidian\run de 116 para 132
    arquivos -- 15 travas com chave aleatoria e um arquivo de presenca --, e
    nada foi removido. Rodando de novo, 47 para 64. Cada teste que abre socket
    num cofre de t.TempDir() nomeia o socket, a trava e o log pela chave do
    cofre, e o diretorio que resolve esses caminhos era o do usuario.

    check_test_isolation.ps1 ja recusa o desvio por variavel de ambiente DENTRO
    do teste. O que ele nao ve e o teste que nao desvia nada: esse chega na
    funcao de producao e escreve no perfil real sem nenhuma linha suspeita.

    # Por que isca, e nao fotografia do diretorio real

    O plano (I1.2) descrevia fotografar %LOCALAPPDATA%\gobsidian\run antes e
    depois de `go test` e reprovar arquivo novo. Numa maquina com o produto no
    ar isso reprova a toa: o Claude Desktop e o Antigravity abrem pontes e
    daemons a qualquer momento, e cada partida cria trava e presenca. Gate que
    reprova sem defeito e gate que se aprende a pular.

    A isca mede a mesma coisa sem o ruido. O verify.ps1 roda `go test` com
    LOCALAPPDATA, XDG_RUNTIME_DIR e XDG_CACHE_HOME apontando para diretorios
    vazios e descartaveis -- variaveis do PROCESSO `go test`, nao do teste,
    entao nao cai na regra do check_test_isolation. Um teste isolado nunca
    passa pela funcao de producao e a isca fica vazia. Um teste que escapa do
    isolamento resolve o caminho de producao, e o caminho de producao, agora,
    e a isca: qualquer coisa sob <isca>/*/gobsidian e o vazamento, com nome.

    LIMITE CONHECIDO: no macOS os.UserCacheDir ignora XDG_CACHE_HOME, entao
    vazamento para o CACHE nao cai na isca naquela plataforma. O diretorio de
    runtime cai: no Unix ele sai de XDG_RUNTIME_DIR. O verify.ps1 roda no
    Windows, onde as duas metades caem.

.PARAMETER Isca
    Raiz da isca, com os subdiretorios local, runtime e cache.

.NOTES
    Saida em ASCII puro. Sai 1 se algo existir sob <isca>/*/gobsidian.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$Isca,
    [switch]$Silencioso
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Cada subdiretorio da isca e a variavel que aponta para ele. So a subarvore
# `gobsidian` conta: outras ferramentas podem escrever no LOCALAPPDATA falso, e
# isso nao e defeito deste projeto.
#
# O subdiretorio do produto nao tem o mesmo nome nos quatro: no perfil ele e
# `.gobsidian`, onde o runtime do Windows mora desde 2026-09-14.
$Alvos = @(
    [pscustomobject]@{ Sub = 'local';   Nome = 'gobsidian';  Via = 'LOCALAPPDATA (cache no Windows; runtime ate 2026-09-14)' }
    [pscustomobject]@{ Sub = 'perfil';  Nome = '.gobsidian'; Via = 'USERPROFILE/HOME (runtime no Windows desde 2026-09-14)' }
    [pscustomobject]@{ Sub = 'runtime'; Nome = 'gobsidian';  Via = 'XDG_RUNTIME_DIR (runtime no Unix)' }
    [pscustomobject]@{ Sub = 'cache';   Nome = 'gobsidian';  Via = 'XDG_CACHE_HOME (cache no Linux)' }
)

$Achados = @()
foreach ($alvo in $Alvos) {
    $raiz = Join-Path (Join-Path $Isca $alvo.Sub) $alvo.Nome
    if (-not (Test-Path -LiteralPath $raiz)) { continue }
    # O diretorio existir ja e o vazamento: Listen e Registrar criam o
    # diretorio antes de criar o arquivo. Lista o conteudo para dizer quem.
    $itens = @(Get-ChildItem -LiteralPath $raiz -Recurse -Force -ErrorAction SilentlyContinue)
    if ($itens.Count -eq 0) {
        $Achados += "$($alvo.Sub)/$($alvo.Nome)/ (diretorio vazio) -- via $($alvo.Via)"
        continue
    }
    foreach ($i in $itens) {
        $rel = $i.FullName.Substring($Isca.Length).TrimStart('\', '/') -replace '\\', '/'
        $Achados += "$rel -- via $($alvo.Via)"
    }
}

if ($Achados.Count -gt 0) {
    Write-Output "[!] testes escreveram no diretorio do usuario (resolvido para a isca):"
    $Achados | ForEach-Object { Write-Output "     $_" }
    Write-Output ""
    Write-Output "     O pacote que criou esses arquivos precisa de"
    Write-Output "     func TestMain(m *testing.M) { os.Exit(ipc.RodarComRuntimeIsolado(m)) }"
    Write-Output "     -- ver docs/ARMADILHAS.md."
    exit 1
}

if (-not $Silencioso) {
    Write-Output "[OK] nenhum teste escreveu no diretorio de runtime ou de cache do usuario"
}
exit 0
