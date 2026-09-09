//go:build windows

package instalar

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// A entrada de PATH do usuario mora no registro, em HKCU\Environment\Path.
//
// HKCU, e nunca HKLM: a decisao D-04 do dono e SEM ELEVACAO. O PATH de maquina
// exige administrador, e pedir UAC para instalar um servidor MCP e desproporcional.
const (
	chaveDoAmbiente = `Environment`
	valorDoPath     = "Path"
)

// AdicionarAoPath acrescenta dir ao PATH do usuario, se ele ainda nao estiver
// la. Devolve se mudou alguma coisa.
//
// Preserva o tipo do valor. Um PATH gravado como REG_EXPAND_SZ contem
// referencias como %USERPROFILE%, e reescreve-lo como REG_SZ congelaria essas
// referencias no valor que elas tinham hoje -- um estrago silencioso no PATH do
// usuario, causado por um instalador que so queria acrescentar uma linha.
func AdicionarAoPath(dir string) (mudou bool, err error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, chaveDoAmbiente, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return false, fmt.Errorf("abrindo HKCU\\%s: %w", chaveDoAmbiente, err)
	}
	defer func() { _ = k.Close() }()

	atual, tipo, err := k.GetStringValue(valorDoPath)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return false, fmt.Errorf("lendo o PATH do usuario: %w", err)
	}
	if errors.Is(err, registry.ErrNotExist) {
		tipo = registry.EXPAND_SZ
		atual = ""
	}

	if temNoPath(atual, dir) {
		return false, nil
	}

	novo := dir
	if atual != "" {
		novo = strings.TrimRight(atual, ";") + ";" + dir
	}

	if tipo == registry.EXPAND_SZ {
		err = k.SetExpandStringValue(valorDoPath, novo)
	} else {
		err = k.SetStringValue(valorDoPath, novo)
	}
	if err != nil {
		return false, fmt.Errorf("gravando o PATH do usuario: %w", err)
	}
	return true, nil
}

// RemoverDoPath tira dir do PATH do usuario. Devolve se mudou alguma coisa.
func RemoverDoPath(dir string) (mudou bool, err error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, chaveDoAmbiente, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return false, fmt.Errorf("abrindo HKCU\\%s: %w", chaveDoAmbiente, err)
	}
	defer func() { _ = k.Close() }()

	atual, tipo, err := k.GetStringValue(valorDoPath)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("lendo o PATH do usuario: %w", err)
	}

	var mantidos []string
	for _, parte := range strings.Split(atual, ";") {
		if parte == "" || mesmoDiretorio(parte, dir) {
			continue
		}
		mantidos = append(mantidos, parte)
	}
	novo := strings.Join(mantidos, ";")
	if novo == atual {
		return false, nil
	}

	if tipo == registry.EXPAND_SZ {
		err = k.SetExpandStringValue(valorDoPath, novo)
	} else {
		err = k.SetStringValue(valorDoPath, novo)
	}
	if err != nil {
		return false, fmt.Errorf("gravando o PATH do usuario: %w", err)
	}
	return true, nil
}

// AvisoDePath e o que o usuario precisa saber depois de o PATH mudar.
func AvisoDePath() string {
	return "abra um terminal NOVO para o PATH atualizado valer; a sessao atual mantem o PATH antigo"
}

// DiretorioPadrao e onde o binario vai sem elevacao (decisao D-04).
//
// %LOCALAPPDATA%\Programs e o lugar que o Windows reserva para programas por
// usuario -- o mesmo que VS Code, Cursor e Antigravity ja usam, e por isso a
// deteccao de hosts o consulta. Program Files exigiria UAC.
func DiretorioPadrao() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "Programs", "gobsidian")
}

// NomeDoExecutavel e o nome do arquivo instalado.
const NomeDoExecutavel = "gobsidian.exe"

// temNoPath diz se dir ja esta na lista separada por ";".
//
// Comparacao de DIRETORIO, e nao de texto: "C:\bin", "C:\bin\\" e "c:\BIN"
// sao o mesmo lugar, e um instalador que nao percebe isso acrescenta a mesma
// entrada a cada execucao ate o PATH estourar o limite do sistema.
func temNoPath(lista, dir string) bool {
	for _, parte := range strings.Split(lista, ";") {
		if mesmoDiretorio(parte, dir) {
			return true
		}
	}
	return false
}
