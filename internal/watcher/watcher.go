package watcher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/search"
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
	inv       *search.Inverted
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
func New(v *vault.Vault, idx *index.Index, inv *search.Inverted, debounce time.Duration, log *slog.Logger) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("criando fsnotify.Watcher: %w", err)
	}

	root := v.Root()

	if err := fsWatcher.Add(root); err != nil {
		_ = fsWatcher.Close()
		return nil, fmt.Errorf("observando raiz %q: %w", root, err)
	}

	// fsnotify v1.10.1 no Windows não é recursivo. Adiciona subdiretorios dinamicamente.
	errWalk := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			log.Warn("erro ao acessar diretório na varredura inicial", "path", path, "err", err)
			return nil
		}
		if d.IsDir() {
			if vault.IsExcludedDir(d.Name()) {
				return filepath.SkipDir
			}
			if path != root {
				if err := fsWatcher.Add(path); err != nil {
					log.Warn("falha ao adicionar watch em subdiretório", "path", path, "err", err)
				}
			}
		}
		return nil
	})
	if errWalk != nil {
		_ = fsWatcher.Close()
		return nil, fmt.Errorf("varrendo subdiretórios de %q: %w", root, errWalk)
	}

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
		inv:       inv,
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
		Apply(ctx, w.debounced, w.reconcile, w.idx, w.v, w.log, &w.processed, &w.skipped, &w.reconciledUpdated, &w.reconciledRemoved, w.inv)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return nil
			}
			w.handleFSError(err)
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
						if err := w.fsWatcher.Add(e.Name); err != nil {
							w.log.Warn("falha ao adicionar watch em novo subdiretório", "path", e.Name, "err", err)
						}
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

// handleFSError trata um erro vindo do fsnotify. Overflow agenda reconciliação;
// o resto vira log.
//
// Existe como método, e não inline no select de Run, para que o teste de
// overflow possa exercitar ESTE código. Antes, TestRun_OverflowSchedulesExactlyOne
// injetava `fsnotify.ErrEventOverflow` direto em `w.fsWatcher.Errors` e mantinha
// uma cópia deste corpo dentro do próprio teste — de modo que ele afirmava sobre
// a reimplementação, não sobre a produção. Escrever no canal do fsnotify também
// corria com o backend kqueue, e o detector de corrida reprovava em macOS.
func (w *Watcher) handleFSError(err error) {
	if errors.Is(err, fsnotify.ErrEventOverflow) {
		w.reconciliations.Add(1)
		select {
		case w.reconcile <- struct{}{}:
			w.log.Warn("Overflow de fsnotify detectado, reconciliação agendada")
		default:
			w.log.Warn("Overflow de fsnotify detectado, reconciliação já em andamento")
		}
		return
	}
	w.log.Error("Erro do fsnotify", "err", err)
}

// Close fecha o fsnotify e limpa handles.
func (w *Watcher) Close() error {
	return w.fsWatcher.Close()
}
