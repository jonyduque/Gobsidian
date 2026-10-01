package main

import (
	"os"
	"testing"

	"github.com/jonyduque/Gobsidian/internal/ipc"
	"github.com/jonyduque/Gobsidian/internal/vazamentotest"
)

// TestMain isola o diretorio de runtime da suite: servePonte registra presenca
// e disca o socket do cofre, e o arquivo `serve.<pid>.presenca` caia no
// %LocalAppData% do usuario a cada rodada. Ver ipc.RodarComRuntimeIsolado.
//
// E reprova o pacote se sobrar goroutine vazada (vazamentotest), com UMA
// exceção: o watchStdin do lifecycle. Medido em 2026-10-01, teste a teste: as 7
// que sobram vêm de testes que passam pelas saídas antecipadas de
// serveEmProcesso -- Montar falhou, ou prepararProcesso mandou sair --, que
// retornam sem fechar o espelho do stdin. Em produção as duas são seguidas do
// fim do processo, que é a saída documentada daquela goroutine (ver
// lifecycle.watchStdin); só o binário de teste sobrevive a elas.
func TestMain(m *testing.M) {
	os.Exit(vazamentotest.Conferir(ipc.RodarComRuntimeIsolado(m), "lifecycle.(*Lifecycle).watchStdin"))
}
