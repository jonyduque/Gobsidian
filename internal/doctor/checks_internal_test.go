package doctor

// Testes internos (package doctor, nao doctor_test) porque scanStatus nao e
// exportado. E a rota preferida da Task R1 para testar a distincao entre
// cancelamento de contexto e cofre inacessivel: TestRunFlagsMissingVault (em
// doctor_test.go) ja documenta que tornar um diretorio ilegivel neste
// ambiente Windows nao funciona sem privilegio (ACLs herdadas de Full
// Control sobrescrevem a negacao), entao um cenario de integracao fiel para
// "raiz existe e e listavel mas a varredura falha" nao tem forma robusta e
// sem privilegio aqui. Testar scanStatus diretamente e deterministico e nao
// depende de manipular permissoes do sistema de arquivos.

import (
	"context"
	"errors"
	"testing"
)

// TestScanStatusDistinguishesCancelamentoDeFalha confirma o ponto central da
// Task R1: cancelamento de contexto vira aviso (nao gateia o exit code), mas
// qualquer outro erro de varredura — o que vault.Walk produz quando falha na
// propria raiz, ou seja cofre desmontado, share caido, pasta movida pelo
// cliente de nuvem — vira falha bloqueante. Antes desta mudanca os dois
// desfechos colapsavam no mesmo StatusWarn, e um cofre inacessivel fazia
// doctor sair 0.
func TestScanStatusDistinguishesCancelamentoDeFalha(t *testing.T) {
	const name = "verificacao de teste"

	t.Run("sem erro nao produz Result", func(t *testing.T) {
		res, failed := scanStatus(vaultScan{}, name)
		if failed {
			t.Errorf("scanStatus com scan.err nulo nao deveria reportar falha, obteve %+v", res)
		}
	})

	t.Run("context.Canceled produz aviso", func(t *testing.T) {
		res, failed := scanStatus(vaultScan{err: context.Canceled}, name)
		if !failed {
			t.Fatal("scanStatus com context.Canceled deveria reportar houve erro")
		}
		if res.Status != StatusWarn {
			t.Errorf("context.Canceled deveria produzir StatusWarn, obteve %v (%+v)", res.Status, res)
		}
	})

	t.Run("context.DeadlineExceeded produz aviso", func(t *testing.T) {
		res, failed := scanStatus(vaultScan{err: context.DeadlineExceeded}, name)
		if !failed {
			t.Fatal("scanStatus com context.DeadlineExceeded deveria reportar houve erro")
		}
		if res.Status != StatusWarn {
			t.Errorf("context.DeadlineExceeded deveria produzir StatusWarn, obteve %v (%+v)", res.Status, res)
		}
	})

	t.Run("erro de varredura na raiz produz falha bloqueante", func(t *testing.T) {
		// Formato real que vault.Walk produz quando falha em d == nil na
		// propria raiz — ver internal/vault/walk.go.
		scanErr := errors.New(`varrendo a raiz do cofre "C:\\cofre": acesso negado`)

		res, failed := scanStatus(vaultScan{err: scanErr}, name)
		if !failed {
			t.Fatal("scanStatus com erro de raiz deveria reportar houve erro")
		}
		if res.Status != StatusFail {
			t.Errorf("erro de raiz deveria produzir StatusFail (cofre inacessivel nao pode virar aviso), obteve %v (%+v)", res.Status, res)
		}
		if res.Name != name {
			t.Errorf("Result.Name = %q, esperava %q", res.Name, name)
		}
	})
}
