//go:build windows

package instalar

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ProcessosDoSistema lista os processos cujo executavel se chama
// NomeDoExecutavel.
//
// # Por que existe, se a presenca e trava de kernel
//
// A presenca so enxerga quem a registra, e ela entrou em 2026-09-09. Medido em
// 2026-09-14: cinco processos v1.5.1, abertos pelo Claude Code a partir de
// C:\Program Files\gobsidian, serviam cofres sem aparecer no `doctor`. Esta
// listagem NAO decide nada -- quem encerra processo continua sendo a presenca,
// pelo motivo escrito em Registrar. Ela so torna visivel quem a presenca nao
// ve.
//
// Codigo de plataforma, atras de build tag: e o custo de responder pelo
// sistema operacional, e por isso ele fica restrito ao diagnostico.
func ProcessosDoSistema() ([]ProcessoDoSistema, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("listando processos: %w", err)
	}
	defer func() { _ = windows.CloseHandle(snap) }()

	var entrada windows.ProcessEntry32
	entrada.Size = uint32(unsafe.Sizeof(entrada))

	var saida []ProcessoDoSistema
	for err = windows.Process32First(snap, &entrada); err == nil; err = windows.Process32Next(snap, &entrada) {
		if !strings.EqualFold(windows.UTF16ToString(entrada.ExeFile[:]), NomeDoExecutavel) {
			continue
		}
		saida = append(saida, ProcessoDoSistema{
			PID:        int(entrada.ProcessID),
			Executavel: caminhoDoExecutavel(entrada.ProcessID),
		})
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return saida, fmt.Errorf("percorrendo processos: %w", err)
	}
	return saida, nil
}

// caminhoDoExecutavel devolve o caminho completo do executavel de pid, ou "?"
// quando o sistema nao deixa perguntar. O caminho e o que separa duas
// instalacoes com o mesmo nome de arquivo.
func caminhoDoExecutavel(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "?"
	}
	defer func() { _ = windows.CloseHandle(h) }()

	buf := make([]uint16, windows.MAX_LONG_PATH)
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return "?"
	}
	return windows.UTF16ToString(buf[:n])
}
