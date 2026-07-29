package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

// Watcher escuta mudancas no cofre e emite eventos filtrados.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	root      string
	log       *slog.Logger
	events    chan Event
	debounced chan []vault.CanonicalPath
	debounce  time.Duration
	v         *vault.Vault
	idx       *index.Index
	reconcile chan struct{}

	// Contadores (Task 32 e Task 37)
	active              atomic.Bool
	received            atomic.Int64
	droppedChmod        atomic.Int64
	droppedOutsideVault atomic.Int64
	droppedExcluded     atomic.Int64
	droppedUnknownOp    atomic.Int64
	coalesced           atomic.Int64
	processed           atomic.Int64
	skipped             atomic.Int64
	reconciliations     atomic.Int64
	reconciledUpdated   atomic.Int64
	reconciledRemoved   atomic.Int64
}

// New cria um novo Watcher observando a raiz do cofre.
func New(v *vault.Vault, idx *index.Index, debounce time.Duration, log *slog.Logger) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("criando fsnotify.Watcher: %w", err)
	}

	root := v.Root()

	if err := fsWatcher.Add(root); err != nil {
		fsWatcher.Close()
		return nil, fmt.Errorf("observando raiz %q: %w", root, err)
	}

	// fsnotify v1.10.1 no Windows não é recursivo. Adiciona subdiretorios dinamicamente.
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if vault.IsExcludedDir(d.Name()) {
				return filepath.SkipDir
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
		debounced: make(chan []vault.CanonicalPath, 100),
		reconcile: make(chan struct{}, 1),
		debounce:  debounce,
		v:         v,
		idx:       idx,
	}, nil
}

// Run entra em loop processando eventos do fsnotify ate o contexto ser cancelado
// ou o watcher ser fechado.
func (w *Watcher) Run(ctx context.Context) error {
	w.active.Store(true)
	defer w.active.Store(false)

	defer close(w.events)
	// Lança o debouncer lendo de w.events e escrevendo em w.debounced
	go func() {
		Debounce(ctx, w.events, w.debounced, w.debounce, w.log, &w.coalesced)
		close(w.debounced)
	}()

	// Lança o aplicador lendo de w.debounced e escrevendo no índice
	go func() {
		Apply(ctx, w.debounced, w.reconcile, w.idx, w.v, w.log, &w.processed, &w.skipped, &w.reconciledUpdated, &w.reconciledRemoved)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return nil
			}
			if err == fsnotify.ErrEventOverflow || (err != nil && err.Error() == fsnotify.ErrEventOverflow.Error()) {
				w.reconciliations.Add(1)
				select {
				case w.reconcile <- struct{}{}:
					w.log.Warn("Overflow de fsnotify detectado, reconciliação agendada")
				default:
					w.log.Warn("Overflow de fsnotify detectado, reconciliação já em andamento")
				}
				continue
			}
			w.log.Error("Erro do fsnotify", "err", err)
		case e, ok := <-w.fsWatcher.Events:
			if !ok {
				return nil
			}
			w.received.Add(1)

			// Se um diretorio novo for criado, adiciona o watch nele
			if e.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(e.Name)
				if err == nil && info.IsDir() {
					if !vault.IsExcludedDir(info.Name()) {
						w.fsWatcher.Add(e.Name)
					}
				}
			}

			evt, ok, reason := filter(e, w.root, w.log)
			if !ok {
				switch reason {
				case DropChmod:
					w.droppedChmod.Add(1)
				case DropOutsideVault:
					w.droppedOutsideVault.Add(1)
				case DropExcluded:
					w.droppedExcluded.Add(1)
				case DropUnknownOp:
					w.droppedUnknownOp.Add(1)
				}
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
