package main

import (
	"testing"

	"github.com/jonyduque/Gobsidian/internal/instalar"
)

// TestAvisoDeDuplicidadeContaSoGravadores e G8.2.
//
// Em 2026-09-14 o aviso contava toda presenca de um cofre e mandava encerrar
// os extras: pontes do Antigravity, que nao gravam nada, entravam na conta.
func TestAvisoDeDuplicidadeContaSoGravadores(t *testing.T) {
	const cofre = `C:\Cofre`
	p := func(pid int, papel, modo string) instalar.Presenca {
		return instalar.Presenca{PID: pid, Cofre: cofre, Papel: papel, Modo: modo}
	}

	t.Run("daemon e tres pontes nao sao duplicidade", func(t *testing.T) {
		s := unicaSituacao(t, analisarGravadores([]instalar.Presenca{
			p(1, "daemon", instalar.ModoDaemon),
			p(2, "serve", instalar.ModoPonte),
			p(3, "serve", instalar.ModoPonte),
			p(4, "serve", instalar.ModoPonte),
		}))
		if len(s.Gravadores) != 1 {
			t.Fatalf("gravadores = %+v, esperado so o daemon", s.Gravadores)
		}
	})

	t.Run("daemon e um em processo sao dois gravadores, com os pids", func(t *testing.T) {
		s := unicaSituacao(t, analisarGravadores([]instalar.Presenca{
			p(1, "daemon", instalar.ModoDaemon),
			p(2, "serve", instalar.ModoPonte),
			p(5, "serve", instalar.ModoEmProcesso),
		}))
		if len(s.Gravadores) != 2 || s.Gravadores[0].PID != 1 || s.Gravadores[1].PID != 5 {
			t.Fatalf("gravadores = %+v, esperado pids 1 e 5", s.Gravadores)
		}
	})

	t.Run("processo sem modo nao conta como gravador, mas aparece", func(t *testing.T) {
		s := unicaSituacao(t, analisarGravadores([]instalar.Presenca{
			p(1, "daemon", instalar.ModoDaemon),
			p(7, "serve", ""),
		}))
		if len(s.Gravadores) != 1 {
			t.Fatalf("gravadores = %+v, esperado so o daemon", s.Gravadores)
		}
		if len(s.SemModo) != 1 || s.SemModo[0].PID != 7 {
			t.Fatalf("sem modo = %+v, esperado o pid 7", s.SemModo)
		}
	})
}

func unicaSituacao(t *testing.T, s []situacaoDoCofre) situacaoDoCofre {
	t.Helper()
	if len(s) != 1 {
		t.Fatalf("situacoes = %+v, esperado exatamente um cofre", s)
	}
	return s[0]
}
