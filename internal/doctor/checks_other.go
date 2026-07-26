//go:build !windows

package doctor

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// platformChecks devolve vazio fora do Windows: caminhos longos exigem um
// opt-in de registro que so existe la, arquivos somente-nuvem sao detectados
// por atributo NTFS especifico do OneDrive, e colisao de casing so importa
// em sistema de arquivos insensivel a maiusculas.
func platformChecks() []check {
	return nil
}

// diskFreeBytes usa statfs(2). Bavail (espaco disponivel a um usuario sem
// privilegio) e o numero relevante para saber se uma escrita do produto vai
// caber — Bfree inclui a reserva do superusuario, que o processo normalmente
// nao pode gastar.
func diskFreeBytes(path string) (uint64, error) {
	var buf unix.Statfs_t
	if err := unix.Statfs(path, &buf); err != nil {
		return 0, fmt.Errorf("statfs(%q): %w", path, err)
	}
	return uint64(buf.Bsize) * buf.Bavail, nil
}
