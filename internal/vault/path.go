package vault

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// CanonicalPath e a unica forma interna de identificar uma nota: relativa a
// raiz do cofre, com separador "/", sem "./" inicial, e com a grafia exata do
// disco — maiusculas e minusculas inclusive.
//
// A grafia exata importa. Cofres reais acumulam inconsistencia de casing entre
// pastas, e o indice precisa refletir o que esta no disco, nao uma
// normalizacao inventada.
//
// ATENCAO: esta camada nao consulta o disco e portanto NAO garante a grafia
// dele. Ela preserva o que o chamador passou. Quem produz a grafia real e a
// varredura (vault.Walk), que le as entradas de diretorio; quem corrige a
// grafia de uma entrada vinda de uma tool e a resolucao insensivel a
// maiusculas do indice. Tratar a saida de Resolve como se fosse a grafia do
// disco e um erro que so aparece em cofre com casing inconsistente.
type CanonicalPath string

// Sentinelas distintas de proposito: cada uma vira um codigo de erro MCP
// diferente, e o cliente decide o que fazer em seguida a partir do codigo.
// Colapsar as quatro em uma tornaria a mensagem a unica informacao util.
var (
	ErrOutsideVault = errors.New("caminho fora do cofre")
	ErrAbsolutePath = errors.New("caminho absoluto nao aceito")
	ErrEmptyPath    = errors.New("caminho vazio")
	ErrInvalidPath  = errors.New("caminho malformado")
)

// Resolve traduz a entrada de uma tool em caminho absoluto e canonico.
//
// O confinamento tem duas camadas, e as duas sao necessarias. A verificacao
// lexica (validateLocal) rejeita formas que o sistema operacional resolveria
// para fora do cofre sem nunca produzir um `..`. A verificacao por componente
// (Canonicalize) rejeita a travessia propriamente dita.
//
// LIMITE CONHECIDO: a verificacao e puramente lexica. Nao resolve links
// simbolicos nem junctions. Um link dentro do cofre apontando para fora e
// aceito aqui, e rejeita-lo exige I/O — responsabilidade da camada que abre o
// arquivo, nao desta.
func Resolve(root, input string) (string, CanonicalPath, error) {
	if strings.TrimSpace(input) == "" {
		return "", "", ErrEmptyPath
	}

	cleaned := path.Clean(filepath.ToSlash(input))
	if cleaned == "." {
		return "", "", ErrEmptyPath
	}

	if err := validateLocal(cleaned); err != nil {
		return "", "", err
	}

	abs := filepath.Join(root, filepath.FromSlash(cleaned))
	canon, err := Canonicalize(root, abs)
	if err != nil {
		return "", "", err
	}
	return abs, canon, nil
}

// validateLocal rejeita as formas que a comparacao por componente nao enxerga.
//
// Tres classes, cada uma descoberta em revisao com prova empirica:
//
//  1. Nomes de dispositivo do Windows. `Resolve(root, "NUL")` produz um
//     caminho lexicamente interno ao cofre, e o sistema operacional o resolve
//     para o dispositivo NUL. `COM1` aceita escrita em porta serial; `CON` le
//     o console, que em um servidor stdio e o proprio transporte. Nenhum
//     desses caminhos contem `..`, entao a checagem por componente passa.
//
//  2. Caminhos com letra de drive que sobrevivem ao Clean. `./C:/Windows`
//     vira `C:/Windows` depois de path.Clean, e uma checagem que so olha os
//     dois primeiros bytes da entrada crua nao ve isso.
//
//  3. Componentes terminados em ponto ou espaco. O Win32 os remove ao abrir,
//     entao `A.md.`, `A.md ` e `A.md` sao o mesmo arquivo — mas tres chaves
//     canonicas distintas. Como CanonicalPath e a identidade da nota no
//     indice, isso da a um arquivo mais de uma identidade, e qualquer
//     verificacao chaveada nela pode ser contornada acrescentando um ponto.
//
// filepath.IsLocal cobre 1 e 2 e substitui a checagem manual de letra de
// drive. A classe 3 ele considera local, e precisa de rejeicao explicita.
func validateLocal(cleaned string) error {
	if strings.ContainsRune(cleaned, 0) {
		return fmt.Errorf("%w: caminho contem byte nulo", ErrInvalidPath)
	}

	if !filepath.IsLocal(filepath.FromSlash(cleaned)) {
		return fmt.Errorf("%w: %q", ErrAbsolutePath, cleaned)
	}

	for _, part := range strings.Split(cleaned, "/") {
		if part == "" {
			continue
		}
		if last := part[len(part)-1]; last == '.' || last == ' ' {
			return fmt.Errorf(
				"%w: componente %q termina em ponto ou espaco, o que o Windows remove ao abrir e faria o mesmo arquivo ter mais de um caminho canonico",
				ErrInvalidPath, part)
		}
	}

	return nil
}

// Canonicalize converte um caminho absoluto em CanonicalPath, verificando o
// confinamento por componente de caminho.
func Canonicalize(root, abs string) (CanonicalPath, error) {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrOutsideVault, abs)
	}

	slashed := filepath.ToSlash(rel)
	if slashed == ".." || strings.HasPrefix(slashed, "../") {
		return "", fmt.Errorf("%w: %q", ErrOutsideVault, abs)
	}
	if slashed == "." {
		return "", fmt.Errorf("%w: %q e a propria raiz do cofre, nao uma nota", ErrInvalidPath, abs)
	}

	// Invariante de saida. Canonicalize e exportada e alcancavel sem passar
	// por Resolve — a varredura da Task 8 chama direto. Sem esta linha,
	// entradas como root+"/C:Windows/win.ini" produziriam um CanonicalPath
	// com letra de drive, que viraria chave do indice e que o proprio Resolve
	// rejeitaria se recebesse de volta. A forma canonica precisa ser estavel
	// sob ida e volta.
	if err := validateLocal(slashed); err != nil {
		return "", err
	}

	return CanonicalPath(slashed), nil
}
