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
#   http get https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install.nu | nu --stdin -c $in
#
# # Por que o corpo esta no NIVEL SUPERIOR, e nao dentro de `def main`
#
# A convencao do `main` so vale quando o nushell executa um ARQUIVO. Em modo
# `-c` -- que e o que a linha acima usa -- ele avalia o codigo, define a funcao
# e sai. Medido em 2026-09-09: com o corpo dentro de `def main`, aquela linha
# imprimia o topo do script e terminava com codigo 0 sem baixar nada.
#
# Este script nao aceita flag nenhuma, e isso tambem e consequencia do `-c`: o
# nushell consome os argumentos para si, e `nu --stdin -c $in --vault X`
# responde "Unknown flag '--vault'". Quem precisa passar flag usa o
# install-flags.nu ao lado -- ver o README.

let ativo = if ($nu.os-info.name == "windows") {
    "gobsidian-windows-amd64.exe"
} else if ($nu.os-info.name == "macos") {
    "gobsidian-darwin-arm64"
} else {
    "gobsidian-linux-amd64"
}

# O diretorio temporario, com as tres formas que existem no mundo.
#
# TMP e TEMP existem no Windows e TMPDIR no resto, e acessar qualquer um deles
# direto e ERRO DURO no nushell quando falta -- "column 'TEMP' is missing". O
# `?` torna o acesso opcional e o `default` encadeia.
#
# O `-p` do mktemp nao e opcional: sem ele, o mktemp usa o diretorio ATUAL, e o
# bootstrap largaria o binario onde quer que o usuario tenha rodado o comando.
#
# $nu.temp-path, que estava aqui ate 2026-09-09, nao existe mais nesta versao.
let base = ($env.TMP? | default $env.TEMP? | default $env.TMPDIR? | default "/tmp")
let url = $"https://github.com/jonyduque/Gobsidian/releases/latest/download/($ativo)"
let tmp = (mktemp -p $base $"XXXXXX($ativo)")

print $"[...] baixando ($ativo)"
http get $url | save -f $tmp
if ($nu.os-info.name != "windows") { chmod +x $tmp }

print "[OK] baixado; entregando ao instalador do proprio binario"
^$tmp install
