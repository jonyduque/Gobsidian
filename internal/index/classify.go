package index

import (
	"github.com/jonyd/gobsidian/internal/vault"
)

// classe diz o que uma entrada do cofre vira dentro do indice e se o conteudo
// dela precisa ser lido do disco.
type classe int

const (
	// classeAnexo entra como Asset: indexado por nome, nunca aberto. O cofre
	// so precisa saber que ele existe para nao chamar de quebrado o embed que
	// aponta para ele.
	classeAnexo classe = iota
	// classeNotaSemLeitura entra como Note com so o que a varredura de
	// diretorio ja sabia. E o placeholder do sincronizador de nuvem: abrir
	// dispararia download sincrono, e um `.md` nao hidratado continua sendo
	// uma nota — e o que o comentario de Note.CloudOnly promete.
	classeNotaSemLeitura
	// classeNotaLida entra como Note com o conteudo lido e parseado.
	classeNotaLida
)

// classificar e a UNICA funcao que decide o que uma entrada vira no indice.
//
// Build (pelo worker de leitura e por insert) e Replace passam as duas por
// aqui. Antes cada uma derivava a resposta por conta propria, e as duas
// divergiam no placeholder de nuvem: Build registrava Note com CloudOnly, e
// Replace registrava Asset — a mesma entrada respondendo diferente conforme
// qual construcao tivesse rodado, com idx.Get devolvendo false num caso e nao
// no outro. Mesma disciplina de aliasKey e de nomeChave: toda derivacao passa
// por uma funcao so, inclusive as que ja estavam certas, para que a proxima
// divergencia seja impossivel sem tocar nesta funcao.
func classificar(e vault.Entry) classe {
	if !e.IsNote {
		return classeAnexo
	}
	if e.CloudOnly {
		return classeNotaSemLeitura
	}
	return classeNotaLida
}

// precisaLer diz se o indexador tem de abrir o arquivo. E a mesma pergunta que
// classificar responde, com o nome do lado de quem le.
func (c classe) precisaLer() bool {
	return c == classeNotaLida
}

// notaSemLeitura monta a Note de uma entrada registrada sem abrir o arquivo.
//
// Title vazio e Hash zero nao sao lacuna: sao a afirmacao de que o conteudo
// nao foi lido. Title vazio e o mesmo estado de uma nota sem frontmatter e sem
// H1, e o chamador cai no nome do arquivo. Existe uma vez so para que as duas
// construcoes produzam o MESMO objeto, campo a campo.
func notaSemLeitura(e vault.Entry) *Note {
	return &Note{
		Path:      e.Path,
		TitleNorm: normalizeTitleForNote(""),
		Size:      e.Size,
		ModTime:   e.ModTime,
		CloudOnly: e.CloudOnly,
	}
}

// anexoDaEntrada monta o Asset de uma entrada, pelo mesmo motivo de
// notaSemLeitura.
func anexoDaEntrada(e vault.Entry) *Asset {
	return &Asset{
		Path:    e.Path,
		Size:    e.Size,
		ModTime: e.ModTime,
	}
}
