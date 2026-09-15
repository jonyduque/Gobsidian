package instalar

import "testing"

func TestSemPresencaTiraRegistradosEOProprio(t *testing.T) {
	processos := []ProcessoDoSistema{{PID: 10}, {PID: 20}, {PID: 30}}
	vivos := []Presenca{{PID: 20, Papel: "serve", Modo: ModoPonte}}

	sem := SemPresenca(processos, vivos, 30)
	if len(sem) != 1 || sem[0].PID != 10 {
		t.Fatalf("SemPresenca() = %+v, esperado so o pid 10 (20 tem presenca, 30 e o proprio)", sem)
	}
}

func TestGravaCacheSoDaemonEEmProcesso(t *testing.T) {
	casos := map[string]bool{
		ModoDaemon:     true,
		ModoEmProcesso: true,
		ModoPonte:      false,
		"":             false,
	}
	for modo, quer := range casos {
		if got := GravaCache(modo); got != quer {
			t.Errorf("GravaCache(%q) = %v, esperado %v", modo, got, quer)
		}
	}
}
