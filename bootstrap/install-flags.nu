#!/usr/bin/env nu
# Bootstrap do gobsidian para nushell, com repasse de flags.
#
# O irmao install.nu faz o caso simples e roda direto de um cano:
#
#   http get .../install.nu | nu --stdin -c $in
#
# Esta versao existe porque aquela NAO consegue receber flag. Em modo `-c` o
# nushell consome os argumentos para si -- `nu --stdin -c $in --vault X`
# responde "Unknown flag '--vault'" --, e a convencao do `main`, que e o que
# permite receber argumentos, so vale quando o nushell executa um ARQUIVO.
#
# Por isso este aqui e salvo em disco antes de rodar:
#
#   http get https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install-flags.nu
#     | save -f ($env.TMP | path join gobsidian-install.nu)
#   nu ($env.TMP | path join gobsidian-install.nu) -- --vault "C:\Meu Cofre" --yes
#
# O `--` antes das flags NAO e enfeite: sem ele o nushell tenta interpretar
# `--vault` como flag DELE, e nao do script. Medido em 2026-09-09.
#
# Tudo o mais e igual ao install.nu: baixa o executavel da ultima versao e
# entrega o trabalho ao instalador que mora dentro dele (decisao D-01), sem
# conferir SHA-256 aqui (decisao D-03 -- quem confere e o `gobsidian update`).

def main [...resto: string] {
    let ativo = if ($nu.os-info.name == "windows") {
        "gobsidian-windows-amd64.exe"
    } else if ($nu.os-info.name == "macos") {
        "gobsidian-darwin-arm64"
    } else {
        "gobsidian-linux-amd64"
    }

    # A mesma cadeia de install.nu, pela mesma razao: TMP e TEMP existem no
    # Windows e TMPDIR no resto, e acesso direto a um que falta e erro duro.
    let base = ($env.TMP? | default $env.TEMP? | default $env.TMPDIR? | default "/tmp")
    let url = $"https://github.com/jonyduque/Gobsidian/releases/latest/download/($ativo)"
    let tmp = (mktemp -p $base $"XXXXXX($ativo)")

    print $"[...] baixando ($ativo)"
    http get $url | save -f $tmp
    if ($nu.os-info.name != "windows") { chmod +x $tmp }

    print "[OK] baixado; entregando ao instalador do proprio binario"
    ^$tmp install ...$resto
}
