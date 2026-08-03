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

$Snippets = @(
    "Esta e uma nota automatica com acentuacao portuguesa: verificacao, execucao, organizacao.",
    "O servidor MCP expoe o cofre local via JSON-RPC sobre stdio no ambiente Windows.",
    "Decisoes arquiteturais AD-01 a AD-09 regulam o comportamento do watcher e do indice.",
    "A busca full-text utiliza o algoritmo BM25 com pesos customizados para titulo e headings.",
    "Varredura incremental mantem o indice atualizado em tempo real com fsnotify."
)

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
    $body += $Snippets[$rand.Next($Snippets.Count)] + "`n`n"

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
            [void]$sb.AppendLine($Snippets[$rand.Next($Snippets.Count)])
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
