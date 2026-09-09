#!/usr/bin/env sh
# Bootstrap do gobsidian para Linux e macOS.
#
# Ele faz UMA coisa: baixa o executavel da ultima versao e o roda. Detectar
# cofre, mexer no PATH e configurar hosts de IA e trabalho do proprio binario
# desde 2026-09-08 (decisao D-01 do dono) -- la isso e testavel, e aqui nao era.
#
# A conferencia de SHA-256 NAO acontece aqui, e isso e deliberado (decisao
# D-03): quem confere e o `gobsidian update`, o binario VELHO sobre o NOVO. Um
# binario nao verifica a si mesmo com credibilidade depois de ja estar rodando,
# e um script que finge conferir da uma garantia que nao tem.
#
#   curl -fsSL https://raw.githubusercontent.com/jonyduque/Gobsidian/master/bootstrap/install.sh | sh
#
# Argumentos passam adiante:
#   ... | sh -s -- --vault "/caminho/do/cofre" --yes
set -eu

REPO="jonyduque/Gobsidian"

case "$(uname -s)" in
    Linux)  SO="linux" ;;
    Darwin) SO="darwin" ;;
    *) echo "[!] sistema nao suportado: $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "[!] arquitetura nao suportada: $(uname -m)" >&2; exit 1 ;;
esac

ATIVO="gobsidian-${SO}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ATIVO}"

TMP="$(mktemp -d)"
# O temporario some mesmo se o download falhar no meio.
trap 'rm -rf "$TMP"' EXIT

echo "[...] baixando ${ATIVO}"
if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$URL" -o "$TMP/gobsidian"
elif command -v wget >/dev/null 2>&1; then
    wget -q "$URL" -O "$TMP/gobsidian"
else
    echo "[!] nem curl nem wget encontrados" >&2
    exit 1
fi

chmod +x "$TMP/gobsidian"
echo "[OK] baixado; entregando ao instalador do proprio binario"
exec "$TMP/gobsidian" install "$@"
