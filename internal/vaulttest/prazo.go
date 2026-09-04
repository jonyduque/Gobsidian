package vaulttest

import "time"

// Prazo e o unico limite de espera dos testes que aguardam algo assincrono
// (socket, handshake, desligamento). Um defeito real nao pode travar
// "go test -race ./..." ate o timeout de 10 minutos; e tres pacotes com tres
// valores (2 s, 3 s, 5 s) eram tres respostas para a mesma pergunta.
const Prazo = 5 * time.Second
