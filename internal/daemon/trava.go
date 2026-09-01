package daemon

import (
	"fmt"
	"os"
	"path/filepath"
)

// travaDeArquivo e a exclusao mutua entre processos deste pacote: uma trava do
// KERNEL sobre um arquivo, `flock` no Unix e `LockFileEx` no Windows.
//
// Ela substituiu, em 2026-08-31, um esquema de arquivo-com-PID: criar o lock
// com O_EXCL, gravar o PID dentro, e quem encontrasse o arquivo ocupado lia o
// PID, perguntava se aquele processo ainda vivia e, se nao, REMOVIA o lock e
// tomava o lugar. O esquema tinha corridas que tres rodadas de medicao nao
// conseguiram fechar — 55% de reprovacao no teste de dez daemons concorrentes,
// caindo para 20% depois de duas correcoes, com a causa do resto ainda sem
// nome.
//
// A raiz e que todo aquele mecanismo imita, mal, o que o kernel ja faz: soltar
// a trava quando o dono morre. Com a trava do kernel nao existe arquivo
// obsoleto, nem PID a parsear, nem sondagem de vitalidade, nem caminho de
// recuperacao — e portanto nao existe a classe de corrida inteira. A correcao
// nao e um remendo mais esperto; e apagar o que precisava de remendo.
//
// O arquivo NUNCA e removido, e isso e deliberado. Remover era a origem de toda
// corrida: entre o remover e o recriar, qualquer um entrava. Um arquivo de zero
// byte que sobra no diretorio de runtime nao custa nada e nao bloqueia ninguem
// — quem chega depois abre o MESMO arquivo e pede a trava dele.
type travaDeArquivo struct {
	f *os.File
}

// Liberar solta a trava e fecha o descritor.
//
// Fechar sozinho ja soltaria no Unix, mas soltar explicitamente mantem as duas
// plataformas com o mesmo texto — e no Windows a ordem importa.
func (t *travaDeArquivo) Liberar() {
	if t == nil || t.f == nil {
		return
	}
	_ = destravarArquivo(t.f.Fd())
	_ = t.f.Close()
}

// tentarTravar abre (ou cria) path e pede a trava exclusiva sem bloquear.
//
// Devolve (nil, false, nil) quando outro processo a detem: perder a disputa
// nao e erro, e a decisao deste projeto e que quem perde SAI, em vez de dormir
// esperando.
func tentarTravar(path string) (*travaDeArquivo, bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, false, fmt.Errorf("criando diretorio do lock: %w", err)
	}

	// Sem O_EXCL, de proposito: o arquivo persistir entre execucoes e o
	// comportamento desejado. Quem decide a posse e a trava, nao a existencia.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, fmt.Errorf("abrindo lock %s: %w", path, err)
	}

	tomou, err := travarArquivo(f.Fd())
	if err != nil {
		_ = f.Close()
		return nil, false, fmt.Errorf("travando lock %s: %w", path, err)
	}
	if !tomou {
		_ = f.Close()
		return nil, false, nil
	}

	// O PID vai para dentro do arquivo DEPOIS da trava, e serve so para
	// diagnostico — `doctor` mostra quem detem. Nenhuma decisao de exclusao
	// depende dele, que e precisamente o que mudou: antes, um PID ilegivel
	// autorizava outro processo a roubar o lock.
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	_ = f.Sync()

	return &travaDeArquivo{f: f}, true, nil
}

// TravaEmUso diz se algum processo detem a trava do arquivo em path.
//
// E a pergunta que `doctor` faz, e ela substituiu "o PID escrito la dentro
// ainda vive?". A diferenca nao e de estilo: o PID respondia sobre um PROCESSO,
// e a pergunta util e sobre a TRAVA. Um lock cujo dono morreu tinha PID morto e
// continuava, para todos os efeitos, ocupando o caminho; hoje ele simplesmente
// nao esta travado, e nao ha estado obsoleto a diagnosticar.
//
// Adquirir e soltar na hora e seguro: se a trava estava livre, ninguem a
// esperava; se estava tomada, nao a tocamos.
func TravaEmUso(path string) (bool, error) {
	trava, tomou, err := tentarTravar(path)
	if err != nil {
		return false, err
	}
	if !tomou {
		return true, nil
	}
	trava.Liberar()
	return false, nil
}
