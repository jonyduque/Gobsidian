#!/usr/bin/env bash
# Instalador do gobsidian para macOS e Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.bash | bash
#
# Precisa de um terminal de verdade: o script pergunta quais cofres configurar.
# Num pipe o stdin ja pertence ao curl, entao ele e reaberto de /dev/tty; sem
# isso os prompts leriam EOF e o instalador seguiria sem escolha nenhuma.
set -euo pipefail

command -v node >/dev/null 2>&1 || {
	echo "⚠️  Instale Node.js 18+: https://nodejs.org" >&2
	exit 1
}

URL="https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.js"

f="$(mktemp)"
rm -f "$f"
f="$f.mjs"
trap 'rm -f "$f"' EXIT

curl -fsSL "$URL" -o "$f"

if [ -t 0 ]; then
	node "$f" "$@"
elif [ -e /dev/tty ]; then
	node "$f" "$@" </dev/tty
else
	echo "ℹ️  Sem terminal interativo; passe --vault e --hosts." >&2
	node "$f" "$@"
fi
