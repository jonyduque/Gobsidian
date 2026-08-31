package ipc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/ipc"
)

// TestHandshakeRecusaMaxResultsDivergente é o achado M9.
//
// A flag `--max-results` da ponte era no-op silencioso no modo daemon: não ia
// para o spawn e não entrava no handshake, então valia o `cfg` do PRIMEIRO
// daemon e a segunda ponte era atendida com um teto que ninguém pediu. É a
// mesma classe de `ReadOnlySet`/`DebounceMSSet` — parâmetro aceito que não faz
// efeito, e o cliente não tem como saber.
//
// O handshake é unidirecional, então a ponte não pode IMPOR o dela. O desenho
// segue o precedente do `ReadOnly`: o daemon anuncia o seu, e a ponte que quer
// outro RECUSA e cai para o modo em processo, onde a configuração dela vale.
// Divergência visível é melhor que silêncio.
func TestHandshakeRecusaMaxResultsDivergente(t *testing.T) {
	vault := t.TempDir()

	ln, _, err := ipc.Listen(vault)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		// O daemon subiu com teto 50; a ponte abaixo quer 200.
		saudacao := ipc.HandshakeConfig{VaultKey: config.VaultKey(vault), MaxResults: 50}
		if err := ipc.Greet(c, saudacao); err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), boundedWait)
	defer cancel()

	conn, err := ipc.DialAndHandshake(ctx, vault, false, 200, boundedWait)
	if err == nil {
		_ = conn.Close()
		t.Fatal("handshake ACEITOU um daemon com teto diferente: a flag da ponte viraria no-op silencioso")
	}
	if !errors.Is(err, ipc.ErrConfigMismatch) {
		t.Errorf("err = %v, queria ErrConfigMismatch", err)
	}
}

// TestHandshakeAceitaMaxResultsIgual é o contrapeso, e sem ele "corrigir" o M9
// recusando toda conexão passaria no teste acima.
func TestHandshakeAceitaMaxResultsIgual(t *testing.T) {
	vault := t.TempDir()

	ln, _, err := ipc.Listen(vault)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		saudacao := ipc.HandshakeConfig{VaultKey: config.VaultKey(vault), MaxResults: 200}
		if err := ipc.Greet(c, saudacao); err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), boundedWait)
	defer cancel()

	conn, err := ipc.DialAndHandshake(ctx, vault, false, 200, boundedWait)
	if err != nil {
		t.Fatalf("handshake com o MESMO teto foi recusado: %v", err)
	}
	_ = conn.Close()
}
