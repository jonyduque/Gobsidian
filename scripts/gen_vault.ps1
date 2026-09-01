#Requires -Version 7.0
<#
.SYNOPSIS
    Gerador determinístico de cofre sintético para testes de escala (RF-52, RNF-01).

.DESCRIPTION
    Produz um cofre Obsidian completo com 5.000 notas, anexos, tags desiguais,
    aliases, links validos e quebrados, mistura de BOM e EOL (CRLF/LF) e acentuacao
    em portugues. A partir de uma semente fixa, a geracao e 100% deterministica
    e reproduz o mesmo cofre byte a byte.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Out,

    [int]$Notes = 5000,

    [int]$Seed = 42,

    # Tamanho aproximado do CORPO de cada nota, em KB.
    #
    # O padrao gera notas de ~250 bytes, o que da 1,27 MB em 5.000 notas. Um
    # cofre real de estudo tem 109 MB em 3.148 notas -- 35 KB por nota. O custo
    # do indice invertido e proporcional aos BYTES tokenizados, nao a contagem
    # de notas, entao medir o boot com notas minusculas responde outra pergunta.
    [int]$BodyKB = 0
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if (Test-Path $Out) {
    Remove-Item $Out -Recurse -Force
}
New-Item -ItemType Directory -Path $Out -Force | Out-Null

$rand = [System.Random]::new($Seed)

$Folders = @(
    "",
    "Projetos",
    "Projetos/Gobsidian",
    "Projetos/Gobsidian/Docs",
    "Notas",
    "Notas/Pessoais",
    "Notas/Estudos",
    "Arquivo",
    "Arquivo/2025",
    "Arquivo/2026",
    "Tecnologia",
    "Tecnologia/Golang",
    "Tecnologia/PowerShell"
)

$PopularTags = @("projeto", "golang", "obsidian", "mcp", "estudo", "documentacao", "tarefa", "bug", "revisao", "ideia", "refatoracao", "teste")
$LongTailTags = 1..100 | ForEach-Object { "tag_raridade_$_" }

# VOCABULARIO, e nao um punhado de frases fixas.
#
# Ate 2026-09-01 este script tinha cinco frases repetidas ate encher o corpo, o
# que dava 50 termos distintos no cofre inteiro. Medido no mesmo dia: um cofre
# real do dono tem 46.786 termos distintos em 1.254 notas -- 936x mais.
#
# Por que isso importa mais do que parece: o indice invertido e construido sobre
# termos DISTINTOS. Com 50 termos cada um aparece em quase toda nota, e o
# resultado e uma tabela de termos minuscula com posting lists maximamente
# densas -- a forma OPOSTA de um cofre real, onde a tabela e enorme e a
# distribuicao e de Zipf. Medir memoria de busca contra esse cofre responde
# outra pergunta.
#
# As pecas se combinam em TRES dimensoes: prefixo x radical x sufixo. Sem o
# prefixo eram 60 x 40 = 2.400 formas, e o cofre saturava em 2.491 termos --
# medido, 5% do real. Com ele o espaco vai a ~48.000, que e a ordem de grandeza
# dos 46.786 termos de um cofre real de 1.254 notas.
#
# O texto nao precisa fazer sentido; precisa ter a CARDINALIDADE certa, porque e
# ela que dimensiona a tabela de termos do indice invertido.
$Prefixos = @(
    "", "", "", "in", "re", "sub", "inter", "contra", "pre", "pos", "extra",
    "infra", "supra", "anti", "co", "des", "im", "ultra", "retro", "trans"
)
$Radicais = @(
    "prescric", "decadenc", "usucapi", "servid", "enfiteus", "hipotec", "penhor",
    "anticres", "comodat", "mutu", "deposit", "mandat", "gestao", "corretag",
    "transport", "seguro", "fianc", "aval", "endoss", "protest", "falenc",
    "recuperac", "concordat", "insolvenc", "arrematac", "adjudicac", "remicao",
    "evicc", "redibitori", "empreitad", "locac", "arrendament", "parcer",
    "condomini", "incorporac", "loteament", "desapropriac", "tombament",
    "servidao", "alienac", "fiduciari", "leasing", "factoring", "franquia",
    "concessao", "permissao", "autorizac", "licenc", "outorg", "delegac",
    "avocac", "revogac", "anulac", "convalidac", "homologac", "ratificac",
    "denunciac", "resilic", "resoluc", "rescis"
)
$Sufixos = @(
    "ao", "oes", "ista", "istas", "ario", "arios", "aria", "arias", "avel",
    "aveis", "ivo", "ivos", "iva", "ivas", "ante", "antes", "ente", "entes",
    "encia", "encias", "ancia", "ancias", "mento", "mentos", "dade", "dades",
    "izacao", "izacoes", "ismo", "ismos", "idade", "idades", "orio", "orios",
    "atorio", "atorios", "ional", "ionais", "itude", "itudes"
)
$Qualificadores = @(
    "civil", "penal", "tributaria", "administrativa", "processual", "material",
    "constitucional", "trabalhista", "previdenciaria", "ambiental", "eleitoral",
    "empresarial", "consumerista", "internacional", "sumaria", "ordinaria",
    "cautelar", "executiva", "monitoria", "declaratoria", "constitutiva",
    "condenatoria", "mandamental", "preventiva", "repressiva", "originaria",
    "derivada", "solidaria", "subsidiaria", "concorrente"
)

# Monta uma frase nova a cada chamada. A semente e a mesma, entao o cofre
# continua deterministico -- o que muda e a CARDINALIDADE do vocabulario.
function New-Frase {
    # $Gerador, e nao $R: nome de variavel no PowerShell NAO distingue caixa,
    # entao `$r = $R.Next(...)` sobrescreve o proprio gerador com um inteiro na
    # primeira linha. Medido: 2.000 frases produziam 20 termos distintos em vez
    # de milhares. Mesma armadilha que o install.ps1 registra para $Pid.
    param([System.Random]$Gerador)
    $forma = $Gerador.Next(0, 4)
    $t1 = $Prefixos[$Gerador.Next($Prefixos.Count)] + $Radicais[$Gerador.Next($Radicais.Count)] + $Sufixos[$Gerador.Next($Sufixos.Count)]
    $t2 = $Prefixos[$Gerador.Next($Prefixos.Count)] + $Radicais[$Gerador.Next($Radicais.Count)] + $Sufixos[$Gerador.Next($Sufixos.Count)]
    $q1 = $Qualificadores[$Gerador.Next($Qualificadores.Count)]
    $q2 = $Qualificadores[$Gerador.Next($Qualificadores.Count)]
    switch ($forma) {
        0 { "A $t1 $q1 nao afasta a $t2 $q2 quando o prazo ja se consumou." }
        1 { "Discute-se a $t1 $q1 em face da $t2, com efeitos sobre a $q2." }
        2 { "O acordao firmou que a $t1 $q1 precede a $t2 $q2 na ordem de exame." }
        default { "Cabe $t1 $q1 contra a decisao que reconheceu a $t2 $q2." }
    }
}

$FileInfos = [System.Collections.Generic.List[PSCustomObject]]::new()
for ($i = 1; $i -le $Notes; $i++) {
    $folder = $Folders[$rand.Next($Folders.Count)]
    $name = if ($i % 10 -eq 0) { "Nota Acentuada $i - Execucao" } else { "Nota_$i" }
    $relPath = if ($folder -ne "") { "$folder/$name.md" } else { "$name.md" }
    $FileInfos.Add([PSCustomObject]@{
        Index   = $i
        RelPath = $relPath
        Name    = $name
    })
}

$AssetInfos = [System.Collections.Generic.List[PSCustomObject]]::new()
for ($a = 1; $a -le 50; $a++) {
    $folder = $Folders[$rand.Next($Folders.Count)]
    $ext = if ($a % 2 -eq 0) { "png" } else { "pdf" }
    $relPath = if ($folder -ne "") { "$folder/anexo_$a.$ext" } else { "anexo_$a.$ext" }
    $AssetInfos.Add([PSCustomObject]@{
        RelPath = $relPath
    })
}

$TotalLinks = 0
$BrokenLinks = 0

foreach ($info in $FileInfos) {
    $numTags = $rand.Next(1, 4)
    $noteTags = [System.Collections.Generic.List[string]]::new()
    for ($t = 0; $t -lt $numTags; $t++) {
        if ($rand.NextDouble() -lt 0.8) {
            $tag = $PopularTags[$rand.Next($PopularTags.Count)]
            if (-not $noteTags.Contains($tag)) { $noteTags.Add($tag) }
        } else {
            $tag = $LongTailTags[$rand.Next($LongTailTags.Count)]
            if (-not $noteTags.Contains($tag)) { $noteTags.Add($tag) }
        }
    }

    $hasAlias = ($rand.NextDouble() -lt 0.2)
    $aliasesStr = if ($hasAlias) { "aliases: [Alias_$($info.Index), OutroAlias_$($info.Index)]`n" } else { "" }

    $fm = "---`n"
    $fm += "title: ""$($info.Name)""`n"
    $fm += "tags: [$($noteTags -join ', ')]`n"
    if ($aliasesStr) { $fm += $aliasesStr }
    $fm += "---`n"

    $body = "# Heading Principal`n`n"
    $body += (New-Frase -Gerador $rand) + "`n`n"

    $numLinks = $rand.Next(1, 4)
    for ($l = 0; $l -lt $numLinks; $l++) {
        $TotalLinks++
        $r = $rand.NextDouble()
        if ($r -lt 0.7) {
            $targetIdx = $rand.Next($FileInfos.Count)
            $targetNote = $FileInfos[$targetIdx].Name
            $body += "Veja [[$targetNote]] para mais detalhes.`n"
        } elseif ($r -lt 0.85) {
            $targetIdx = $rand.Next($FileInfos.Count)
            $targetNote = $FileInfos[$targetIdx].Name
            $body += "Veja [[$targetNote#Heading Principal]] no capitulo.`n"
        } else {
            $BrokenLinks++
            $body += "Link quebrado: [[nota_inexistente_$TotalLinks]].`n"
        }
    }

    if ($rand.NextDouble() -lt 0.1) {
        $assetIdx = $rand.Next($AssetInfos.Count)
        $body += "Anexo: ![[anexo_$($assetIdx + 1).png]]`n"
    }

    if ($BodyKB -gt 0) {
        # Enche com frases do mesmo pool, para o texto continuar parecido com
        # prosa e o analisador ter o que fazer. Repetir uma frase so produziria
        # um dicionario de termos irrealmente pequeno.
        $alvoBytes = $BodyKB * 1024
        $sb = New-Object System.Text.StringBuilder
        while ($sb.Length -lt $alvoBytes) {
            [void]$sb.AppendLine((New-Frase -Gerador $rand))
        }
        $body += "`n" + $sb.ToString()
    }

    $contentStr = $fm + "`n" + $body

    $hasCRLF = ($info.Index % 2 -eq 0)
    if ($hasCRLF) {
        $contentStr = $contentStr -replace "`r`n", "`n" -replace "`n", "`r`n"
    } else {
        $contentStr = $contentStr -replace "`r`n", "`n"
    }

    $encoding = [System.Text.UTF8Encoding]::new($false)
    $hasBOM = ($info.Index % 3 -eq 0)
    if ($hasBOM) {
        $encoding = [System.Text.UTF8Encoding]::new($true)
    }

    $fullPath = [System.IO.Path]::Combine($Out, $info.RelPath.Replace('/', [System.IO.Path]::DirectorySeparatorChar))
    $parentDir = [System.IO.Path]::GetDirectoryName($fullPath)
    if (-not (Test-Path $parentDir)) {
        [System.IO.Directory]::CreateDirectory($parentDir) | Out-Null
    }

    [System.IO.File]::WriteAllText($fullPath, $contentStr, $encoding)
}

foreach ($asset in $AssetInfos) {
    $fullPath = [System.IO.Path]::Combine($Out, $asset.RelPath.Replace('/', [System.IO.Path]::DirectorySeparatorChar))
    $parentDir = [System.IO.Path]::GetDirectoryName($fullPath)
    if (-not (Test-Path $parentDir)) {
        [System.IO.Directory]::CreateDirectory($parentDir) | Out-Null
    }
    [System.IO.File]::WriteAllBytes($fullPath, [byte[]](0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A))
}

$TotalBytes = (Get-ChildItem -Path $Out -Recurse -File | Measure-Object -Property Length -Sum).Sum
$MB = [math]::Round($TotalBytes / 1MB, 2)

Write-Output "[OK] Cofre sintético gerado em $Out"
Write-Output "[*] Notas: $Notes"
Write-Output "[*] Anexos: $($AssetInfos.Count)"
Write-Output "[*] Tamanho total: $MB MB ($TotalBytes bytes)"
Write-Output "[*] Links totais: $TotalLinks"
Write-Output "[*] Links quebrados: $BrokenLinks"
