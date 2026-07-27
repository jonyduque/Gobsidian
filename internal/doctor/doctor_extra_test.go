package doctor_test

// Testes adicionais aos dois exigidos pela Task 10, cobrindo comportamentos
// que a brief pede para confirmar mas cujos testes nao escreve: ExitCode com
// slice vazio, propagacao de cancelamento de contexto para as verificacoes
// que varrem o cofre, limpeza do arquivo temporario da verificacao de
// escrita, e a medida correta da verificacao de comprimento de caminho.

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/doctor"
)

// TestStatusMarkerDistinctPerStatus fixa os tres marcadores em valores
// distintos. Sem este teste, uma regressao que faca StatusWarn e StatusFail
// devolverem a mesma string (o bug original desta revisao) passaria
// silenciosamente: nenhum outro teste compara os tres valores de Marker()
// entre si, so verifica presenca de substring em Detail/Name.
func TestStatusMarkerDistinctPerStatus(t *testing.T) {
	ok := doctor.StatusOK.Marker()
	warn := doctor.StatusWarn.Marker()
	fail := doctor.StatusFail.Marker()

	if ok == warn || ok == fail || warn == fail {
		t.Fatalf("marcadores devem ser distintos: OK=%q Warn=%q Fail=%q", ok, warn, fail)
	}
	if ok != "[OK]" {
		t.Errorf("StatusOK.Marker() = %q, esperava [OK]", ok)
	}
	if warn != "[*]" {
		t.Errorf("StatusWarn.Marker() = %q, esperava [*]", warn)
	}
	if fail != "[!]" {
		t.Errorf("StatusFail.Marker() = %q, esperava [!]", fail)
	}
}

// TestCheckCacheDirCreatable confirma o ramo real de sucesso: um CacheDir
// nao-vazio que pode ser criado reporta [OK]. Sem este teste, todo teste do
// pacote monta cfg via config.Defaults(), que deixa CacheDir vazio e so
// exercita o retorno defensivo antecipado — os ramos que de fato criam o
// diretorio nunca rodam.
func TestCheckCacheDirCreatable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "A.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := config.Defaults()
	cfg.VaultPath = root
	cfg.CacheDir = filepath.Join(t.TempDir(), "cache", "nested")

	results := doctor.Run(context.Background(), cfg)

	if !hasStatus(results, "diretorio de cache", doctor.StatusOK) {
		t.Errorf("esperava [OK] para diretorio de cache criavel: %+v", results)
	}
	if _, err := os.Stat(cfg.CacheDir); err != nil {
		t.Errorf("diretorio de cache deveria ter sido criado: %v", err)
	}
}

// TestCheckCacheDirUncreatable confirma o ramo de aviso: um CacheDir cujo
// caminho tem um arquivo regular como ancestral nao pode ser criado por
// MkdirAll em nenhum sistema operacional, e o check deve reportar aviso (nao
// falha bloqueante nem panico).
func TestCheckCacheDirUncreatable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "A.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	blocker := filepath.Join(t.TempDir(), "nao-e-diretorio")
	if err := os.WriteFile(blocker, []byte("bloqueia MkdirAll"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := config.Defaults()
	cfg.VaultPath = root
	cfg.CacheDir = filepath.Join(blocker, "cache")

	results := doctor.Run(context.Background(), cfg)

	if !hasStatus(results, "diretorio de cache", doctor.StatusWarn) {
		t.Errorf("esperava aviso para diretorio de cache nao-criavel: %+v", results)
	}
	if doctor.ExitCode(results) != 0 {
		t.Errorf("diretorio de cache nao-criavel nao deveria ser falha bloqueante: %+v", results)
	}
}

func TestExitCodeEmptyResults(t *testing.T) {
	if code := doctor.ExitCode(nil); code != 0 {
		t.Errorf("ExitCode(nil) = %d, esperava 0", code)
	}
	if code := doctor.ExitCode([]doctor.Result{}); code != 0 {
		t.Errorf("ExitCode([]Result{}) = %d, esperava 0", code)
	}
}

// TestRunContextCancelledStopsWalk confirma que um contexto ja cancelado
// impede as verificacoes que varrem o cofre de completar a varredura. Sem
// isso, um cofre grande sincronizado na nuvem faria doctor travar em vez de
// respeitar o cancelamento do chamador.
//
// ExitCode continuar zero aqui e proposital, nao uma omissao: cancelamento
// veio de quem chamou doctor (Ctrl-C, ou o processo de cima desistindo), nao
// e um fato sobre o ambiente, e nao deve gatear um script de setup. Contraste
// com TestScanStatusDistinguishesCancelamentoDeFalha, em
// checks_internal_test.go, onde o mesmo scan.err nao-nulo vem de vault.Walk
// falhando na propria raiz — cofre desmontado, nao cancelamento — e por isso
// precisa continuar codigo de saida nao-zero. A forma de integracao (derrubar
// o cofre de verdade no meio de um Run) foi rejeitada no Step 5 da Task R1:
// no Windows, ACEs de Full Control herdadas sobrepoem o deny explicito,
// entao tornar um diretorio ilegivel para o proprio processo nao e encenavel
// sem privilegio elevado.
func TestRunContextCancelledStopsWalk(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"A.md", "B.md", "C.md", "D.md", "E.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("# nota\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	cfg := config.Defaults()
	cfg.VaultPath = root

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelado antes mesmo de Run comecar

	results := doctor.Run(ctx, cfg)

	if doctor.ExitCode(results) != 0 {
		t.Errorf("cancelamento de contexto nao deveria virar falha bloqueante: %+v", results)
	}

	found := false
	for _, r := range results {
		if r.Name != "contagem de notas" {
			continue
		}
		found = true
		if !strings.Contains(r.Detail, "interrompida") {
			t.Errorf("contagem de notas deveria reportar varredura interrompida com contexto cancelado, recebi: %+v", r)
		}
		if strings.Contains(r.Detail, "5 notas") {
			t.Errorf("contagem de notas completou a varredura apesar do contexto cancelado: %+v", r)
		}
	}
	if !found {
		t.Fatal("verificacao de contagem de notas nao encontrada nos resultados")
	}
}

// TestCheckWritableNoLeftoverTempFile confirma que a verificacao de escrita
// nao deixa nenhum arquivo temporario para tras na raiz do cofre.
func TestCheckWritableNoLeftoverTempFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "A.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := config.Defaults()
	cfg.VaultPath = root

	_ = doctor.Run(context.Background(), cfg)

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Name()), "gobsidian-doctor") {
			t.Errorf("arquivo temporario da verificacao de escrita nao foi limpo: %s", e.Name())
		}
	}
}

// TestCheckLongestPathMeasuresAbsolutePath confirma que a verificacao de
// comprimento de caminho mede o caminho absoluto completo (o que um syscall
// receberia), nao apenas a porcao relativa a raiz do cofre — caso contrario
// o limiar de 240 caracteres, que existe por causa do MAX_PATH absoluto do
// Windows, nunca dispararia para um cofre com nomes de arquivo curtos mas
// localizado em um caminho profundo.
func TestCheckLongestPathMeasuresAbsolutePath(t *testing.T) {
	root := t.TempDir()
	const name = "A.md"
	if err := os.WriteFile(filepath.Join(root, name), []byte("# A\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := config.Defaults()
	cfg.VaultPath = root

	results := doctor.Run(context.Background(), cfg)

	wantLen := len(filepath.Join(root, name))
	relLen := len(name)

	for _, r := range results {
		if r.Name != "comprimento de caminho" {
			continue
		}
		wantSubstr := strconv.Itoa(wantLen)
		if !strings.Contains(r.Detail, wantSubstr) {
			t.Errorf("esperava o comprimento do caminho absoluto (%d) no detalhe, recebi %q", wantLen, r.Detail)
		}
		relSubstr := strconv.Itoa(relLen)
		if wantSubstr != relSubstr && strings.Contains(r.Detail, relSubstr) && !strings.Contains(r.Detail, wantSubstr) {
			t.Errorf("detalhe parece medir o caminho relativo (%d), nao o absoluto (%d): %q", relLen, wantLen, r.Detail)
		}
		return
	}
	t.Fatal("verificacao de comprimento de caminho nao encontrada nos resultados")
}
