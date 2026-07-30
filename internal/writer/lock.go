// Package writer implementa operacoes de escrita e modificacao de notas
// no cofre com suporte a escritas atomicas, locks por caminho e preservacao de EOL/BOM.
package writer

import (
	"strings"
	"sync"

	"github.com/jonyd/gobsidian/internal/vault"
)

// normalizeKey converte o caminho para minusculas para garantir sensibilidade a casing uniforme (Windows).
func normalizeKey(p vault.CanonicalPath) string {
	return strings.ToLower(string(p))
}

type lockEntry struct {
	mu       sync.Mutex
	refCount int
}

// PathLocker gerencia travas de escrita por caminho de nota.
type PathLocker struct {
	mu    sync.Mutex
	locks map[string]*lockEntry
}

// NewPathLocker cria uma nova instancia de PathLocker.
func NewPathLocker() *PathLocker {
	return &PathLocker{
		locks: make(map[string]*lockEntry),
	}
}

// Lock adquire a trava de escrita para o caminho fornecido.
// Retorna uma funcao de desbloqueio (unlock) que deve ser chamada para liberar a trava.
func (l *PathLocker) Lock(path vault.CanonicalPath) func() {
	key := normalizeKey(path)

	l.mu.Lock()
	entry, ok := l.locks[key]
	if !ok {
		entry = &lockEntry{}
		l.locks[key] = entry
	}
	entry.refCount++
	l.mu.Unlock()

	entry.mu.Lock()

	return func() {
		entry.mu.Unlock()

		l.mu.Lock()
		entry.refCount--
		if entry.refCount == 0 {
			delete(l.locks, key)
		}
		l.mu.Unlock()
	}
}

// Count retorna o numero de travas ativas armazenadas no registro.
func (l *PathLocker) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.locks)
}
