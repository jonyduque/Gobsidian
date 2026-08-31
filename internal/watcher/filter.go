package watcher

import (
	"log/slog"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Op descreve a operacao que gerou o evento.
type Op int

// As quatro operacoes que o watcher propaga. Chmod nao entra: o OneDrive o
// emite constantemente e ele nunca significa mudanca de conteudo.
const (
	OpCreate Op = iota
	OpWrite
	OpRemove
	OpRename
)

// Event e o evento de dominio que o watcher exporta.
type Event struct {
	Path vault.CanonicalPath
	Op   Op
}

// DropReason e fechado: quatro valores, e nenhum outro entra sem mudar o plano.
type DropReason string

// Os quatro motivos de descarte, contados separadamente porque pedem acoes
// diferentes: chmod alto e OneDrive em operacao normal; outside_vault alto
// indica que a raiz e um link e o confinamento esta recusando eventos;
// excluded alto indica atividade em .obsidian/ ou .git/; unknown_op alto
// indica evento que o filtro nao soube classificar.
const (
	DropChmod        DropReason = "chmod"
	DropOutsideVault DropReason = "outside_vault"
	DropExcluded     DropReason = "excluded"
	DropUnknownOp    DropReason = "unknown_op"
)

// filter verifica se o evento do fsnotify e relevante.
// Retorna um evento de dominio, um booleano indicando se deve ser emitido, e o motivo do descarte caso contrario.
func filter(e fsnotify.Event, root string, log *slog.Logger) (Event, bool, DropReason) {
	// Chmod SOZINHO e irrelevante para conteudo e muito comum. Chmod
	// ACOMPANHADO de Write ou Create nao e: o evento traz uma mudanca de
	// conteudo junto.
	//
	// Ate 2026-08-28 o teste era `e.Op&Chmod == Chmod`, que e verdadeiro para
	// qualquer mascara QUE CONTENHA Chmod — entao um `Write|Chmod` era
	// descartado inteiro e a nota so voltava ao indice no proximo boot (achado
	// M11). No Windows o backend raramente compoe mascaras, o que conteve o
	// dano; em kqueue (macOS, BSD) compor e o padrao, e linux/darwin sao alvos
	// declarados que nao tem reconciliacao por overflow — la nao ha rede de
	// seguranca nenhuma.
	if e.Op == fsnotify.Chmod {
		return Event{}, false, DropChmod
	}

	canon, err := vault.Canonicalize(root, e.Name)
	if err != nil {
		log.Warn("Evento sobre caminho invalido", "path", e.Name, "err", err)
		return Event{}, false, DropOutsideVault
	}

	class := vault.Classify(canon)
	if class == vault.ClassExcluded {
		return Event{}, false, DropExcluded
	}

	var op Op
	switch {
	case e.Op&fsnotify.Create == fsnotify.Create:
		op = OpCreate
	case e.Op&fsnotify.Write == fsnotify.Write:
		op = OpWrite
	case e.Op&fsnotify.Remove == fsnotify.Remove:
		op = OpRemove
	case e.Op&fsnotify.Rename == fsnotify.Rename:
		op = OpRename
	default:
		// Não deveria ocorrer pois já filtramos Chmod, mas descarta.
		return Event{}, false, DropUnknownOp
	}

	return Event{
		Path: canon,
		Op:   op,
	}, true, ""
}
