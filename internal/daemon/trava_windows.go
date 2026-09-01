//go:build windows

package daemon

import (
	"errors"

	"golang.org/x/sys/windows"
)

// travarArquivo pede a trava exclusiva do kernel, sem bloquear.
//
// LOCKFILE_FAIL_IMMEDIATELY e o par de LOCK_NB do Unix: sem ele a chamada
// BLOQUEIA ate o dono soltar, e quem perde a disputa deve sair, nao esperar.
//
// A trava do Windows e por FAIXA DE BYTES, e a faixa escolhida fica LONGE dos
// dados de proposito.
//
// Travar o byte 0 tornaria o PID ilegivel para qualquer outro processo: no
// Windows a faixa travada recusa leitura alheia com ERROR_LOCK_VIOLATION, e o
// `doctor`, que le esse PID para dizer quem detem o lock, passaria a falhar.
// Foi assim que este detalhe apareceu — um teste tentou ler o arquivo e recebeu
// "another process has locked a portion of the file".
//
// A faixa e o byte em 1<<62, que nenhum conteudo real alcanca. O arquivo tem
// alguns bytes; a trava mora numa regiao que nunca vai existir.
func travarArquivo(fd uintptr) (bool, error) {
	sobreposto := faixaDaTrava()
	err := windows.LockFileEx(
		windows.Handle(fd),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, sobreposto,
	)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
		return false, nil
	}
	return false, err
}

// destravarArquivo solta a faixa travada por travarArquivo.
func destravarArquivo(fd uintptr) error {
	sobreposto := faixaDaTrava()
	return windows.UnlockFileEx(windows.Handle(fd), 0, 1, 0, sobreposto)
}

// faixaDaTrava nomeia o byte que as duas pontas travam e destravam. Uma conta
// so: offsets diferentes entre travar e destravar deixariam a trava presa ate
// o processo morrer.
func faixaDaTrava() *windows.Overlapped {
	const deslocamento = uint64(1) << 62
	return &windows.Overlapped{
		Offset:     uint32(deslocamento & 0xFFFFFFFF),
		OffsetHigh: uint32(deslocamento >> 32),
	}
}
