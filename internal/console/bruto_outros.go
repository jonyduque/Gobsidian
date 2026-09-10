//go:build !windows && !linux && !darwin

package console

import (
	"errors"
	"os"
)

// entrarNoModoBruto nas plataformas que este projeto nao mira.
//
// Devolver erro aqui NAO degrada nada: quem chama Selecionar cai na pergunta
// digitada, que e o mesmo caminho de quando a entrada e um pipe. O que nao
// pode e o pacote deixar de compilar num GOOS qualquer.
func entrarNoModoBruto(*os.File) (restaurar func(), err error) {
	return nil, errors.New("plataforma sem modo bruto de terminal")
}
