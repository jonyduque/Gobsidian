package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Watcher escuta mudancas no cofre e emite eventos filtrados.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	root      string
	log       *slog.Logger
	events    chan Event

	// Contadores (Task 32)
	received int64
	dropped  int64
}

// New cria um novo Watcher observando a raiz do cofre.
func New(v *vault.Vault, log *slog.Logger) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("criando fsnotify.Watcher: %w", err)
	}

	root := v.Root() // root can be fetched directly or via getter.

	if err := fsWatcher.Add(root); err != nil {
		fsWatcher.Close()
		return nil, fmt.Errorf("observando raiz %q: %w", root, err)
	}

	// fsnotify v1.10.1 no Windows não é recursivo. Adiciona subdiretorios dinamicamente.
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Ignora erro em subpasta para nao falhar tudo
		}
		if d.IsDir() {
			// Se for excluido, pula (por exemplo .git)
			canon, errCanon := vault.Canonicalize(root, path)
			if errCanon == nil {
				if vault.Classify(canon) == vault.ClassExcluded {
					return filepath.SkipDir
				}
			}
			fsWatcher.Add(path)
		}
		return nil
	})

	return &Watcher{
		fsWatcher: fsWatcher,
		root:      root,
		log:       log,
		events:    make(chan Event, 100),
	}, nil
}

// Events devolve o canal onde os eventos sao publicados.
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Run entra em loop processando eventos do fsnotify ate o contexto ser cancelado
// ou o watcher ser fechado.
func (w *Watcher) Run(ctx context.Context) error {
	defer close(w.events)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return nil
			}
			w.log.Error("Erro do fsnotify", "err", err)
		case e, ok := <-w.fsWatcher.Events:
			if !ok {
				return nil
			}
			w.received++

			// Se um diretorio novo for criado, adiciona o watch nele
			if e.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(e.Name)
				if err == nil && info.IsDir() {
					canon, _ := vault.Canonicalize(w.root, e.Name)
					if vault.Classify(canon) != vault.ClassExcluded {
						w.fsWatcher.Add(e.Name)
					}
				}
			}

			evt, ok := filter(e, w.root, w.log)
			if !ok {
				w.dropped++
				continue
			}

			select {
			case w.events <- evt:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// Close fecha o fsnotify e limpa handles.
func (w *Watcher) Close() error {
	return w.fsWatcher.Close()
}
