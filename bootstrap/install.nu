#!/usr/bin/env nu
# Bootstrap do gobsidian para nushell.
#
# Ele faz UMA coisa: baixa o executavel da ultima versao e o roda. Detectar
# cofre, mexer no PATH e configurar hosts de IA e trabalho do proprio binario
# desde 2026-09-08 (decisao D-01 do dono).
#
# A conferencia de SHA-256 NAO acontece aqui (decisao D-03): quem confere e o
# `gobsidian update`, o binario VELHO sobre o NOVO.
#
#   http get https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install.nu | save -f /tmp/gi.nu; nu /tmp/gi.nu

def main [...resto: string] {
    let repo = "jonyduque/Gobsidian"

    let ativo = if ($nu.os-info.name == "windows") {
        "gobsidian-windows-amd64.exe"
    } else if ($nu.os-info.name == "macos") {
        "gobsidian-darwin-arm64"
    } else {
        "gobsidian-linux-amd64"
    }

    let url = $"https://github.com/($repo)/releases/latest/download/($ativo)"
    let tmp = ($nu.temp-path | path join $"gobsidian_bootstrap_(random chars -l 12)")
    mkdir $tmp
    let exe = ($tmp | path join (if ($nu.os-info.name == "windows") { "gobsidian.exe" } else { "gobsidian" }))

    print $"[...] baixando ($ativo)"
    http get $url | save -f $exe
    if ($nu.os-info.name != "windows") { chmod +x $exe }

    print "[OK] baixado; entregando ao instalador do proprio binario"
    ^$exe install ...$resto
}
