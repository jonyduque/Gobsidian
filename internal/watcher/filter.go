package watcher

import (
	"log/slog"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Op descreve a operacao que gerou o evento.
type Op int

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

// filter verifica se o evento do fsnotify e relevante.
// Retorna um evento de dominio e um booleano indicando se deve ser emitido.
// Eventos de Chmod e de caminhos irrelevantes sao descartados.
func filter(e fsnotify.Event, root string, log *slog.Logger) (Event, bool) {
	if e.Op&fsnotify.Chmod == fsnotify.Chmod {
		// Ignora mudanca de permissao que e muito comum e irrelevante para conteudo.
		return Event{}, false
	}

	canon, err := vault.Canonicalize(root, e.Name)
	if err != nil {
		log.Warn("Evento sobre caminho invalido", "path", e.Name, "err", err)
		return Event{}, false
	}

	class := vault.Classify(canon)
	if class == vault.ClassExcluded {
		return Event{}, false
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
		return Event{}, false
	}

	return Event{
		Path: canon,
		Op:   op,
	}, true
}
