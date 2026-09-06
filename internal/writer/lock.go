// Package writer implementa operacoes de escrita e modificacao de notas
// no cofre com suporte a escritas atomicas, locks por caminho e preservacao de EOL/BOM.
package writer

import (
	"sync"

	"github.com/jonyd/gobsidian/internal/text"
	"github.com/jonyd/gobsidian/internal/vault"
)

// normalizeKey e a chave do registro de travas.
//
// Delega a text.ChaveDeCaminho, que e a mesma conta que o indice usa para
// lowerPath. Aqui ela so baixava a caixa: o indice tratava as duas grafias
// Unicode do mesmo nome como UMA nota e o locker as tratava como dois arquivos,
// entao duas escritas concorrentes na mesma nota pegavam travas diferentes e
// nao se excluiam. Uma conta por regra, inclusive nos pontos que ja pareciam
// certos.
func normalizeKey(p vault.CanonicalPath) string {
	return text.ChaveDeCaminho(string(p))
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
